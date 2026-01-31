package cli

import (
	"fmt"

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
	// TODO: Wire this up in Wave 4
	// For now, this is a placeholder that indicates MCP mode is starting
	fmt.Println("MCP server starting on stdio...")
	fmt.Println("(MCP implementation will be completed in Wave 4)")
	return nil
}
