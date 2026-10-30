package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewTask(t *testing.T) {
	stageID := uuid.New()

	task, err := NewTask(stageID, "  Revisar  ", "  Detalle  ")

	if err != nil {
		t.Fatal(err)
	}
	if task.ID == uuid.Nil || task.StageID != stageID {
		t.Errorf("tarea = %+v", task)
	}
	if task.Title != "Revisar" || task.Description != "Detalle" {
		t.Errorf("tarea = %+v", task)
	}
	if task.Status != StatusBacklog || task.Blocked || task.Version != 1 {
		t.Errorf("tarea = %+v", task)
	}
}

func TestNewTaskRequiresTitle(t *testing.T) {
	if _, err := NewTask(uuid.New(), " ", ""); !errors.Is(err, ErrTaskTitleRequired) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateKeepsDataWhenInvalid(t *testing.T) {
	task, err := NewTask(uuid.New(), "Revisar", "Detalle")
	if err != nil {
		t.Fatal(err)
	}

	if err := task.Update("", "Otro"); !errors.Is(err, ErrTaskTitleRequired) {
		t.Fatalf("err = %v", err)
	}
	if task.Title != "Revisar" || task.Description != "Detalle" {
		t.Errorf("tarea = %+v", task)
	}
}
