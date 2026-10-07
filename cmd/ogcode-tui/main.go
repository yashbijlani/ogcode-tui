// Command ogcode-tui is the standalone entrypoint for the ogcode terminal UI.
//
// It is the same binary as ogcode — the TUI runs in-process, driving the agent
// loop through the same managers the `ogcode run`/`ogcode prompt` path uses —
// but its default action is the TUI rather than the server, so a bare
// `ogcode-tui` opens the chat with no arguments. `ogcode tui` remains the
// equivalent subcommand of the main binary.
package main

import (
	"os"

	"github.com/prasenjeet-symon/ogcode/internal/cli"
)

func main() {
	if err := cli.ExecuteTUI(); err != nil {
		os.Exit(1)
	}
}
