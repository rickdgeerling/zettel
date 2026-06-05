package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var storePathCmd = &cobra.Command{
	Use:   "store-path",
	Short: "Print the resolved zettel store path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(GetStore().Root)
	},
}
