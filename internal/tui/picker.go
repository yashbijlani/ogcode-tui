package tui

import (
	"fmt"
	"strings"

	"github.com/prasenjeet-symon/ogcode/internal/provider"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

// picker is the shared scrollable list behind the model chooser and the session
// list. mode records which of the two it is so enter does the right thing.
type picker struct {
	mode   modalKind
	items  []pickerItem
	cursor int
}

type pickerItem struct {
	label string
	desc  string
	model provider.ModelInfo
	sess  *session.Session
	kind  string // "new" for the leading new-session entry
}

const pickerWindow = 12

func (a *App) openModelPicker() {
	models := a.registry.ListModels()
	items := make([]pickerItem, 0, len(models))
	for _, m := range models {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		label := name
		if a.sess != nil && (m.ID == a.sess.Model && (a.sess.Provider == "" || m.ProviderID == a.sess.Provider)) {
			label += styleOK.Render("  ✓")
		}
		desc := m.ProviderID
		if m.ContextWindow > 0 {
			desc += fmt.Sprintf(" · %dk ctx", m.ContextWindow/1000)
		}
		if m.InputPricePerM > 0 || m.OutputPricePerM > 0 {
			desc += fmt.Sprintf(" · $%.2f/$%.2f per M", m.InputPricePerM, m.OutputPricePerM)
		}
		items = append(items, pickerItem{label: label, desc: desc, model: m})
	}
	a.picker = picker{mode: modalModel, items: items}
	// Start on the current model, if listed.
	if a.sess != nil {
		for i, it := range items {
			if it.model.ID == a.sess.Model {
				a.picker.cursor = i
				break
			}
		}
	}
	a.modal = modalModel
}

func (a *App) openSessionPicker() {
	sessions, _ := a.store.ListAll()
	items := make([]pickerItem, 0, len(sessions)+1)
	items = append(items, pickerItem{label: styleAccent.Render("+ New session"), kind: "new"})
	for _, s := range sessions {
		label := s.Title
		if label == "" {
			label = "(untitled)"
		}
		if a.sess != nil && s.ID == a.sess.ID {
			label += styleOK.Render("  ← current")
		}
		desc := fmt.Sprintf("%s · %s", s.SessionType, shortID(string(s.ID)))
		items = append(items, pickerItem{label: label, desc: desc, sess: s})
	}
	a.picker = picker{mode: modalSessions, items: items}
	a.modal = modalSessions
}

func (p picker) render() string {
	if len(p.items) == 0 {
		return styleMuted.Render("(nothing to show)") + "\n\n" + styleMuted.Render("esc close")
	}
	start := 0
	if p.cursor >= pickerWindow {
		start = p.cursor - pickerWindow + 1
	}
	end := start + pickerWindow
	if end > len(p.items) {
		end = len(p.items)
	}

	var b strings.Builder
	if start > 0 {
		b.WriteString(styleFaint.Render(fmt.Sprintf("  ↑ %d more", start)) + "\n")
	}
	for i := start; i < end; i++ {
		it := p.items[i]
		marker := "  "
		if i == p.cursor {
			marker = styleSelected.Render("❯ ")
		}
		line := marker + it.label
		if it.desc != "" {
			line += styleMuted.Render("  " + it.desc)
		}
		b.WriteString(line + "\n")
	}
	if end < len(p.items) {
		b.WriteString(styleFaint.Render(fmt.Sprintf("  ↓ %d more", len(p.items)-end)) + "\n")
	}
	b.WriteString("\n" + styleMuted.Render("↑/↓ move · enter select · esc close"))
	return b.String()
}

func (a *App) pickerMove(delta int) {
	n := len(a.picker.items)
	if n == 0 {
		return
	}
	a.picker.cursor = (a.picker.cursor + delta + n) % n
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
