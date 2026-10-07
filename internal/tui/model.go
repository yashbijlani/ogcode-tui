package tui

import (
	"context"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/prasenjeet-symon/ogcode/internal/agent"
	"github.com/prasenjeet-symon/ogcode/internal/bus"
	"github.com/prasenjeet-symon/ogcode/internal/id"
	"github.com/prasenjeet-symon/ogcode/internal/permission"
	"github.com/prasenjeet-symon/ogcode/internal/provider"
	"github.com/prasenjeet-symon/ogcode/internal/question"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

const (
	headerLines   = 1
	helpLines     = 1
	composerMin   = 1
	composerMax   = 6
	maxTranscript = 400
)

// App is the root bubbletea model. It owns the in-process ogcode session: the
// store, the bus subscription, the permission/question managers and the agent
// loop runner. Model mutation only ever happens inside Update.
type App struct {
	// services wired in by NewApp.
	store     *session.Store
	buses     *bus.Bus
	perms     *permission.Manager
	questions *question.Manager
	registry  *provider.Registry
	lr        *agent.LoopRunner
	dir       string

	// bus subscription → tea program.
	events <-chan bus.Event
	prog   *tea.Program

	sess *session.Session

	composer textarea.Model
	viewport viewport.Model

	running bool
	working bool
	dirty   bool
	ticks   int

	mu       sync.Mutex
	cancel   context.CancelFunc
	lc       *agent.LoopControl
	token    int
	runToken int
	runID    id.SessionID

	modal      modalKind
	permModal  permissionModal
	queryModal questionModal
	picker     picker

	transcript string
	status     string
	statusErr  bool
	quitArmed  bool

	width  int
	height int
	ready  bool
}

// NewApp builds the TUI over an already-wired session. The heavy lifting
// (providers, tools, runner) is done by the caller so this package keeps no
// dependency on the CLI's flag parsing.
func NewApp(store *session.Store, b *bus.Bus, perms *permission.Manager, questions *question.Manager, registry *provider.Registry, lr *agent.LoopRunner, dir string, sess *session.Session) *App {
	ta := textarea.New()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.Placeholder = "Message ogcode…  (enter to send · alt+enter newline)"
	ta.Prompt = ""
	ta.FocusedStyle.Placeholder = styleFaint
	// Plain enter sends; alt+enter (or ctrl+j) inserts a newline.
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("alt+enter", "ctrl+j"))
	ta.Focus()

	return &App{
		store:     store,
		buses:     b,
		perms:     perms,
		questions: questions,
		registry:  registry,
		lr:        lr,
		dir:       dir,
		sess:      sess,
		composer:  ta,
		viewport:  viewport.New(80, 20),
	}
}

func (a *App) Init() tea.Cmd {
	a.dirty = true
	return tea.Batch(textarea.Blink, a.armTick())
}

func (a *App) armTick() tea.Cmd { return tickCmd() }

func (a *App) ID() id.SessionID { return a.sess.ID }

// Run starts the program and pumps bus events into it until it exits.
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	a.prog = p
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Subscribe before p.Run: pumpEvents takes the channel by value and Init
	// runs inside p.Run, so subscribing there would hand the pump a nil channel
	// and silently drop every bus event — the streaming reply and loop.done
	// included, leaving the transcript unbuilt and the header stuck "working".
	a.events = a.buses.SubscribeAll()
	go pumpEvents(a.events, p.Send, ctx.Done())
	_, err := p.Run()
	cancel()
	return err
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.resize()
		a.ready = true
		a.rebuildTranscript()
		a.syncViewport()
		return a, nil

	case tickMsg:
		a.ticks++
		if a.dirty {
			a.dirty = false
			a.rebuildTranscript()
			a.syncViewport()
		}
		return a, a.armTick()

	case busMsg:
		return a.handleBus(msg.evt)

	case errMsg:
		a.setStatus(errStyle(msg.err), true)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

// resize recomputes the transcript viewport and composer dimensions from the
// current terminal size.
func (a *App) resize() {
	if a.width <= 0 || a.height <= 0 {
		return
	}
	composerH := a.composer.Height()
	if composerH < composerMin {
		composerH = composerMin
	}
	// Reserve header, help, the composer, and the two framing rules around the
	// composer box (top/bottom borders + a blank spacer line).
	bodyH := a.height - headerLines - helpLines - composerH - 4
	if bodyH < 3 {
		bodyH = 3
	}
	a.viewport.Width = a.width
	a.viewport.Height = bodyH
	a.composer.SetWidth(a.width - 4)
	a.composer.SetHeight(composerH)
}

func (a *App) composerHeight() int {
	h := a.composer.LineCount()
	if h < composerMin {
		h = composerMin
	}
	if h > composerMax {
		h = composerMax
	}
	return h
}

// rebuildTranscript re-reads the session's messages from the store and renders
// them into the cached transcript string.
func (a *App) rebuildTranscript() {
	if a.sess == nil || a.store == nil {
		return
	}
	msgs, err := a.store.GetMessages(a.sess.ID, "", maxTranscript)
	if err != nil {
		a.setStatus("load transcript: "+err.Error(), true)
		return
	}
	a.transcript = renderMessages(msgs, a.viewport.Width)
}

// syncViewport pushes the transcript into the viewport, following the tail
// unless the user has scrolled up.
func (a *App) syncViewport() {
	atBottom := a.viewport.AtBottom()
	a.viewport.SetContent(a.transcript)
	if atBottom || a.running {
		a.viewport.GotoBottom()
	}
}

func (a *App) setStatus(s string, isErr bool) {
	a.status = s
	a.statusErr = isErr
}

func errStyle(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (a *App) handleBus(evt bus.Event) (tea.Model, tea.Cmd) {
	switch evt.Type {
	case "message.updated", "message.part.updated":
		a.dirty = true
	case "permission.requested":
		var p struct {
			PermissionID string `json:"permissionId"`
		}
		_ = jsonUnmarshal(evt.Properties, &p)
		if reqs := a.perms.PendingForSession(string(a.sess.ID)); len(reqs) > 0 {
			// Show the newest pending request for this session.
			for _, r := range reqs {
				if p.PermissionID == "" || string(r.ID) == p.PermissionID {
					a.permModal = newPermissionModal(r)
					a.modal = modalPermission
					break
				}
			}
		}
	case "question.requested":
		var req question.Request
		if jsonUnmarshal(evt.Properties, &req) == nil && req.ID != "" {
			a.queryModal = newQuestionModal(req)
			a.queryModal.cursor = 0
			a.modal = modalQuestion
		}
	case "permission.replied", "question.replied":
		// The modal that prompted this is already closing (or has closed).
	case "session.updated":
		var sess session.Session
		if jsonUnmarshal(evt.Properties, &sess) == nil && sess.ID == a.sess.ID {
			a.sess = &sess
		}
	case "loop.done":
		a.running = false
		a.dirty = true
		var p struct {
			Reason string `json:"reason"`
			Error  string `json:"error"`
		}
		_ = jsonUnmarshal(evt.Properties, &p)
		if p.Error != "" {
			a.setStatus("loop error: "+p.Error, true)
		} else if p.Reason == "aborted" {
			a.setStatus("aborted", false)
		} else {
			a.setStatus("", false)
		}
	}
	return a, a.armTick()
}
