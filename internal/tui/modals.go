package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prasenjeet-symon/ogcode/internal/permission"
	"github.com/prasenjeet-symon/ogcode/internal/question"
)

type modalKind int

const (
	modalNone modalKind = iota
	modalPermission
	modalQuestion
	modalModel
	modalSessions
	modalHelp
)

// permissionModal is one tool-approval prompt awaiting an answer.
type permissionModal struct {
	req    permission.Request
	cursor int
}

func newPermissionModal(req permission.Request) permissionModal {
	return permissionModal{req: req}
}

var permissionChoices = []struct {
	label    string
	response string
}{
	{"Allow once", "once"},
	{"Always allow (this exact target)", "always"},
	{"Reject", "reject"},
}

// questionModal is one ask_user batch, walked screen by screen. Options are
// toggled with space, committed with enter; a free-text answer is typed
// directly and committed with enter.
type questionModal struct {
	req      question.Request
	idx      int
	answers  []question.Answer
	cursor   int
	text     string
	typing   bool // free-text field focused (vs. options)
	multi    bool
	finished bool
}

func newQuestionModal(req question.Request) questionModal {
	answers := make([]question.Answer, len(req.Questions))
	return questionModal{
		req:     req,
		answers: answers,
	}
}

func (m questionModal) current() question.Question { return m.req.Questions[m.idx] }

// visible reports whether the current screen is shown given the answers so far.
func (m questionModal) visible(i int) bool {
	answers := question.AnswersByID(m.req.Questions, question.Reply{Answers: m.answers})
	vis := question.Visibility(m.req.Questions, answers)
	if i < len(vis) {
		return vis[i]
	}
	return true
}

func (m questionModal) render(width int) string {
	if m.finished || m.idx >= len(m.req.Questions) {
		return "Finishing…"
	}
	q := m.current()
	w := width
	if w > 84 {
		w = 84
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render(fmt.Sprintf("Question %d of %d", m.idx+1, len(m.req.Questions))) + "\n")
	if q.Header != "" {
		b.WriteString(styleLabel.Render(q.Header) + "\n")
	}
	b.WriteString(wrapText(q.Question, w) + "\n\n")
	for i, opt := range q.Options {
		mark := "  "
		if m.cursor == i {
			mark = styleSelected.Render("❯") + " "
		} else {
			mark = "  "
		}
		if q.MultiSelect {
			if contains(m.answers[m.idx].Selected, opt.Label) {
				mark += styleOK.Render("[x] ")
			} else {
				mark += "[ ] "
			}
		}
		line := mark + opt.Label
		if opt.Description != "" {
			line += styleMuted.Render(" — " + opt.Description)
		}
		b.WriteString(line + "\n")
	}
	// Free-text field is the last row.
	cursorOnText := m.cursor >= len(q.Options)
	textPrompt := "answer"
	if m.typing || cursorOnText {
		textPrompt = styleSelected.Render("answer")
	}
	b.WriteString("\n" + textPrompt + ": ")
	if m.text != "" {
		b.WriteString(m.text)
	} else {
		b.WriteString(styleFaint.Render("(type your own, or pick above)"))
	}
	b.WriteString("\n\n")
	hint := "enter select · "
	if q.MultiSelect {
		hint = "space toggle · enter next · "
	}
	hint += "esc cancel"
	b.WriteString(styleMuted.Render(hint))
	return b.String()
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func (m permissionModal) render(width int) string {
	w := width
	if w > 84 {
		w = 84
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render("Permission required") + "\n\n")
	b.WriteString("The agent wants to run " + styleLabel.Render(m.req.Tool) + ":\n\n")
	detail := m.req.Input
	if detail == "" && len(m.req.Patterns) > 0 {
		detail = strings.Join(m.req.Patterns, ", ")
	}
	b.WriteString(indent(wrapText(prettyInput(detail), w), "  ") + "\n")
	if len(m.req.Patterns) > 0 {
		b.WriteString(styleMuted.Render("  target: "+truncate(oneLine(m.req.Patterns[0]), w)) + "\n")
	}
	b.WriteString("\n")
	for i, c := range permissionChoices {
		if i == m.cursor {
			b.WriteString(styleSelected.Render("❯ "+c.label) + "\n")
		} else {
			b.WriteString("  " + c.label + "\n")
		}
	}
	b.WriteString("\n" + styleMuted.Render("enter select · esc reject"))
	return b.String()
}

// prettyInput renders a tool call's JSON input compactly, falling back to the
// raw string when it does not parse.
func prettyInput(s string) string {
	var v any
	if json.Unmarshal([]byte(s), &v) != nil {
		return s
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	return string(b)
}
