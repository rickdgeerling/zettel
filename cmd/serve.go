package cmd

import (
	"github.com/spf13/cobra"
	"github.com/user/zettel-mcp/internal/mcp"
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
