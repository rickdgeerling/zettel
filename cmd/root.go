package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/zettel-mcp/internal/store"
)

var storeInstance *store.Store

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(writeCmd)
	rootCmd.AddCommand(linksCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(serveCmd)
}

var rootCmd = &cobra.Command{
	Use:   "zettel",
	Short: "Zettelkasten memory system for AI agents",
	Long:  `A CLI and MCP server for managing Markdown-based knowledge cards with YAML frontmatter.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		s, err := store.DefaultStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize store: %v\n", err)
			os.Exit(1)
		}
		storeInstance = s
	},
}

func GetStore() *store.Store {
	return storeInstance
}
