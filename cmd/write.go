package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/rickdgeerling/zettel/internal/store"
	"github.com/spf13/cobra"
)

var writeBody string

var writeCmd = &cobra.Command{
	Use:   "write <slug>",
	Short: "Write a card (reads body from stdin or --body flag)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]

		var body string
		if writeBody != "" {
			body = writeBody
		} else if stdinPiped() {
			data, _ := io.ReadAll(os.Stdin)
			body = string(data)
		} else {
			fmt.Fprintf(os.Stderr, "Error: provide card content via stdin or --body flag\n")
			os.Exit(1)
		}

		card, err := store.UnmarshalCard(slug, []byte(body))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing card: %v\n", err)
			os.Exit(1)
		}

		err = GetStore().WriteCard(slug, card, "cli")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Card %q written successfully\n", slug)
	},
}

func stdinPiped() bool {
	info, _ := os.Stdin.Stat()
	return (info.Mode() & os.ModeCharDevice) == 0
}
