package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/prasenjeet-symon/ogcode/internal/permission"
	"github.com/prasenjeet-symon/ogcode/internal/provider"
	"github.com/prasenjeet-symon/ogcode/internal/question"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// A modal owns the keyboard while it is open.
	if a.modal != modalNone {
		return a.handleModalKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		if a.quitArmed {
			return a, tea.Quit
		}
		a.quitArmed = true
		a.setStatus("press ctrl+c again to quit", false)
		return a, nil
	case "ctrl+p":
		a.openModelPicker()
		return a, nil
	case "ctrl+s":
		a.openSessionPicker()
		return a, nil
	case "ctrl+h", "f1", "?":
		a.modal = modalHelp
		return a, nil
	case "shift+tab":
		a.cyclePermission()
		return a, nil
	case "esc":
		if a.running {
			a.abortRun()
		}
		return a, nil
	case "enter":
		if a.running {
			// A running turn takes the composer text as mid-loop guidance.
			text := strings.TrimSpace(a.composer.Value())
			if text != "" {
				a.sendGuidance(text)
				a.composer.Reset()
				a.composer.SetHeight(a.composerHeight())
			}
			return a, nil
		}
		return a.sendPrompt()
	case "up":
		if strings.TrimSpace(a.composer.Value()) == "" {
			a.viewport.LineUp(1)
			return a, nil
		}
	case "down":
		if strings.TrimSpace(a.composer.Value()) == "" {
			a.viewport.LineDown(1)
			return a, nil
		}
	case "pgup":
		a.viewport.PageUp()
		return a, nil
	case "pgdown":
		a.viewport.PageDown()
		return a, nil
	}

	// Anything else edits the composer.
	var cmd tea.Cmd
	a.composer, cmd = a.composer.Update(msg)
	if h := a.composerHeight(); h != a.composer.Height() {
		a.resize()
	}
	a.quitArmed = false
	return a, cmd
}

func (a *App) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.modal {
	case modalHelp:
		if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "?" {
			a.modal = modalNone
		}
		return a, nil

	case modalPermission:
		switch msg.String() {
		case "esc", "q":
			a.replyPermission("reject")
		case "up", "k":
			a.permModal.cursor = (a.permModal.cursor - 1 + len(permissionChoices)) % len(permissionChoices)
		case "down", "j":
			a.permModal.cursor = (a.permModal.cursor + 1) % len(permissionChoices)
		case "left", "h":
			a.permModal.cursor = (a.permModal.cursor - 1 + len(permissionChoices)) % len(permissionChoices)
		case "right", "l":
			a.permModal.cursor = (a.permModal.cursor + 1) % len(permissionChoices)
		case "1":
			a.replyPermission("once")
		case "2":
			a.replyPermission("always")
		case "3":
			a.replyPermission("reject")
		case "enter":
			a.replyPermission(permissionChoices[a.permModal.cursor].response)
		}
		return a, nil

	case modalQuestion:
		return a.handleQuestionKey(msg)

	case modalModel, modalSessions:
		switch msg.String() {
		case "esc", "q":
			a.modal = modalNone
		case "up", "k":
			a.pickerMove(-1)
		case "down", "j":
			a.pickerMove(1)
		case "enter":
			a.commitPicker()
		}
		return a, nil
	}
	return a, nil
}

func (a *App) handleQuestionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	q := a.queryModal.current()
	nOpts := len(q.Options)

	switch msg.String() {
	case "esc":
		a.cancelQuestion()
		return a, nil
	case "tab", "shift+tab":
		// Cycle between the options and the free-text field.
		if nOpts == 0 {
			return a, nil
		}
		if a.queryModal.cursor >= nOpts {
			a.queryModal.cursor = 0
		} else {
			a.queryModal.cursor = nOpts
		}
		a.queryModal.typing = a.queryModal.cursor >= nOpts
		return a, nil
	case "up":
		if a.queryModal.cursor > 0 {
			a.queryModal.cursor--
		}
		a.queryModal.typing = a.queryModal.cursor >= nOpts
		return a, nil
	case "down":
		if a.queryModal.cursor < nOpts {
			a.queryModal.cursor++
		}
		a.queryModal.typing = a.queryModal.cursor >= nOpts
		return a, nil
	case " ":
		if a.queryModal.typing || a.queryModal.cursor >= nOpts {
			a.queryModal.text += " "
			return a, nil
		}
		a.toggleOption()
		return a, nil
	case "enter":
		a.commitQuestion()
		return a, nil
	case "backspace":
		if a.queryModal.typing {
			r := []rune(a.queryModal.text)
			if len(r) > 0 {
				a.queryModal.text = string(r[:len(r)-1])
			}
			return a, nil
		}
	}

	if a.queryModal.typing {
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			a.queryModal.text += msg.String()
		}
		return a, nil
	}

	// Number keys select an option quickly.
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '9' {
		idx := int(msg.Runes[0] - '1')
		if idx < nOpts {
			a.queryModal.cursor = idx
			a.toggleOption()
		}
	}
	return a, nil
}

func (a *App) toggleOption() {
	q := a.queryModal.current()
	idx := a.queryModal.cursor
	if idx < 0 || idx >= len(q.Options) {
		return
	}
	label := q.Options[idx].Label
	cur := a.queryModal.answers[a.queryModal.idx].Selected
	if q.MultiSelect {
		if contains(cur, label) {
			a.queryModal.answers[a.queryModal.idx].Selected = remove(cur, label)
		} else {
			a.queryModal.answers[a.queryModal.idx].Selected = append(cur, label)
		}
	} else {
		a.queryModal.answers[a.queryModal.idx].Selected = []string{label}
	}
}

func remove(xs []string, s string) []string {
	out := xs[:0:0]
	for _, x := range xs {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}

// commitQuestion stores the current screen's answer and advances to the next
// visible screen, submitting the batch when none remain.
func (a *App) commitQuestion() {
	if a.queryModal.text != "" {
		a.queryModal.answers[a.queryModal.idx].Text = a.queryModal.text
	}
	a.queryModal.text = ""
	a.queryModal.typing = false
	a.queryModal.cursor = 0
	a.queryModal.idx++
	a.advanceQuestion()
}

func (a *App) advanceQuestion() {
	for a.queryModal.idx < len(a.queryModal.req.Questions) {
		if a.queryModal.visible(a.queryModal.idx) {
			return
		}
		a.queryModal.idx++
	}
	a.submitQuestion()
}

func (a *App) submitQuestion() {
	req := a.queryModal.req
	reply := question.Reply{Answers: a.queryModal.answers}
	ok := a.questions.Reply(req.ID, reply)
	a.modal = modalNone
	if !ok {
		a.setStatus("question expired", true)
	}
	a.buses.Publish("question.replied", map[string]string{
		"sessionId":  string(a.sess.ID),
		"questionId": string(req.ID),
		"response":   "answered",
	})
}

func (a *App) cancelQuestion() {
	req := a.queryModal.req
	// Cancel by answering nothing: the model reads an empty answer as "not
	// answered", and the loop carries on.
	a.questions.Reply(req.ID, question.Reply{Answers: make([]question.Answer, len(req.Questions))})
	a.modal = modalNone
	a.buses.Publish("question.replied", map[string]string{
		"sessionId":  string(a.sess.ID),
		"questionId": string(req.ID),
		"response":   "cancelled",
	})
}

func (a *App) replyPermission(response string) {
	req := a.permModal.req
	ok := a.perms.Reply(req.ID, response)
	a.modal = modalNone
	if !ok {
		a.setStatus("permission expired", true)
	}
	a.buses.Publish("permission.replied", map[string]string{
		"sessionId":    string(a.sess.ID),
		"permissionId": string(req.ID),
		"response":     response,
	})
}

func (a *App) commitPicker() {
	if a.picker.cursor < 0 || a.picker.cursor >= len(a.picker.items) {
		a.modal = modalNone
		return
	}
	it := a.picker.items[a.picker.cursor]
	switch a.picker.mode {
	case modalModel:
		a.selectModel(it.model)
	case modalSessions:
		if it.kind == "new" {
			a.modal = modalNone
			a.newSession()
			return
		}
		a.modal = modalNone
		a.switchSession(it.sess)
	}
}

func (a *App) selectModel(m provider.ModelInfo) {
	providerID := m.ProviderID
	if a.sess != nil {
		a.sess.Model = m.ID
		a.sess.Provider = providerID
		a.sess.UpdatedAt = session.Now()
		if err := a.store.Update(a.sess); err != nil {
			a.setStatus("save model: "+err.Error(), true)
			return
		}
		a.buses.Publish("session.updated", a.sess)
	}
	a.modal = modalNone
	a.setStatus("model set to "+m.ID, false)
}

func (a *App) cyclePermission() {
	if a.sess == nil {
		return
	}
	modes := []string{permission.ModeAsk, permission.ModeAuto, permission.ModeYolo}
	cur := a.sess.Permission
	next := modes[0]
	for i, m := range modes {
		if m == cur {
			next = modes[(i+1)%len(modes)]
			break
		}
	}
	a.sess.Permission = next
	a.sess.UpdatedAt = session.Now()
	if err := a.perms.SetDefaultMode(next); err != nil {
		a.setStatus("save permission mode: "+err.Error(), true)
	}
	if err := a.store.Update(a.sess); err != nil {
		a.setStatus("save session: "+err.Error(), true)
		return
	}
	a.buses.Publish("session.updated", a.sess)
	a.setStatus("permission mode: "+next, false)
}
