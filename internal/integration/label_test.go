package integration_test

import (
	"net/http"
	"testing"
)

func TestLabel_FullCRUD(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	resp := post(t, "/projects", map[string]any{"name": "P"})
	assertStatus(t, resp, http.StatusCreated)
	projectID := str(t, decode(t, resp), "id")

	labelBase := "/projects/" + projectID + "/labels"

	// Create label with explicit color
	resp = post(t, labelBase, map[string]any{
		"name":  "bug",
		"color": "#ef4444",
	})
	assertStatus(t, resp, http.StatusCreated)
	created := decode(t, resp)
	labelID := str(t, created, "id")
	if str(t, created, "name") != "bug" {
		t.Errorf("unexpected label name: %q", str(t, created, "name"))
	}
	if str(t, created, "color") != "#ef4444" {
		t.Errorf("unexpected color: %q", str(t, created, "color"))
	}

	// Create label without color — gets default
	resp = post(t, labelBase, map[string]any{"name": "feature"})
	assertStatus(t, resp, http.StatusCreated)
	defaultColor := str(t, decode(t, resp), "color")
	if defaultColor == "" {
		t.Error("expected default color to be set")
	}

	// List
	resp = get(t, labelBase)
	assertStatus(t, resp, http.StatusOK)
	list := decodeList(t, resp)
	if len(list) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(list))
	}

	// Update name and color
	resp = patch(t, labelBase+"/"+labelID, map[string]any{
		"name":  "critical-bug",
		"color": "#dc2626",
	})
	assertStatus(t, resp, http.StatusOK)
	updated := decode(t, resp)
	if str(t, updated, "name") != "critical-bug" {
		t.Error("name not updated")
	}
	if str(t, updated, "color") != "#dc2626" {
		t.Error("color not updated")
	}

	// Delete
	resp = delete(t, labelBase+"/"+labelID)
	assertStatus(t, resp, http.StatusNoContent)

	resp = get(t, labelBase)
	assertStatus(t, resp, http.StatusOK)
	if len(decodeList(t, resp)) != 1 {
		t.Error("expected 1 label after delete")
	}
}

func TestLabel_AssignAndRemove_PhaseIssue(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	projectID, _, issueBase := setupProjectAndPhase(t)
	labelBase := "/projects/" + projectID + "/labels"

	// Create two labels
	resp := post(t, labelBase, map[string]any{"name": "bug", "color": "#ff0000"})
	assertStatus(t, resp, http.StatusCreated)
	label1ID := str(t, decode(t, resp), "id")

	resp = post(t, labelBase, map[string]any{"name": "priority", "color": "#ff9900"})
	assertStatus(t, resp, http.StatusCreated)
	label2ID := str(t, decode(t, resp), "id")

	// Create issue
	resp = post(t, issueBase, map[string]any{"title": "Broken header", "type": "bug"})
	assertStatus(t, resp, http.StatusCreated)
	issueID := str(t, decode(t, resp), "id")

	assignBase := issueBase + "/" + issueID + "/labels"

	// Assign both labels
	resp = post(t, assignBase, map[string]any{"label_id": label1ID})
	assertStatus(t, resp, http.StatusNoContent)

	resp = post(t, assignBase, map[string]any{"label_id": label2ID})
	assertStatus(t, resp, http.StatusNoContent)

	// Issue must now return both label IDs
	resp = get(t, issueBase+"/"+issueID)
	assertStatus(t, resp, http.StatusOK)
	labelIDs := strSlice(t, decode(t, resp), "label_ids")
	if len(labelIDs) != 2 {
		t.Fatalf("expected 2 labels on issue, got %d", len(labelIDs))
	}
	if !contains(labelIDs, label1ID) || !contains(labelIDs, label2ID) {
		t.Error("label IDs not found in issue response")
	}

	// Remove one label
	resp = delete(t, assignBase+"/"+label1ID)
	assertStatus(t, resp, http.StatusNoContent)

	resp = get(t, issueBase+"/"+issueID)
	assertStatus(t, resp, http.StatusOK)
	labelIDs = strSlice(t, decode(t, resp), "label_ids")
	if len(labelIDs) != 1 {
		t.Fatalf("expected 1 label after removal, got %d", len(labelIDs))
	}
	if labelIDs[0] != label2ID {
		t.Error("wrong label remaining after removal")
	}
}

func TestLabel_AssignAndRemove_BacklogIssue(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	resp := post(t, "/projects", map[string]any{"name": "P"})
	assertStatus(t, resp, http.StatusCreated)
	projectID := str(t, decode(t, resp), "id")

	labelBase := "/projects/" + projectID + "/labels"
	backlogBase := "/projects/" + projectID + "/issues"

	resp = post(t, labelBase, map[string]any{"name": "improvement"})
	assertStatus(t, resp, http.StatusCreated)
	labelID := str(t, decode(t, resp), "id")

	resp = post(t, backlogBase, map[string]any{"title": "Refactor auth"})
	assertStatus(t, resp, http.StatusCreated)
	issueID := str(t, decode(t, resp), "id")

	// Assign via backlog route
	resp = post(t, backlogBase+"/"+issueID+"/labels", map[string]any{"label_id": labelID})
	assertStatus(t, resp, http.StatusNoContent)

	resp = get(t, backlogBase+"/"+issueID)
	assertStatus(t, resp, http.StatusOK)
	labelIDs := strSlice(t, decode(t, resp), "label_ids")
	if len(labelIDs) != 1 || labelIDs[0] != labelID {
		t.Error("label not found on backlog issue")
	}
}

func TestLabel_Errors(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	resp := post(t, "/projects", map[string]any{"name": "P"})
	assertStatus(t, resp, http.StatusCreated)
	projectID := str(t, decode(t, resp), "id")
	labelBase := "/projects/" + projectID + "/labels"

	t.Run("empty name rejected", func(t *testing.T) {
		resp := post(t, labelBase, map[string]any{"name": ""})
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("invalid color rejected", func(t *testing.T) {
		resp := post(t, labelBase, map[string]any{"name": "x", "color": "red"})
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("assign non-existent label returns 404", func(t *testing.T) {
		backlogBase := "/projects/" + projectID + "/issues"
		resp := post(t, backlogBase, map[string]any{"title": "T"})
		assertStatus(t, resp, http.StatusCreated)
		issueID := str(t, decode(t, resp), "id")

		resp = post(t, backlogBase+"/"+issueID+"/labels",
			map[string]any{"label_id": "00000000-0000-0000-0000-000000000001"})
		assertStatus(t, resp, http.StatusNotFound)
	})
}
