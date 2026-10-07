package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/prasenjeet-symon/ogcode/internal/permission"
)

func (a *App) View() string {
	if !a.ready || a.width <= 0 {
		return "starting ogcode…"
	}
	header := a.renderHeader()
	body := a.viewport.View()
	composer := a.renderComposer()
	help := a.renderHelp()

	base := lipgloss.JoinVertical(lipgloss.Left, header, body, composer, help)

	if a.modal != modalNone {
		return a.renderModal(base)
	}
	return base
}

func (a *App) renderHeader() string {
	title := ""
	model := ""
	mode := ""
	if a.sess != nil {
		title = a.sess.Title
		model = a.sess.Model
		mode = a.sess.Permission
	}
	if model == "" {
		model = "(default model)"
	}
	left := styleTitle.Render("ogcode") + styleMuted.Render("  "+truncate(title, 40))
	right := styleMuted.Render("model: ") + styleLabel.Render(model)
	if mode == "" {
		mode = permission.ModeAsk
	}
	right += styleMuted.Render("   mode: ") + modeStyle(mode).Render(mode)
	if a.working || a.running {
		right += " " + styleAccent.Render("● working")
	}
	return spread(left, right, a.width)
}

func modeStyle(mode string) lipgloss.Style {
	switch mode {
	case permission.ModeYolo:
		return lipgloss.NewStyle().Bold(true).Foreground(colErr)
	case permission.ModeAuto:
		return lipgloss.NewStyle().Bold(true).Foreground(colOK)
	default:
		return lipgloss.NewStyle().Bold(true).Foreground(colTool)
	}
}

func (a *App) renderComposer() string {
	inner := a.composer.View()
	box := styleBox.Width(a.width - 2)
	return box.Render(inner)
}

func (a *App) renderHelp() string {
	keys := [][2]string{
		{"enter", "send"},
		{"esc", "abort/close"},
		{"ctrl+p", "model"},
		{"ctrl+s", "sessions"},
		{"shift+tab", "perm mode"},
		{"ctrl+c", "quit"},
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, styleHelpKey.Render(k[0])+styleHelpDesc.Render(" "+k[1]))
	}
	left := strings.Join(parts, styleFaint.Render("  ·  "))

	if a.status != "" {
		st := styleMuted
		if a.statusErr {
			st = styleErr
		}
		if a.quitArmed {
			left = styleErr.Render("press ctrl+c again to quit")
		}
		return spread(left, st.Render(truncate(a.status, a.width/2)), a.width)
	}
	if a.quitArmed {
		return styleErr.Render("press ctrl+c again to quit")
	}
	return left
}

func (a *App) renderModal(base string) string {
	var title, body string
	var footer string
	switch a.modal {
	case modalPermission:
		title = "permission"
		body = a.permModal.render(a.width)
	case modalQuestion:
		title = "ask_user"
		body = a.queryModal.render(a.width)
	case modalModel:
		title = "select model"
		body = a.picker.render()
	case modalSessions:
		title = "sessions"
		body = a.picker.render()
	case modalHelp:
		title = "help"
		body = helpBody()
	}
	_ = footer
	box := styleModalBox.Width(min(a.width-4, 92)).Render(styleTitle.Render(title) + "\n\n" + body)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "))
}

func helpBody() string {
	lines := []string{
		"enter         send message (or newline when composing multi-line with alt+enter)",
		"alt+enter     insert a newline in the composer",
		"↑ / ↓         scroll the transcript (when the composer is empty)",
		"esc           abort the running turn, or close an open dialog",
		"ctrl+p        choose model / provider",
		"ctrl+s        session list (switch or start a new session)",
		"shift+tab     cycle permission mode: ask → auto → yolo",
		"tab           in a dialog, move between options and the text field",
		"space         toggle a choice in a multi-select question",
		"ctrl+c        quit (press twice)",
	}
	return strings.Join(lines, "\n")
}

func spread(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
