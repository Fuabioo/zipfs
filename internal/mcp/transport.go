package mcp

// This file provides stdio transport for the MCP server.
// The mcp-go library handles stdio transport natively via server.Serve().
//
// No additional transport layer is needed - the Server.Serve() method
// automatically uses stdio as the transport mechanism.
