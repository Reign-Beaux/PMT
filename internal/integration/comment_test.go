package integration_test

import (
	"net/http"
	"testing"
)

func TestComment_PhaseIssue(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	_, _, issueBase := setupProjectAndPhase(t)

	resp := post(t, issueBase, map[string]any{"title": "Investigate crash"})
	assertStatus(t, resp, http.StatusCreated)
	issueID := str(t, decode(t, resp), "id")

	commentBase := issueBase + "/" + issueID + "/comments"

	// Create comment
	resp = post(t, commentBase, map[string]any{"body": "Root cause: nil pointer in auth middleware"})
	assertStatus(t, resp, http.StatusCreated)
	c := decode(t, resp)
	commentID := str(t, c, "id")
	if str(t, c, "body") != "Root cause: nil pointer in auth middleware" {
		t.Errorf("unexpected comment body: %q", str(t, c, "body"))
	}
	if str(t, c, "issue_id") != issueID {
		t.Error("comment issue_id mismatch")
	}

	// Create second comment
	resp = post(t, commentBase, map[string]any{"body": "Fixed in commit abc123"})
	assertStatus(t, resp, http.StatusCreated)
	comment2ID := str(t, decode(t, resp), "id")
	_ = comment2ID

	// List — ordered by created_at ASC
	resp = get(t, commentBase)
	assertStatus(t, resp, http.StatusOK)
	list := decodeList(t, resp)
	if len(list) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(list))
	}
	if str(t, list[0], "body") != "Root cause: nil pointer in auth middleware" {
		t.Error("comments not ordered by created_at ASC")
	}

	// Delete first comment
	resp = delete(t, commentBase+"/"+commentID)
	assertStatus(t, resp, http.StatusNoContent)

	resp = get(t, commentBase)
	assertStatus(t, resp, http.StatusOK)
	list = decodeList(t, resp)
	if len(list) != 1 {
		t.Fatalf("expected 1 comment after delete, got %d", len(list))
	}
	if str(t, list[0], "id") != comment2ID {
		t.Error("wrong comment remaining")
	}
}

func TestComment_BacklogIssue(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	resp := post(t, "/projects", map[string]any{"name": "P"})
	assertStatus(t, resp, http.StatusCreated)
	projectID := str(t, decode(t, resp), "id")

	backlogBase := "/projects/" + projectID + "/issues"

	resp = post(t, backlogBase, map[string]any{"title": "Refactor config"})
	assertStatus(t, resp, http.StatusCreated)
	issueID := str(t, decode(t, resp), "id")

	commentBase := backlogBase + "/" + issueID + "/comments"

	// Create and list
	resp = post(t, commentBase, map[string]any{"body": "Config module is tightly coupled"})
	assertStatus(t, resp, http.StatusCreated)

	resp = get(t, commentBase)
	assertStatus(t, resp, http.StatusOK)
	list := decodeList(t, resp)
	if len(list) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(list))
	}
	if str(t, list[0], "body") != "Config module is tightly coupled" {
		t.Error("unexpected comment body")
	}
}

func TestComment_Errors(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	_, _, issueBase := setupProjectAndPhase(t)

	resp := post(t, issueBase, map[string]any{"title": "T"})
	assertStatus(t, resp, http.StatusCreated)
	issueID := str(t, decode(t, resp), "id")

	t.Run("empty body rejected", func(t *testing.T) {
		resp := post(t, issueBase+"/"+issueID+"/comments", map[string]any{"body": ""})
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("comment on non-existent issue returns 404", func(t *testing.T) {
		resp := post(t, issueBase+"/00000000-0000-0000-0000-000000000001/comments",
			map[string]any{"body": "Hello"})
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("delete non-existent comment returns 404", func(t *testing.T) {
		resp := delete(t, issueBase+"/"+issueID+"/comments/00000000-0000-0000-0000-000000000001")
		assertStatus(t, resp, http.StatusNotFound)
	})
}
