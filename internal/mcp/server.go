// ABOUTME: MCP server initialization and configuration
// ABOUTME: Sets up server with tools, resources, and prompts

package mcp

import (
	"context"
	"database/sql"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps MCP server with database connection.
type Server struct {
	mcp *mcp.Server
	db  *sql.DB
}

// NewServer creates MCP server with all capabilities.
func NewServer(db *sql.DB) (*Server, error) {
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "toki",
			Version: "1.0.0",
		},
		nil,
	)

	s := &Server{
		mcp: mcpServer,
		db:  db,
	}

	// Register tools, resources, prompts
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s, nil
}

// Serve starts the MCP server in stdio mode.
func (s *Server) Serve(ctx context.Context) error {
	// TODO: Implement actual stdio transport
	// For now, just block until context is cancelled
	<-ctx.Done()
	return ctx.Err()
}
