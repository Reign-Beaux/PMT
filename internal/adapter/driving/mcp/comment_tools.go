package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	commentapp "project-management-tools/internal/application/comment"
	"project-management-tools/internal/domain/comment"
	"project-management-tools/internal/domain/shared"
)

func (s *Server) registerCommentTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("list_comments",
			mcp.WithDescription("List all comments on an issue, ordered by creation date. Returns a paginated array."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default: 50)")),
			mcp.WithNumber("offset", mcp.Description("Number of results to skip (default: 0)")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleListComments,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("create_comment",
			mcp.WithDescription("Add a comment to an issue. The issue must exist. Body must be non-empty."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue to comment on")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Comment text (required, non-empty)")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleCreateComment,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("delete_comment",
			mcp.WithDescription("Delete a comment by its ID."),
			mcp.WithString("comment_id", mcp.Required(), mcp.Description("UUID of the comment to delete")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleDeleteComment,
	)
}

func (s *Server) handleListComments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueID, err := shared.ParseID(args["issue_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	comments, err := s.comments.ListByIssue(ctx, issueID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list comments: %v", err)), nil
	}

	all := make([]map[string]any, len(comments))
	for i, c := range comments {
		all[i] = marshalComment(c)
	}
	offset, limit := paginationArgs(args, 50)
	return jsonResult(paginate(all, offset, limit))
}

func (s *Server) handleCreateComment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueID, _ := args["issue_id"].(string)
	body, _ := args["body"].(string)

	c, err := s.comments.Create(ctx, commentapp.CreateInput{
		IssueID: issueID,
		Body:    body,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create comment: %v", err)), nil
	}
	return jsonResult(marshalComment(c))
}

func (s *Server) handleDeleteComment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["comment_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid comment_id: %v", err)), nil
	}

	if err := s.comments.Delete(ctx, id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete comment: %v", err)), nil
	}
	return mcp.NewToolResultText("comment deleted"), nil
}

func marshalComment(c comment.Comment) map[string]any {
	return map[string]any{
		"id":         c.ID().String(),
		"issue_id":   c.IssueID().String(),
		"body":       c.Body().String(),
		"created_at": c.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at": c.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
