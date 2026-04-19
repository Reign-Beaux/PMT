package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	phaseapp "project-management-tools/internal/application/phase"
	"project-management-tools/internal/domain/phase"
	"project-management-tools/internal/domain/shared"
)

func (s *Server) registerPhaseTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("list_phases",
			mcp.WithDescription("List all phases in a project, ordered by phase order. Returns a paginated array of phase objects."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the project")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default: 50)")),
			mcp.WithNumber("offset", mcp.Description("Number of results to skip (default: 0)")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleListPhases,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_phase",
			mcp.WithDescription("Get a phase by its ID. Returns the phase object."),
			mcp.WithString("phase_id", mcp.Required(), mcp.Description("UUID of the phase")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleGetPhase,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("create_phase",
			mcp.WithDescription("Create a new phase in a project. Order is assigned automatically. The project must exist."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the parent project")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Phase name (required, non-empty)")),
			mcp.WithString("description", mcp.Description("Phase description (optional)")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleCreatePhase,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("update_phase",
			mcp.WithDescription("Update an existing phase. Only provided fields are changed."),
			mcp.WithString("phase_id", mcp.Required(), mcp.Description("UUID of the phase to update")),
			mcp.WithString("name", mcp.Description("New phase name")),
			mcp.WithString("description", mcp.Description("New description")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleUpdatePhase,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("delete_phase",
			mcp.WithDescription("Delete a phase by its ID. Issues assigned to this phase may become orphaned — consider reassigning them first."),
			mcp.WithString("phase_id", mcp.Required(), mcp.Description("UUID of the phase to delete")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleDeletePhase,
	)
}

func (s *Server) handleListPhases(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, err := shared.ParseID(args["project_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid project_id: %v", err)), nil
	}

	phases, err := s.phases.ListByProject(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list phases: %v", err)), nil
	}

	all := make([]map[string]any, len(phases))
	for i, p := range phases {
		all[i] = marshalPhase(p)
	}
	offset, limit := paginationArgs(args, 50)
	return jsonResult(paginate(all, offset, limit))
}

func (s *Server) handleGetPhase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["phase_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid phase_id: %v", err)), nil
	}

	p, err := s.phases.GetByID(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get phase: %v", err)), nil
	}
	return jsonResult(marshalPhase(p))
}

func (s *Server) handleCreatePhase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, _ := args["project_id"].(string)
	name, _ := args["name"].(string)
	desc, _ := args["description"].(string)

	p, err := s.phases.Create(ctx, phaseapp.CreateInput{
		ProjectID:   projectID,
		Name:        name,
		Description: desc,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create phase: %v", err)), nil
	}
	return jsonResult(marshalPhase(p))
}

func (s *Server) handleUpdatePhase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["phase_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid phase_id: %v", err)), nil
	}

	input := phaseapp.UpdateInput{}

	if v, ok := args["name"].(string); ok {
		input.Name = &v
	}
	if v, ok := args["description"].(string); ok {
		input.Description = &v
	}

	p, err := s.phases.Update(ctx, id, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update phase: %v", err)), nil
	}
	return jsonResult(marshalPhase(p))
}

func (s *Server) handleDeletePhase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["phase_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid phase_id: %v", err)), nil
	}

	if err := s.phases.Delete(ctx, id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete phase: %v", err)), nil
	}
	return mcp.NewToolResultText("phase deleted"), nil
}

func marshalPhase(p phase.Phase) map[string]any {
	return map[string]any{
		"id":          p.ID().String(),
		"project_id":  p.ProjectID().String(),
		"name":        p.Name().String(),
		"description": p.Description(),
		"order":       p.Order().Value(),
		"created_at":  p.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":  p.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
