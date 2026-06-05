package main

import (
	"os"

	"github.com/user/zettel-mcp/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
