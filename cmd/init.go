package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rickdgeerling/zettel/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a .zettel store in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		if flagStore != "" {
			fmt.Fprintf(os.Stderr, "Error: --store flag is not supported with init; cd to the target directory and run 'zettel init'\n")
			os.Exit(1)
		}
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot determine current directory: %v\n", err)
			os.Exit(1)
		}
		target := filepath.Join(cwd, ".zettel")

		if info, err := os.Stat(target); err == nil {
			if info.IsDir() {
				fmt.Fprintf(os.Stderr, "Error: .zettel store already exists in %s\n", cwd)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s exists but is not a directory\n", target)
			}
			os.Exit(1)
		}

		s, err := store.Init(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing store: %v\n", err)
			os.Exit(1)
		}

		indexCard := &store.Card{
			Slug:    "index",
			Title:   "Index",
			Created: time.Now().UTC(),
			Body:    "# Index\n",
		}
		if err := s.WriteCard("index", indexCard, "init"); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating index card: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Initialized empty zettel store at %s\n", target)
	},
}
