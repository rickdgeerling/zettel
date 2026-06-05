package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <slug>",
	Short: "Read a card by slug",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]
		card, err := GetStore().ReadCard(slug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(card.Marshal())
	},
}
