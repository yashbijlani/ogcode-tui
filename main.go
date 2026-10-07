package main

import (
	"os"

	"github.com/prasenjeet-symon/ogcode/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}