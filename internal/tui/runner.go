package tui

import (
	"context"
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/prasenjeet-symon/ogcode/internal/agent"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

// sendPrompt persists the composer text as a user message and starts a turn.
// It is the TUI's equivalent of the server's handlePrompt: the same store
// writes, the same orphan repair, and startRun for the loop itself.
func (a *App) sendPrompt() (tea.Model, tea.Cmd) {
	content := strings.TrimSpace(a.composer.Value())
	if content == "" {
		return a, nil
	}
	if a.sess == nil {
		a.setStatus("no session", true)
		return a, nil
	}
	a.composer.Reset()
	a.composer.SetHeight(a.composerHeight())

	userMsg := &session.MessageInfo{
		ID:        session.NewMessageID(),
		SessionID: a.sess.ID,
		Role:      session.RoleUser,
		Agent:     a.agentName(),
		CreatedAt: session.Now(),
	}
	if err := a.store.CreateMessage(userMsg); err != nil {
		a.setStatus("create message: "+err.Error(), true)
		return a, nil
	}
	textData, _ := json.Marshal(session.TextPartData{Text: content})
	if err := a.store.CreatePart(&session.Part{
		ID:        session.NewPartID(),
		MessageID: userMsg.ID,
		SessionID: a.sess.ID,
		Type:      session.PartText,
		Data:      textData,
		CreatedAt: session.Now(),
		UpdatedAt: session.Now(),
	}); err != nil {
		a.setStatus("create part: "+err.Error(), true)
		return a, nil
	}
	a.buses.Publish("message.updated", userMsg)

	// If the session still carries the default title, name it after the first
	// prompt so the project's session list shows something meaningful instead of
	// a wall of identical "New session" rows. Mirrors the one-shot CLI in run.go.
	if a.sess.Title == "New session" {
		title := oneLine(content)
		if len(title) > 60 {
			title = title[:60] + "…"
		}
		a.sess.Title = title
		a.sess.UpdatedAt = session.Now()
		if err := a.store.Update(a.sess); err != nil {
			a.setStatus("title session: "+err.Error(), true)
		} else {
			a.buses.Publish("session.updated", a.sess)
		}
	}

	// Repair orphaned assistant messages from a previous crash so the transcript
	// does not show a turn stuck in progress forever.
	aborted := "aborted"
	if orphans, err := a.store.GetMessages(a.sess.ID, "", 100); err == nil {
		for _, m := range orphans {
			if m.Info.Role == session.RoleAssistant && m.Info.Finish == nil && m.Info.Error == nil {
				m.Info.Finish = &aborted
				if a.store.UpdateMessage(&m.Info) == nil {
					a.buses.Publish("message.updated", &m.Info)
				}
			}
		}
	}

	a.running = true
	a.dirty = true
	a.syncViewport()
	a.startRun(a.agentName(), a.viewport.Width, a.viewport.Height)
	return a, nil
}

// sendGuidance folds mid-loop guidance into the running turn. The text goes to
// the loop control (delivered to the model on its next step) and a display-only
// message records it in the transcript.
func (a *App) sendGuidance(text string) {
	a.mu.Lock()
	lc := a.lc
	running := a.running
	a.mu.Unlock()
	if !running || lc == nil {
		return
	}
	a.buses.Publish("loop.guidance", map[string]string{
		"sessionId": string(a.sess.ID),
		"status":    "queued",
	})
	lc.PushGuidance(text)
	a.recordGuidanceMessage(text)
}

func (a *App) recordGuidanceMessage(text string) {
	msg := &session.MessageInfo{
		ID:          session.NewMessageID(),
		SessionID:   a.sess.ID,
		Role:        session.RoleUser,
		DisplayOnly: true,
		CreatedAt:   session.Now(),
	}
	if err := a.store.CreateMessage(msg); err != nil {
		return
	}
	textData, _ := json.Marshal(session.TextPartData{Text: text})
	_ = a.store.CreatePart(&session.Part{
		ID:        session.NewPartID(),
		MessageID: msg.ID,
		SessionID: a.sess.ID,
		Type:      session.PartText,
		Data:      textData,
		CreatedAt: session.Now(),
		UpdatedAt: session.Now(),
	})
	a.buses.Publish("message.updated", msg)
	a.dirty = true
}

// abortRun cancels the running loop and marks the unfinished tool calls and
// assistant message exactly as the server's abort handler does.
func (a *App) abortRun() {
	a.mu.Lock()
	cancel := a.cancel
	a.cancel = nil
	a.lc = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	a.running = false
	a.setStatus("aborted", false)

	msgs, err := a.store.GetMessages(a.sess.ID, "", 100)
	if err != nil {
		return
	}
	aborted := "aborted"
	cancelledMsg := "Request cancelled by user"
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Info.Role == session.RoleAssistant && m.Info.Finish == nil && m.Info.Error == nil {
			m.Info.Finish = &aborted
			if a.store.UpdateMessage(&m.Info) == nil {
				a.buses.Publish("message.updated", &m.Info)
			}
		}
		for _, p := range m.Parts {
			if p.Type != session.PartTool {
				continue
			}
			var td session.ToolPartData
			if json.Unmarshal(p.Data, &td) != nil {
				continue
			}
			if td.State.Status != session.ToolPending && td.State.Status != session.ToolRunning {
				continue
			}
			td.State.Status = session.ToolError
			td.State.Error = &cancelledMsg
			td.State.Time.End = session.Now()
			data, _ := json.Marshal(td)
			p.Data = data
			p.UpdatedAt = session.Now()
			if a.store.UpdatePart(&p) == nil {
				a.buses.Publish("message.part.updated", map[string]string{
					"sessionId": string(a.sess.ID),
					"partId":    string(p.ID),
				})
			}
		}
	}
	a.dirty = true
}

// startRun launches lr.RunLoop on its own goroutine with the same context,
// gating and bookkeeping the server's startSessionLoop installs.
func (a *App) startRun(agentName string, vw, vh int) {
	ctx, cancel := context.WithCancel(context.Background())
	lc := agent.NewLoopControl()
	ctx = agent.WithLoopControl(ctx, lc)
	ctx = agent.WithPermissionGating(ctx)

	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.token++
	token := a.token
	a.cancel = cancel
	a.lc = lc
	a.runToken = token
	a.runID = a.sess.ID
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			if a.runToken == token {
				a.cancel = nil
				a.lc = nil
			}
			a.mu.Unlock()
		}()
		_ = a.lr.RunLoop(ctx, a.runID, agentName, vw, vh)
	}()
}

func (a *App) agentName() string {
	if a.sess != nil && a.sess.SessionType != "" {
		return a.sess.SessionType
	}
	return "build"
}

// newSession creates a fresh session in the current project and switches to it.
func (a *App) newSession() {
	sess := &session.Session{
		ID:          session.NewSessionID(),
		ProjectID:   a.dir,
		Directory:   a.dir,
		Title:       "New session",
		SessionType: "build",
		Permission:  a.perms.DefaultMode(),
		CreatedAt:   session.Now(),
		UpdatedAt:   session.Now(),
	}
	if err := a.store.Create(sess); err != nil {
		a.setStatus("create session: "+err.Error(), true)
		return
	}
	a.buses.Publish("session.created", sess)
	a.loadSession(sess)
}

// switchSession makes an existing session current.
func (a *App) switchSession(sess *session.Session) {
	if sess == nil {
		return
	}
	a.loadSession(sess)
}

func (a *App) loadSession(sess *session.Session) {
	a.sess = sess
	a.running = false
	a.modal = modalNone
	a.dirty = true
	a.rebuildTranscript()
	a.syncViewport()
	a.setStatus("session: "+truncate(sess.Title, 40), false)
}
