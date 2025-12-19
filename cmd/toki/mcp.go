// ABOUTME: MCP server command implementation
// ABOUTME: Starts toki MCP server in stdio mode for AI agent integration

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/harper/toki/internal/charm"
	"github.com/harper/toki/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio mode)",
	Long: `Start the Model Context Protocol server for AI agent integration.

The MCP server communicates via stdio, allowing AI agents like Claude
to interact with your toki tasks through a standardized protocol.

This command will run continuously until interrupted (Ctrl+C).`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Charm client is initialized by root command's PersistentPreRunE
	// and available via charm.GetClient()
	client := charm.GetClient()
	if client == nil {
		return fmt.Errorf("charm client not initialized")
	}

	// Create MCP server with Charm client
	server, err := mcp.NewServer(client)
	if err != nil {
		return err
	}

	// Start server in stdio mode
	return server.Serve(ctx)
}
