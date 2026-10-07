package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prasenjeet-symon/ogcode/internal/session"
)

// renderMessages turns a page of stored messages into the transcript body.
func renderMessages(msgs []*session.MessageWithParts, width int) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(renderMessage(m, width))
		b.WriteString("\n\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderMessage(m *session.MessageWithParts, width int) string {
	var b strings.Builder
	switch m.Info.Role {
	case session.RoleUser:
		if m.Info.DisplayOnly {
			b.WriteString(styleMuted.Render("↳ guidance"))
		} else {
			b.WriteString(styleUser.Render("❯ you"))
		}
	default:
		label := "assistant"
		if m.Info.Model != "" {
			label = m.Info.Model
		}
		icon := "●"
		if m.Info.Error != nil && *m.Info.Error != "" {
			icon = "✖"
			label = *m.Info.Error
			b.WriteString(styleErr.Render(icon + " " + label))
			b.WriteString("\n")
			break
		}
		b.WriteString(styleAssistant.Render(icon + " " + label))
		if m.Info.Finish != nil && *m.Info.Finish == "aborted" {
			b.WriteString(styleMuted.Render("  (aborted)"))
		}
	}
	b.WriteString("\n")

	textWidth := width - 4
	if textWidth < 20 {
		textWidth = 20
	}
	for _, p := range m.Parts {
		switch p.Type {
		case session.PartText:
			var td session.TextPartData
			if json.Unmarshal(p.Data, &td) == nil && strings.TrimSpace(td.Text) != "" {
				b.WriteString(indent(wrapText(strings.TrimRight(td.Text, "\n"), textWidth), "  "))
				b.WriteString("\n")
			}
		case session.PartReasoning:
			var rd session.ReasoningPartData
			if json.Unmarshal(p.Data, &rd) == nil && strings.TrimSpace(rd.Text) != "" {
				b.WriteString(styleFaint.Render(indent(firstLines(rd.Text, 6), "  ")))
				b.WriteString("\n")
			}
		case session.PartTool:
			b.WriteString(renderToolPart(p, textWidth))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderToolPart(p session.Part, width int) string {
	var td session.ToolPartData
	if json.Unmarshal(p.Data, &td) != nil {
		return ""
	}
	icon := "⏺"
	st := styleTool
	switch td.State.Status {
	case session.ToolRunning:
		icon = "◐"
	case session.ToolCompleted:
		icon = "✔"
		st = styleOK
	case session.ToolError:
		icon = "✖"
		st = styleErr
	case session.ToolDenied:
		icon = "⊘"
		st = styleErr
	}
	title := td.Tool
	if td.State.Title != nil && *td.State.Title != "" {
		title = *td.State.Title
	}
	line := fmt.Sprintf("%s %s", icon, title)
	if arg := toolArg(td.State.Input); arg != "" {
		line += styleMuted.Render(" " + arg)
	}
	var b strings.Builder
	b.WriteString("  " + st.Render(line) + "\n")
	if td.State.Output != nil && strings.TrimSpace(*td.State.Output) != "" {
		b.WriteString(styleFaint.Render(indent(wrapText(firstLines(*td.State.Output, 10), width), "    ")) + "\n")
	}
	if td.State.Error != nil && strings.TrimSpace(*td.State.Error) != "" {
		b.WriteString(styleErr.Render(indent(firstLines(*td.State.Error, 6), "    ")) + "\n")
	}
	return b.String()
}

// toolArg pulls a short, human-readable hint out of a tool call's JSON input —
// the command for bash, the path for the file tools — and falls back to a
// truncated raw snippet.
func toolArg(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return truncate(string(raw), 60)
	}
	for _, key := range []string{"command", "pattern", "path", "filePath", "file_path", "url", "query", "prompt", "description"} {
		if v, ok := m[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil && s != "" {
				return truncate(oneLine(s), 72)
			}
		}
	}
	return ""
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:n], "\n") + fmt.Sprintf("\n… (+%d lines)", len(lines)-n)
}

// wrapText word-wraps each line of s to width columns. It is byte-based, which
// is good enough for a terminal transcript and keeps long prose from scrolling
// off the right edge.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		if strings.TrimSpace(para) == "" {
			out = append(out, "")
			continue
		}
		line := ""
		for _, w := range strings.Fields(para) {
			switch {
			case line == "":
				line = w
			case len(line)+1+len(w) <= width:
				line += " " + w
			default:
				out = append(out, line)
				line = w
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
