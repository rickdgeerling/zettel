package cmd

import (
	"github.com/spf13/cobra"
	"github.com/user/zettel-mcp/internal/mcp"
	"github.com/user/zettel-mcp/internal/store"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server over stdio",
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.DefaultStore()
		if err != nil {
			panic("Failed to initialize store: " + err.Error())
		}

		mcpServer := mcp.NewZettelServer(s)
		if err := mcpServer.Run(); err != nil {
			panic("MCP server error: " + err.Error())
		}
	},
}
