// ABOUTME: MCP server initialization and configuration
// ABOUTME: Sets up server with tools, resources, and prompts

package mcp

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps MCP server with database connection.
type Server struct {
	mcp *mcp.Server
	db  *sql.DB
}

// NewServer creates MCP server with all capabilities.
func NewServer(db *sql.DB) (*Server, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

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
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}
