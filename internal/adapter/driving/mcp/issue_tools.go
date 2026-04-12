package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	issueapp "project-management-tools/internal/application/issue"
	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/shared"
)

func (s *Server) registerIssueTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("pmt_list_issues_by_phase",
			mcp.WithDescription("List all issues assigned to a specific phase. Returns a paginated array of issue objects."),
			mcp.WithString("phase_id", mcp.Required(), mcp.Description("UUID of the phase")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default: 50)")),
			mcp.WithNumber("offset", mcp.Description("Number of results to skip (default: 0)")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleListIssuesByPhase,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("pmt_list_backlog",
			mcp.WithDescription("List all backlog issues for a project (issues not assigned to any phase). Returns a paginated array."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the project")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default: 50)")),
			mcp.WithNumber("offset", mcp.Description("Number of results to skip (default: 0)")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleListBacklog,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("pmt_get_issue",
			mcp.WithDescription("Get an issue by its ID. Returns the full issue object including spec, labels, status, priority, and type."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleGetIssue,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("pmt_create_issue",
			mcp.WithDescription("Create a new issue in a project. If phase_id is omitted, the issue goes to the backlog. Defaults: priority=medium, type=task, status=open. The spec field accepts full specification documents in markdown."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("UUID of the parent project")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Issue title (required, non-empty)")),
			mcp.WithString("phase_id", mcp.Description("UUID of the phase (optional — omit for backlog)")),
			mcp.WithString("spec", mcp.Description("Issue specification/description in markdown (optional)")),
			mcp.WithString("priority", mcp.Description("Issue priority (default: medium)"), mcp.Enum("low", "medium", "high")),
			mcp.WithString("type", mcp.Description("Issue type (default: task)"), mcp.Enum("task", "bug", "feature", "improvement")),
			mcp.WithString("due_date", mcp.Description("Due date in RFC3339 format, e.g. 2026-04-15T00:00:00Z (optional)")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleCreateIssue,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("pmt_update_issue",
			mcp.WithDescription("Update an existing issue. Only provided fields are changed. To change status, use pmt_transition_issue instead. To clear the due date, set clear_due_date to true."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue to update")),
			mcp.WithString("title", mcp.Description("New issue title")),
			mcp.WithString("spec", mcp.Description("New specification/description")),
			mcp.WithString("priority", mcp.Description("New priority"), mcp.Enum("low", "medium", "high")),
			mcp.WithString("type", mcp.Description("New issue type"), mcp.Enum("task", "bug", "feature", "improvement")),
			mcp.WithString("due_date", mcp.Description("New due date in RFC3339 format")),
			mcp.WithBoolean("clear_due_date", mcp.Description("Set to true to remove the due date")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleUpdateIssue,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("pmt_transition_issue",
			mcp.WithDescription("Change the status of an issue. Valid transitions: open->in_progress|canceled, in_progress->done|stopped|canceled, stopped->in_progress|canceled, done->in_progress|closed. Terminal states: closed, canceled."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue")),
			mcp.WithString("status", mcp.Required(), mcp.Description("Target status"), mcp.Enum("open", "in_progress", "done", "closed", "stopped", "canceled")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleTransitionIssue,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("pmt_delete_issue",
			mcp.WithDescription("Delete an issue by its ID. This is irreversible."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue to delete")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleDeleteIssue,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("pmt_add_label_to_issue",
			mcp.WithDescription("Attach a label to an issue. Both the issue and label must exist."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue")),
			mcp.WithString("label_id", mcp.Required(), mcp.Description("UUID of the label to attach")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleAddLabelToIssue,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("pmt_remove_label_from_issue",
			mcp.WithDescription("Detach a label from an issue."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue")),
			mcp.WithString("label_id", mcp.Required(), mcp.Description("UUID of the label to detach")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleRemoveLabelFromIssue,
	)
}

func (s *Server) handleListIssuesByPhase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	phaseID, err := shared.ParseID(args["phase_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid phase_id: %v", err)), nil
	}

	issues, err := s.issues.ListByPhase(ctx, phaseID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list issues: %v", err)), nil
	}

	all := make([]map[string]any, len(issues))
	for i, iss := range issues {
		all[i] = marshalIssue(iss)
	}
	offset, limit := paginationArgs(args, 50)
	return jsonResult(paginate(all, offset, limit))
}

func (s *Server) handleListBacklog(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, err := shared.ParseID(args["project_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid project_id: %v", err)), nil
	}

	issues, err := s.issues.ListBacklog(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list backlog: %v", err)), nil
	}

	all := make([]map[string]any, len(issues))
	for i, iss := range issues {
		all[i] = marshalIssue(iss)
	}
	offset, limit := paginationArgs(args, 50)
	return jsonResult(paginate(all, offset, limit))
}

func (s *Server) handleGetIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["issue_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	iss, err := s.issues.GetByID(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get issue: %v", err)), nil
	}
	return jsonResult(marshalIssue(iss))
}

func (s *Server) handleCreateIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, _ := args["project_id"].(string)
	title, _ := args["title"].(string)

	input := issueapp.CreateInput{
		ProjectID: projectID,
		Title:     title,
	}

	if v, ok := args["phase_id"].(string); ok && v != "" {
		input.PhaseID = &v
	}
	if v, ok := args["spec"].(string); ok {
		input.Spec = v
	}
	if v, ok := args["priority"].(string); ok {
		input.Priority = v
	}
	if v, ok := args["type"].(string); ok {
		input.Type = v
	}
	if v, ok := args["due_date"].(string); ok && v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid due_date format: %v", err)), nil
		}
		input.DueDate = &t
	}

	iss, err := s.issues.Create(ctx, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create issue: %v", err)), nil
	}
	return jsonResult(marshalIssue(iss))
}

func (s *Server) handleUpdateIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["issue_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	input := issueapp.UpdateInput{}

	if v, ok := args["title"].(string); ok {
		input.Title = &v
	}
	if v, ok := args["spec"].(string); ok {
		input.Spec = &v
	}
	if v, ok := args["priority"].(string); ok {
		input.Priority = &v
	}
	if v, ok := args["type"].(string); ok {
		input.Type = &v
	}
	if v, ok := args["clear_due_date"].(bool); ok && v {
		input.ClearDueDate = true
	} else if v, ok := args["due_date"].(string); ok && v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid due_date format: %v", err)), nil
		}
		input.DueDate = &t
	}

	iss, err := s.issues.Update(ctx, id, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update issue: %v", err)), nil
	}
	return jsonResult(marshalIssue(iss))
}

func (s *Server) handleTransitionIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["issue_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	status, _ := args["status"].(string)
	iss, err := s.issues.Transition(ctx, id, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to transition issue: %v", err)), nil
	}
	return jsonResult(marshalIssue(iss))
}

func (s *Server) handleDeleteIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["issue_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	if err := s.issues.Delete(ctx, id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete issue: %v", err)), nil
	}
	return mcp.NewToolResultText("issue deleted"), nil
}

func (s *Server) handleAddLabelToIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueID, err := shared.ParseID(args["issue_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}
	labelID, err := shared.ParseID(args["label_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid label_id: %v", err)), nil
	}

	if err := s.issues.AddLabel(ctx, issueID, labelID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add label: %v", err)), nil
	}
	return mcp.NewToolResultText("label added to issue"), nil
}

func (s *Server) handleRemoveLabelFromIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueID, err := shared.ParseID(args["issue_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}
	labelID, err := shared.ParseID(args["label_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid label_id: %v", err)), nil
	}

	if err := s.issues.RemoveLabel(ctx, issueID, labelID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove label: %v", err)), nil
	}
	return mcp.NewToolResultText("label removed from issue"), nil
}

func marshalIssue(iss issue.Issue) map[string]any {
	m := map[string]any{
		"id":         iss.ID().String(),
		"project_id": iss.ProjectID().String(),
		"type":       string(iss.Type()),
		"title":      iss.Title().String(),
		"spec":       iss.Spec(),
		"status":     string(iss.Status()),
		"priority":   string(iss.Priority()),
		"created_at": iss.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at": iss.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if iss.PhaseID() != nil {
		m["phase_id"] = iss.PhaseID().String()
	} else {
		m["phase_id"] = nil
	}

	if iss.DueDate() != nil {
		m["due_date"] = iss.DueDate().Format("2006-01-02T15:04:05Z07:00")
	} else {
		m["due_date"] = nil
	}

	labelIDs := make([]string, len(iss.LabelIDs()))
	for i, lid := range iss.LabelIDs() {
		labelIDs[i] = lid.String()
	}
	m["label_ids"] = labelIDs

	return m
}
