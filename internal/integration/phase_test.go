package integration_test

import (
	"net/http"
	"testing"
)

func TestPhase_FullCRUD(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	// Precondition: create a project
	resp := post(t, "/projects", map[string]any{"name": "My Project"})
	assertStatus(t, resp, http.StatusCreated)
	projectID := str(t, decode(t, resp), "id")

	base := "/projects/" + projectID + "/phases"

	// Create first phase
	resp = post(t, base, map[string]any{
		"name":        "Sprint 1",
		"description": "First sprint",
	})
	assertStatus(t, resp, http.StatusCreated)
	p1 := decode(t, resp)
	phase1ID := str(t, p1, "id")
	if str(t, p1, "name") != "Sprint 1" {
		t.Errorf("unexpected phase name: %q", str(t, p1, "name"))
	}

	// Create second phase — order must auto-increment
	resp = post(t, base, map[string]any{"name": "Sprint 2"})
	assertStatus(t, resp, http.StatusCreated)
	p2 := decode(t, resp)
	phase2ID := str(t, p2, "id")
	_ = phase2ID

	// List — two phases
	resp = get(t, base)
	assertStatus(t, resp, http.StatusOK)
	list := decodeList(t, resp)
	if len(list) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(list))
	}

	// Get by ID
	resp = get(t, base+"/"+phase1ID)
	assertStatus(t, resp, http.StatusOK)
	if str(t, decode(t, resp), "id") != phase1ID {
		t.Error("get by id returned wrong phase")
	}

	// Update
	resp = patch(t, base+"/"+phase1ID, map[string]any{
		"name":        "Sprint 1 — done",
		"description": "Completed",
	})
	assertStatus(t, resp, http.StatusOK)
	if str(t, decode(t, resp), "name") != "Sprint 1 — done" {
		t.Error("update did not persist")
	}

	// Delete
	resp = delete(t, base+"/"+phase1ID)
	assertStatus(t, resp, http.StatusNoContent)

	resp = get(t, base+"/"+phase1ID)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestPhase_Errors(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	t.Run("create in non-existent project returns 404", func(t *testing.T) {
		resp := post(t, "/projects/00000000-0000-0000-0000-000000000001/phases",
			map[string]any{"name": "Sprint"})
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("empty phase name rejected", func(t *testing.T) {
		resp := post(t, "/projects", map[string]any{"name": "P"})
		assertStatus(t, resp, http.StatusCreated)
		pid := str(t, decode(t, resp), "id")

		resp = post(t, "/projects/"+pid+"/phases", map[string]any{"name": ""})
		assertStatus(t, resp, http.StatusBadRequest)
	})
}
