package tui

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/prasenjeet-symon/ogcode/internal/bus"
)

// busMsg carries one bus event into the tea update loop. All model mutation
// happens here, never on the publisher's goroutine.
type busMsg struct{ evt bus.Event }

// tickMsg drives the coalesced re-render: streaming produces a burst of
// message.part.updated events, so the transcript is rebuilt from the store on a
// timer rather than once per event.
type tickMsg time.Time

// errMsg surfaces an asynchronous failure in the status line.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// pumpEvents forwards every bus event to the tea program until done closes or
// the subscription channel closes.
func pumpEvents(ch <-chan bus.Event, send func(tea.Msg), done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			select {
			case <-done:
				return
			default:
			}
			send(busMsg{evt: evt})
		}
	}
}

// tickCmd schedules the next coalescing tick.
func tickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// jsonUnmarshal is a thin wrapper so callers in this package read a touch
// cleaner at the bus boundary.
func jsonUnmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
