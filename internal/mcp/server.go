// ABOUTME: MCP server initialization and configuration
// ABOUTME: Sets up server with tools, resources, and prompts

package mcp

import (
	"context"
	"fmt"

	"github.com/harper/toki/internal/charm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps MCP server with Charm KV client.
type Server struct {
	mcp    *mcp.Server
	client *charm.Client
}

// NewServer creates MCP server with all capabilities.
func NewServer(client *charm.Client) (*Server, error) {
	if client == nil {
		return nil, fmt.Errorf("charm client is required")
	}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "toki",
			Version: "1.0.0",
		},
		nil,
	)

	s := &Server{
		mcp:    mcpServer,
		client: client,
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
