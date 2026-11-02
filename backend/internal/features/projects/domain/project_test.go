package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewProject(t *testing.T) {
	p, err := NewProject("  PMT  ", "  Gestor  ")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == uuid.Nil {
		t.Error("ID vacío")
	}
	if p.Name != "PMT" || p.Description != "Gestor" {
		t.Errorf("proyecto = %+v", p)
	}
	if p.Status != StatusPlanning || p.ProductVersion != "" {
		t.Errorf("proyecto = %+v", p)
	}
}

func TestNewProjectRequiresName(t *testing.T) {
	if _, err := NewProject("   ", ""); !errors.Is(err, ErrProjectNameRequired) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateKeepsDataWhenInvalid(t *testing.T) {
	p, err := NewProject("PMT", "Gestor")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Update("PMT", "Gestor", "1.0.0"); err != nil {
		t.Fatal(err)
	}

	if err := p.Update("", "Otro", "2.0.0"); !errors.Is(err, ErrProjectNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if p.Name != "PMT" || p.Description != "Gestor" || p.ProductVersion != "1.0.0" {
		t.Errorf("proyecto = %+v", p)
	}
}

func TestUpdateTrimsAndReplacesProductVersion(t *testing.T) {
	p, err := NewProject("PMT", "Gestor")
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Update("PMT", "Gestor", "  v1 beta  "); err != nil {
		t.Fatal(err)
	}
	if p.ProductVersion != "v1 beta" {
		t.Errorf("versión de producto = %q", p.ProductVersion)
	}

	if err := p.Update("PMT", "Gestor", ""); err != nil {
		t.Fatal(err)
	}
	if p.ProductVersion != "" {
		t.Errorf("versión de producto = %q", p.ProductVersion)
	}
}

func TestUpdateKeepsStatus(t *testing.T) {
	p, err := NewProject("PMT", "Gestor")
	if err != nil {
		t.Fatal(err)
	}
	p.Status = StatusStopped

	if err := p.Update("PMT", "Gestor", ""); err != nil {
		t.Fatal(err)
	}
	if p.Status != StatusStopped {
		t.Errorf("status = %s", p.Status)
	}
}

func TestUpdateFreezesProductVersionOutsidePlanning(t *testing.T) {
	p, err := NewProject("PMT", "Gestor")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Update("PMT", "Gestor", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	p.Status = StatusInDev

	if err := p.Update("Otro", "Nuevo", " 1.0.0 "); err != nil {
		t.Fatal(err)
	}
	if p.Name != "Otro" || p.Description != "Nuevo" {
		t.Errorf("proyecto = %+v", p)
	}

	if err := p.Update("Tercero", "Nuevo", "2.0.0"); !errors.Is(err, ErrProductVersionFrozen) {
		t.Fatalf("err = %v", err)
	}
	if p.Name != "Otro" || p.ProductVersion != "1.0.0" {
		t.Errorf("proyecto = %+v", p)
	}
}

func TestEnsurePlanning(t *testing.T) {
	if err := (Project{Status: StatusPlanning}).EnsurePlanning(); err != nil {
		t.Errorf("err = %v", err)
	}
	for _, status := range []Status{StatusStandBy, StatusInDev, StatusWorking, StatusStopped, StatusCancelled} {
		if err := (Project{Status: status}).EnsurePlanning(); !errors.Is(err, ErrProjectNotPlanning) {
			t.Errorf("%s: err = %v", status, err)
		}
	}
}

func TestIsWorking(t *testing.T) {
	if !(Project{Status: StatusWorking}).IsWorking() {
		t.Error("working debe responder true")
	}
	for _, status := range []Status{StatusPlanning, StatusStandBy, StatusInDev, StatusStopped, StatusCancelled} {
		if (Project{Status: status}).IsWorking() {
			t.Errorf("%s debe responder false", status)
		}
	}
}
