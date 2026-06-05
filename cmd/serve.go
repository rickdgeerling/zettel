package cmd

import (
	"github.com/rickdgeerling/zettel/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server over stdio",
	Run: func(cmd *cobra.Command, args []string) {
		mcpServer := mcp.NewZettelServer(GetStore())
		if err := mcpServer.Run(); err != nil {
			panic("MCP server error: " + err.Error())
		}
	},
}
