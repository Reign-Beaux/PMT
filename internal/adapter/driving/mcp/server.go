package mcp

import (
	"github.com/mark3labs/mcp-go/server"

	"project-management-tools/internal/domain/shared"

	commentapp "project-management-tools/internal/application/comment"
	issueapp "project-management-tools/internal/application/issue"
	labelapp "project-management-tools/internal/application/label"
	phaseapp "project-management-tools/internal/application/phase"
	projectapp "project-management-tools/internal/application/project"
)

const (
	serverName    = "pmt"
	serverVersion = "0.1.0"
)

// Server is the MCP driving adapter. It wraps all application services
// and exposes them as MCP tools via stdio transport.
type Server struct {
	mcpServer *server.MCPServer
	ownerID   shared.ID
	projects  *projectapp.Service
	phases    *phaseapp.Service
	issues    *issueapp.Service
	labels    *labelapp.Service
	comments  *commentapp.Service
}

// NewServer creates a new MCP server with all application services.
func NewServer(
	ownerID shared.ID,
	projects *projectapp.Service,
	phases *phaseapp.Service,
	issues *issueapp.Service,
	labels *labelapp.Service,
	comments *commentapp.Service,
) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer(serverName, serverVersion),
		ownerID:   ownerID,
		projects:  projects,
		phases:    phases,
		issues:    issues,
		labels:    labels,
		comments:  comments,
	}

	s.registerTools()

	return s
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools registers all MCP tools with the server.
func (s *Server) registerTools() {
	s.registerProjectTools()
	s.registerPhaseTools()
	s.registerIssueTools()
	s.registerLabelTools()
	s.registerCommentTools()
	s.registerCompoundTools()
}
