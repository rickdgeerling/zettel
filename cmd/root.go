package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/zettel-mcp/internal/store"
)

var (
	storeInstance *store.Store
	flagStore     string
	flagQuiet     bool
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagStore, "store", "", "Path to zettel store (overrides discovery)")
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "Suppress store path log")

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
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot determine current directory: %v\n", err)
			os.Exit(1)
		}
		s, err := store.ResolveStore(cwd, flagStore, os.Getenv("ZETTEL_HOME"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		storeInstance = s
		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Using zettel store at %s\n", s.Root)
		}
	},
}

func GetStore() *store.Store {
	return storeInstance
}
