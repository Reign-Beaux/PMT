package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	labelapp "project-management-tools/internal/application/label"
	"project-management-tools/internal/domain/label"
	"project-management-tools/internal/domain/shared"
)

func (s *Server) registerLabelTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("list_labels",
			mcp.WithDescription("List all labels in a project. Returns a paginated array of label objects."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the project")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default: 50)")),
			mcp.WithNumber("offset", mcp.Description("Number of results to skip (default: 0)")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleListLabels,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_label",
			mcp.WithDescription("Get a label by its ID."),
			mcp.WithString("label_id", mcp.Required(), mcp.Description("UUID of the label")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleGetLabel,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("create_label",
			mcp.WithDescription("Create a label in a project. Requires name and color (hex format, e.g. #FF0000)."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the parent project")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Label name (required)")),
			mcp.WithString("color", mcp.Required(), mcp.Description("Hex color, e.g. #FF0000 (required)")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleCreateLabel,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("update_label",
			mcp.WithDescription("Update an existing label. Only provided fields are changed."),
			mcp.WithString("label_id", mcp.Required(), mcp.Description("UUID of the label to update")),
			mcp.WithString("name", mcp.Description("New label name")),
			mcp.WithString("color", mcp.Description("New hex color")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleUpdateLabel,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("delete_label",
			mcp.WithDescription("Delete a label by its ID."),
			mcp.WithString("label_id", mcp.Required(), mcp.Description("UUID of the label to delete")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleDeleteLabel,
	)
}

func (s *Server) handleListLabels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, err := shared.ParseID(args["project_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid project_id: %v", err)), nil
	}

	labels, err := s.labels.ListByProject(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list labels: %v", err)), nil
	}

	all := make([]map[string]any, len(labels))
	for i, l := range labels {
		all[i] = marshalLabel(l)
	}
	offset, limit := paginationArgs(args, 50)
	return jsonResult(paginate(all, offset, limit))
}

func (s *Server) handleGetLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["label_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid label_id: %v", err)), nil
	}

	l, err := s.labels.GetByID(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get label: %v", err)), nil
	}
	return jsonResult(marshalLabel(l))
}

func (s *Server) handleCreateLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, _ := args["project_id"].(string)
	name, _ := args["name"].(string)
	color, _ := args["color"].(string)

	l, err := s.labels.Create(ctx, labelapp.CreateInput{
		ProjectID: projectID,
		Name:      name,
		Color:     color,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create label: %v", err)), nil
	}
	return jsonResult(marshalLabel(l))
}

func (s *Server) handleUpdateLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["label_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid label_id: %v", err)), nil
	}

	input := labelapp.UpdateInput{}

	if v, ok := args["name"].(string); ok {
		input.Name = &v
	}
	if v, ok := args["color"].(string); ok {
		input.Color = &v
	}

	l, err := s.labels.Update(ctx, id, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update label: %v", err)), nil
	}
	return jsonResult(marshalLabel(l))
}

func (s *Server) handleDeleteLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["label_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid label_id: %v", err)), nil
	}

	if err := s.labels.Delete(ctx, id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete label: %v", err)), nil
	}
	return mcp.NewToolResultText("label deleted"), nil
}

func marshalLabel(l label.Label) map[string]any {
	return map[string]any{
		"id":         l.ID().String(),
		"project_id": l.ProjectID().String(),
		"name":       l.Name().String(),
		"color":      l.Color().String(),
		"created_at": l.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at": l.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
