package integration_test

import (
	"net/http"
	"testing"
)

// setup creates a project + phase and returns their IDs and the base path for phase issues.
func setupProjectAndPhase(t *testing.T) (projectID, phaseID, issueBase string) {
	t.Helper()

	resp := post(t, "/projects", map[string]any{"name": "Test Project"})
	assertStatus(t, resp, http.StatusCreated)
	projectID = str(t, decode(t, resp), "id")

	resp = post(t, "/projects/"+projectID+"/phases", map[string]any{"name": "Sprint 1"})
	assertStatus(t, resp, http.StatusCreated)
	phaseID = str(t, decode(t, resp), "id")

	issueBase = "/projects/" + projectID + "/phases/" + phaseID + "/issues"
	return
}

// ── Phase issue tests ─────────────────────────────────────────────────────────

func TestPhaseIssue_FullCRUD(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	projectID, _, issueBase := setupProjectAndPhase(t)
	_ = projectID

	// Create with all fields
	resp := post(t, issueBase, map[string]any{
		"title":    "Implement login",
		"spec":     "OAuth2 via Google",
		"priority": "high",
		"type":     "feature",
		"due_date": "2026-06-01T00:00:00Z",
	})
	assertStatus(t, resp, http.StatusCreated)
	created := decode(t, resp)
	issueID := str(t, created, "id")

	if str(t, created, "title") != "Implement login" {
		t.Errorf("unexpected title: %q", str(t, created, "title"))
	}
	if str(t, created, "type") != "feature" {
		t.Errorf("expected type feature, got %q", str(t, created, "type"))
	}
	if str(t, created, "priority") != "high" {
		t.Errorf("expected priority high, got %q", str(t, created, "priority"))
	}
	if str(t, created, "status") != "open" {
		t.Errorf("expected status open, got %q", str(t, created, "status"))
	}
	if str(t, created, "due_date") == "" {
		t.Error("expected due_date to be set")
	}
	if isNull(created, "phase_id") {
		t.Error("expected phase_id to be set")
	}
	if len(strSlice(t, created, "label_ids")) != 0 {
		t.Error("expected empty label_ids on creation")
	}

	// Get by ID
	resp = get(t, issueBase+"/"+issueID)
	assertStatus(t, resp, http.StatusOK)
	if str(t, decode(t, resp), "id") != issueID {
		t.Error("get by id returned wrong issue")
	}

	// List by phase
	resp = get(t, issueBase)
	assertStatus(t, resp, http.StatusOK)
	list := decodeList(t, resp)
	if len(list) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(list))
	}

	// Update — change title, type, clear due date
	resp = patch(t, issueBase+"/"+issueID, map[string]any{
		"title":    "Implement OAuth2 login",
		"type":     "task",
		"due_date": nil,
	})
	assertStatus(t, resp, http.StatusOK)
	updated := decode(t, resp)
	if str(t, updated, "title") != "Implement OAuth2 login" {
		t.Error("title not updated")
	}
	if str(t, updated, "type") != "task" {
		t.Error("type not updated")
	}
	if !isNull(updated, "due_date") {
		t.Error("due_date should have been cleared")
	}

	// Delete
	resp = delete(t, issueBase+"/"+issueID)
	assertStatus(t, resp, http.StatusNoContent)

	resp = get(t, issueBase+"/"+issueID)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestPhaseIssue_Transitions(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	_, _, issueBase := setupProjectAndPhase(t)

	resp := post(t, issueBase, map[string]any{"title": "Some work"})
	assertStatus(t, resp, http.StatusCreated)
	issueID := str(t, decode(t, resp), "id")

	tests := []struct {
		to         string
		wantStatus int
	}{
		{to: "in_progress", wantStatus: http.StatusOK},
		{to: "done", wantStatus: http.StatusOK},
		{to: "closed", wantStatus: http.StatusOK},
		// terminal — all transitions from closed are invalid
		{to: "open", wantStatus: http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run("→ "+tt.to, func(t *testing.T) {
			resp := patch(t, issueBase+"/"+issueID+"/transition", map[string]any{"status": tt.to})
			assertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestPhaseIssue_InvalidTransition(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	_, _, issueBase := setupProjectAndPhase(t)

	resp := post(t, issueBase, map[string]any{"title": "Direct to done"})
	assertStatus(t, resp, http.StatusCreated)
	issueID := str(t, decode(t, resp), "id")

	// open → done is not a valid transition
	resp = patch(t, issueBase+"/"+issueID+"/transition", map[string]any{"status": "done"})
	assertStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestPhaseIssue_Errors(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	t.Run("create in non-existent phase returns 404", func(t *testing.T) {
		resp := post(t, "/projects", map[string]any{"name": "P"})
		assertStatus(t, resp, http.StatusCreated)
		pid := str(t, decode(t, resp), "id")

		resp = post(t, "/projects/"+pid+"/phases/00000000-0000-0000-0000-000000000001/issues",
			map[string]any{"title": "Issue"})
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("empty title rejected", func(t *testing.T) {
		_, _, issueBase := setupProjectAndPhase(t)
		resp := post(t, issueBase, map[string]any{"title": ""})
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("invalid priority rejected", func(t *testing.T) {
		_, _, issueBase := setupProjectAndPhase(t)
		resp := post(t, issueBase, map[string]any{"title": "T", "priority": "urgent"})
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("invalid type rejected", func(t *testing.T) {
		_, _, issueBase := setupProjectAndPhase(t)
		resp := post(t, issueBase, map[string]any{"title": "T", "type": "epic"})
		assertStatus(t, resp, http.StatusBadRequest)
	})
}

// ── Backlog issue tests ───────────────────────────────────────────────────────

func TestBacklogIssue_FullCRUD(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	resp := post(t, "/projects", map[string]any{"name": "My Project"})
	assertStatus(t, resp, http.StatusCreated)
	projectID := str(t, decode(t, resp), "id")

	backlogBase := "/projects/" + projectID + "/issues"

	// Create backlog issue
	resp = post(t, backlogBase, map[string]any{
		"title":    "Fix production crash",
		"type":     "bug",
		"priority": "high",
	})
	assertStatus(t, resp, http.StatusCreated)
	created := decode(t, resp)
	issueID := str(t, created, "id")

	if !isNull(created, "phase_id") {
		t.Errorf("backlog issue must have null phase_id, got %v", created["phase_id"])
	}
	if str(t, created, "type") != "bug" {
		t.Errorf("expected type bug, got %q", str(t, created, "type"))
	}

	// List backlog
	resp = get(t, backlogBase)
	assertStatus(t, resp, http.StatusOK)
	list := decodeList(t, resp)
	if len(list) != 1 {
		t.Fatalf("expected 1 backlog issue, got %d", len(list))
	}
	if !isNull(list[0], "phase_id") {
		t.Error("backlog list item must have null phase_id")
	}

	// Get by ID
	resp = get(t, backlogBase+"/"+issueID)
	assertStatus(t, resp, http.StatusOK)

	// Update
	resp = patch(t, backlogBase+"/"+issueID, map[string]any{"title": "Fix crash — investigated"})
	assertStatus(t, resp, http.StatusOK)
	if str(t, decode(t, resp), "title") != "Fix crash — investigated" {
		t.Error("title not updated in backlog issue")
	}

	// Transition
	resp = patch(t, backlogBase+"/"+issueID+"/transition", map[string]any{"status": "in_progress"})
	assertStatus(t, resp, http.StatusOK)

	// Delete
	resp = delete(t, backlogBase+"/"+issueID)
	assertStatus(t, resp, http.StatusNoContent)

	resp = get(t, backlogBase+"/"+issueID)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestBacklogIssue_IsolatedFromPhaseIssues(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	projectID, _, issueBase := setupProjectAndPhase(t)
	backlogBase := "/projects/" + projectID + "/issues"

	// Create one issue in a phase and one in backlog
	resp := post(t, issueBase, map[string]any{"title": "Phase issue"})
	assertStatus(t, resp, http.StatusCreated)

	resp = post(t, backlogBase, map[string]any{"title": "Backlog issue"})
	assertStatus(t, resp, http.StatusCreated)

	// Backlog must only show backlog issue
	resp = get(t, backlogBase)
	assertStatus(t, resp, http.StatusOK)
	backlogList := decodeList(t, resp)
	if len(backlogList) != 1 {
		t.Fatalf("expected 1 item in backlog, got %d", len(backlogList))
	}
	if str(t, backlogList[0], "title") != "Backlog issue" {
		t.Error("backlog returned phase issue")
	}
}
