package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	commentapp "project-management-tools/internal/application/comment"
	issueapp "project-management-tools/internal/application/issue"
	"project-management-tools/internal/domain/shared"
)

func (s *Server) registerCompoundTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("submit_dev_handoff",
			mcp.WithDescription("Submit a dev handoff brief and transition the issue to 'done' in a single step."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue")),
			mcp.WithString("handoff_brief", mcp.Required(), mcp.Description("The dev handoff brief text")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleSubmitDevHandoff,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("qa_fail_with_findings",
			mcp.WithDescription("Reject an issue by transitioning it to 'in_progress' and creating multiple bug issues."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the original feature issue")),
			mcp.WithString("findings", mcp.Required(), mcp.Description("JSON array of bug objects, e.g. [{\"title\":\"Bug 1\",\"spec\":\"Details\"}]")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleQAFailWithFindings,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_qa_batch_context",
			mcp.WithDescription("Get the latest comment (usually handoff brief) for a batch of issues."),
			mcp.WithString("issue_ids", mcp.Required(), mcp.Description("JSON array of issue UUIDs, e.g. [\"id1\", \"id2\"]")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleGetQABatchContext,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("qa_pass",
			mcp.WithDescription("Add a QA summary comment and transition the issue to 'closed'."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue")),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Summary of QA testing performed")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleQAPass,
	)
	s.mcpServer.AddTool(
		mcp.NewTool("resolve_investigation",
			mcp.WithDescription("Close an investigation by adding a findings comment, creating follow-up issues, and transitioning the investigation to 'done'."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the investigation issue")),
			mcp.WithString("findings", mcp.Required(), mcp.Description("Detailed findings from the investigation")),
			mcp.WithString("follow_up_issues", mcp.Description("Optional: JSON array of issues to create, e.g. [{\"title\":\"Task 1\",\"spec\":\"Details\",\"type\":\"task\"}]")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleResolveInvestigation,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_fix_context",
			mcp.WithDescription("Get context for fixing QA rejected issues. Returns the feature issue, its latest comments, and the full details of all associated finding/bug issues."),
			mcp.WithString("feature_issue_id", mcp.Required(), mcp.Description("UUID of the original feature issue")),
			mcp.WithString("finding_ids", mcp.Required(), mcp.Description("JSON array of finding/bug issue UUIDs, e.g. [\"id1\", \"id2\"]")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleGetFixContext,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("start_issue",
			mcp.WithDescription("Moves an issue to 'in_progress' and returns its full details. Use this to start working on an issue in one step."),
			mcp.WithString("issue_id", mcp.Required(), mcp.Description("UUID of the issue to start")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		s.handleStartIssue,
	)
}

func (s *Server) handleGetFixContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	featureIssueIDStr, _ := args["feature_issue_id"].(string)
	findingIDsJSON, _ := args["finding_ids"].(string)

	featureIssueID, err := shared.ParseID(featureIssueIDStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid feature_issue_id: %v", err)), nil
	}

	featureIssue, err := s.issues.GetByID(ctx, featureIssueID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get feature issue: %v", err)), nil
	}

	comments, err := s.comments.ListByIssue(ctx, featureIssueID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get feature issue comments: %v", err)), nil
	}

	var findingIDsRaw []string
	if err := json.Unmarshal([]byte(findingIDsJSON), &findingIDsRaw); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid finding_ids JSON: %v", err)), nil
	}

	var findings []map[string]any
	for _, idStr := range findingIDsRaw {
		id, err := shared.ParseID(idStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid finding_id '%s': %v", idStr, err)), nil
		}
		findingIss, err := s.issues.GetByID(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get finding issue '%s': %v", idStr, err)), nil
		}
		// marshalIssue is defined in issue_tools.go
		findings = append(findings, marshalIssue(findingIss))
	}

	// Find the last dev handoff and last QA rejection if possible. We will just return the last 5 comments to be safe,
	// or the raw list of comments so the agent has full context.
	var commentsData []map[string]any
	for _, c := range comments {
		commentsData = append(commentsData, map[string]any{
			"id":         c.ID().String(),
			"body":       c.Body().String(),
			"created_at": c.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	featureData := marshalIssue(featureIssue)
	featureData["comments"] = commentsData

	result := map[string]any{
		"feature":  featureData,
		"findings": findings,
	}

	return jsonResult(result)
}
func (s *Server) handleStartIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, err := shared.ParseID(args["issue_id"].(string))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	iss, err := s.issues.Transition(ctx, id, "in_progress")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start issue: %v", err)), nil
	}
	// We use jsonResult and marshalIssue but marshalIssue is in issue_tools.go, which is part of the same package, so this works.
	// However, jsonResult is unexported, let's make sure it works if we use it.
	return jsonResult(marshalIssue(iss))
}

func (s *Server) handleSubmitDevHandoff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueIDStr, _ := args["issue_id"].(string)
	handoffBrief, _ := args["handoff_brief"].(string)

	issueID, err := shared.ParseID(issueIDStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	_, err = s.comments.Create(ctx, commentapp.CreateInput{
		IssueID: issueIDStr,
		Body:    handoffBrief,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create comment: %v", err)), nil
	}

	iss, err := s.issues.Transition(ctx, issueID, "done")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to transition issue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Handoff submitted and issue %s transitioned to done.", iss.ID().String())), nil
}

type findingInput struct {
	Title string `json:"title"`
	Spec  string `json:"spec"`
}

func (s *Server) handleQAFailWithFindings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueIDStr, _ := args["issue_id"].(string)
	findingsJSON, _ := args["findings"].(string)

	issueID, err := shared.ParseID(issueIDStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	parentIssue, err := s.issues.GetByID(ctx, issueID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get parent issue: %v", err)), nil
	}

	var findings []findingInput
	if err := json.Unmarshal([]byte(findingsJSON), &findings); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid findings JSON: %v", err)), nil
	}

	var createdBugIDs []string
	for _, f := range findings {
		input := issueapp.CreateInput{
			ProjectID: parentIssue.ProjectID().String(),
			Title:     f.Title,
			Spec:      f.Spec,
			Type:      "bug",
			Priority:  "high",
		}
		if parentIssue.PhaseID() != nil {
			pid := parentIssue.PhaseID().String()
			input.PhaseID = &pid
		}

		newIss, err := s.issues.Create(ctx, input)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create bug issue '%s': %v", f.Title, err)), nil
		}
		createdBugIDs = append(createdBugIDs, newIss.ID().String())
	}

	commentBody := fmt.Sprintf("QA Rejected. Created bugs: %v", createdBugIDs)
	_, err = s.comments.Create(ctx, commentapp.CreateInput{
		IssueID: issueIDStr,
		Body:    commentBody,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create comment: %v", err)), nil
	}

	iss, err := s.issues.Transition(ctx, issueID, "in_progress")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to transition parent issue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Issue %s rejected and moved to in_progress. Created bugs: %v", iss.ID().String(), createdBugIDs)), nil
}

func (s *Server) handleGetQABatchContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueIDsJSON, _ := args["issue_ids"].(string)

	var issueIDsRaw []string
	if err := json.Unmarshal([]byte(issueIDsJSON), &issueIDsRaw); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_ids JSON: %v", err)), nil
	}

	result := make(map[string]string)

	for _, idStr := range issueIDsRaw {
		id, err := shared.ParseID(idStr)
		if err != nil {
			result[idStr] = fmt.Sprintf("error: invalid id: %v", err)
			continue
		}

		comments, err := s.comments.ListByIssue(ctx, id)
		if err != nil {
			result[idStr] = fmt.Sprintf("error fetching comments: %v", err)
			continue
		}

		if len(comments) == 0 {
			result[idStr] = "No comments found."
		} else {
			// ListByIssue usually returns ordered by time. Let's take the last comment.
			latest := comments[len(comments)-1]
			result[idStr] = latest.Body().String()
		}
	}

	return jsonResult(result)
}

func (s *Server) handleQAPass(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueIDStr, _ := args["issue_id"].(string)
	summary, _ := args["summary"].(string)

	issueID, err := shared.ParseID(issueIDStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	_, err = s.comments.Create(ctx, commentapp.CreateInput{
		IssueID: issueIDStr,
		Body:    summary,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create summary comment: %v", err)), nil
	}

	iss, err := s.issues.Transition(ctx, issueID, "closed")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to transition issue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("QA Passed. Issue %s closed.", iss.ID().String())), nil
}

type followUpInput struct {
	Title string `json:"title"`
	Spec  string `json:"spec"`
	Type  string `json:"type"`
}

func (s *Server) handleResolveInvestigation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	issueIDStr, _ := args["issue_id"].(string)
	findings, _ := args["findings"].(string)
	followUpJSON, _ := args["follow_up_issues"].(string)

	issueID, err := shared.ParseID(issueIDStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid issue_id: %v", err)), nil
	}

	parentIssue, err := s.issues.GetByID(ctx, issueID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get investigation issue: %v", err)), nil
	}

	var createdIssues []string
	if followUpJSON != "" {
		var followUps []followUpInput
		if err := json.Unmarshal([]byte(followUpJSON), &followUps); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid follow_up_issues JSON: %v", err)), nil
		}

		for _, f := range followUps {
			t := f.Type
			if t == "" {
				t = "task"
			}
			input := issueapp.CreateInput{
				ProjectID: parentIssue.ProjectID().String(),
				Title:     f.Title,
				Spec:      f.Spec,
				Type:      t,
			}
			if parentIssue.PhaseID() != nil {
				pid := parentIssue.PhaseID().String()
				input.PhaseID = &pid
			}
			newIss, err := s.issues.Create(ctx, input)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create follow-up issue '%s': %v", f.Title, err)), nil
			}
			createdIssues = append(createdIssues, newIss.ID().String())
		}
	}

	commentBody := fmt.Sprintf("Investigation Findings:\n\n%s", findings)
	if len(createdIssues) > 0 {
		commentBody += fmt.Sprintf("\n\nFollow-up issues created: %v", createdIssues)
	}

	_, err = s.comments.Create(ctx, commentapp.CreateInput{
		IssueID: issueIDStr,
		Body:    commentBody,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create findings comment: %v", err)), nil
	}

	iss, err := s.issues.Transition(ctx, issueID, "done")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to transition investigation issue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Investigation %s resolved. Follow-ups: %v", iss.ID().String(), createdIssues)), nil
}
