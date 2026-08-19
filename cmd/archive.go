package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var archiveDryRun bool

var archiveCmd = &cobra.Command{
	Use:   "archive <slug>",
	Short: "Move a card to the archive",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]
		if archiveDryRun {
			fmt.Printf("Would archive card %q (dry run)\n", slug)
			return
		}
		err := GetStore().ArchiveCard(slug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Card %q archived successfully\n", slug)
	},
}

func init() {
	archiveCmd.Flags().BoolVar(&archiveDryRun, "dry-run", false, "Show what would be archived without changing anything")
}
