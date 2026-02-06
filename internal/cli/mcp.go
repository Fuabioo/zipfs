package cli

import (
	"github.com/Fuabioo/zipfs/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server on stdio",
	Long: `Starts the Model Context Protocol (MCP) server on stdio.

This command is used by MCP clients (Claude Desktop, etc.) to communicate
with zipfs. It should not be run directly by users.`,
	Args: cobra.NoArgs,
	RunE: runMCP,
}

func runMCP(cmd *cobra.Command, args []string) error {
	return mcp.Serve()
}
