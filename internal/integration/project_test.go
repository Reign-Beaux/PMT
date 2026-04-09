package integration_test

import (
	"net/http"
	"testing"
)

func TestProject_FullCRUD(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	// Create
	resp := post(t, "/projects", map[string]any{
		"name":        "PMT",
		"description": "Project Management Tool",
	})
	assertStatus(t, resp, http.StatusCreated)
	created := decode(t, resp)
	id := str(t, created, "id")
	if str(t, created, "name") != "PMT" {
		t.Errorf("expected name %q, got %q", "PMT", str(t, created, "name"))
	}
	if str(t, created, "description") != "Project Management Tool" {
		t.Errorf("unexpected description: %q", str(t, created, "description"))
	}

	// Get by ID
	resp = get(t, "/projects/"+id)
	assertStatus(t, resp, http.StatusOK)
	fetched := decode(t, resp)
	if str(t, fetched, "id") != id {
		t.Errorf("id mismatch: got %q", str(t, fetched, "id"))
	}

	// List — must contain the created project
	resp = get(t, "/projects")
	assertStatus(t, resp, http.StatusOK)
	list := decodeList(t, resp)
	if len(list) != 1 {
		t.Fatalf("expected 1 project, got %d", len(list))
	}
	if str(t, list[0], "id") != id {
		t.Error("list returned wrong project")
	}

	// Update
	resp = patch(t, "/projects/"+id, map[string]any{
		"name":        "PMT v2",
		"description": "Updated",
	})
	assertStatus(t, resp, http.StatusOK)
	updated := decode(t, resp)
	if str(t, updated, "name") != "PMT v2" {
		t.Errorf("expected updated name, got %q", str(t, updated, "name"))
	}

	// Delete
	resp = delete(t, "/projects/"+id)
	assertStatus(t, resp, http.StatusNoContent)

	// Confirm gone
	resp = get(t, "/projects/"+id)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestProject_Errors(t *testing.T) {
	skipIfNoDB(t)
	cleanup(t)

	t.Run("empty name rejected", func(t *testing.T) {
		resp := post(t, "/projects", map[string]any{"name": ""})
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("not found", func(t *testing.T) {
		resp := get(t, "/projects/00000000-0000-0000-0000-000000000001")
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		resp := get(t, "/projects/not-a-uuid")
		assertStatus(t, resp, http.StatusBadRequest)
	})
}
