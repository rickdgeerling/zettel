package main

import (
	"os"

	"github.com/rickdgeerling/zettel/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
