package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	projectapp "project-management-tools/internal/application/project"
	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

func (s *Server) registerProjectTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all projects owned by the current user. Returns an array of project objects."),
		),
		s.handleListProjects,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_project",
			mcp.WithDescription("Get a project by its ID. Returns the project object with id, name, description, status, created_at, and updated_at."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the project")),
		),
		s.handleGetProject,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("create_project",
			mcp.WithDescription("Create a new project. The project is owned by the current user. Name must be non-empty."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Project name (required, non-empty)")),
			mcp.WithString("description", mcp.Description("Project description (optional)")),
		),
		s.handleCreateProject,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("update_project",
			mcp.WithDescription("Update an existing project. Only provided fields are changed. Valid status values: active, archived."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the project to update")),
			mcp.WithString("name", mcp.Description("New project name")),
			mcp.WithString("description", mcp.Description("New description")),
			mcp.WithString("status", mcp.Description("New status"), mcp.Enum("active", "archived")),
		),
		s.handleUpdateProject,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("delete_project",
			mcp.WithDescription("Delete a project by its ID. This is irreversible."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the project to delete")),
		),
		s.handleDeleteProject,
	)
}

func (s *Server) handleListProjects(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := s.projects.List(ctx, s.ownerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list projects: %v", err)), nil
	}

	out := make([]map[string]any, len(projects))
	for i, p := range projects {
		out[i] = marshalProject(p)
	}
	return jsonResult(out)
}

func (s *Server) handleGetProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["project_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid project_id: %v", err)), nil
	}

	p, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get project: %v", err)), nil
	}
	return jsonResult(marshalProject(p))
}

func (s *Server) handleCreateProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	name, _ := args["name"].(string)
	desc, _ := args["description"].(string)

	p, err := s.projects.Create(ctx, projectapp.CreateInput{
		OwnerID:     s.ownerID,
		Name:        name,
		Description: desc,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create project: %v", err)), nil
	}
	return jsonResult(marshalProject(p))
}

func (s *Server) handleUpdateProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["project_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid project_id: %v", err)), nil
	}

	input := projectapp.UpdateInput{}

	if v, ok := args["name"].(string); ok {
		input.Name = &v
	}
	if v, ok := args["description"].(string); ok {
		input.Description = &v
	}
	if v, ok := args["status"].(string); ok {
		status, err := project.ParseStatus(v)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid status: %v", err)), nil
		}
		input.Status = &status
	}

	p, err := s.projects.Update(ctx, id, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update project: %v", err)), nil
	}
	return jsonResult(marshalProject(p))
}

func (s *Server) handleDeleteProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["project_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid project_id: %v", err)), nil
	}

	if err := s.projects.Delete(ctx, id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete project: %v", err)), nil
	}
	return mcp.NewToolResultText("project deleted"), nil
}

func marshalProject(p project.Project) map[string]any {
	return map[string]any{
		"id":          p.ID().String(),
		"name":        p.Name().String(),
		"description": p.Description(),
		"status":      string(p.Status()),
		"created_at":  p.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":  p.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// jsonResult serializes v to JSON and returns it as a text tool result.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}
