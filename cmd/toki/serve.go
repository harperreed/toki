// ABOUTME: MCP server command implementation
// ABOUTME: Starts toki MCP server in stdio mode for AI agent integration

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/harper/toki/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server (stdio mode)",
	Long: `Start the Model Context Protocol server for AI agent integration.

The MCP server communicates via stdio, allowing AI agents like Claude
to interact with your toki tasks through a standardized protocol.

This command will run continuously until interrupted (Ctrl+C).`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	// Context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Database is initialized by root command's PersistentPreRunE
	// and available via the global dbConn variable
	if dbConn == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// Create MCP server with database connection
	server, err := mcp.NewServer(dbConn)
	if err != nil {
		return err
	}

	// Start server in stdio mode
	return server.Serve(ctx)
}
