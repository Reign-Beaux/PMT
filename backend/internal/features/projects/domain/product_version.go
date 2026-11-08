package domain

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrProductVersionNotWorking indica que la versión de producto solo se sube en working.
	ErrProductVersionNotWorking = kernel.Conflict("el proyecto no está en working: change_product_version solo sube la versión en working; en planning usa update_project")
	// ErrProductVersionFormat indica que la versión no tiene la forma X.Y.Z.
	ErrProductVersionFormat = kernel.Invalid("la versión de producto debe tener la forma X.Y.Z: tres enteros sin signo y sin ceros a la izquierda")
	// ErrProductVersionNotGreater indica que la versión nueva no supera a la actual.
	ErrProductVersionNotGreater = kernel.Invalid("la versión de producto nueva debe ser mayor que la actual")
	// ErrProductVersionReasonRequired indica que falta el motivo del cambio de versión.
	ErrProductVersionReasonRequired = kernel.Invalid("el motivo es obligatorio al subir la versión de producto")
	// ErrProductVersionChanged indica que la versión o el status cambiaron entre leerlos y guardarlos.
	ErrProductVersionChanged = kernel.Conflict("la versión de producto cambió, vuelve a consultar")
)

var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// parseVersion devuelve los tres números de una versión X.Y.Z, o false si no tiene esa forma.
func parseVersion(version string) ([3]uint64, bool) {
	var parts [3]uint64
	match := semverPattern.FindStringSubmatch(version)
	if match == nil {
		return parts, false
	}
	for i := range parts {
		n, err := strconv.ParseUint(match[i+1], 10, 64)
		if err != nil {
			return parts, false
		}
		parts[i] = n
	}
	return parts, true
}

// greater indica si a es mayor que b, dígito por dígito.
func greater(a, b [3]uint64) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// ProductVersionChange es un cambio inmutable de la versión de producto de un
// proyecto. Copia nombre y perfil del actor para conservar el que tenía al actuar.
type ProductVersionChange struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	From         string
	To           string
	ActorName    string
	ActorProfile string
	Reason       string
	ChangedAt    time.Time
}

// ChangeProductVersion sube la versión de producto de un proyecto en working y
// devuelve su registro. La versión nueva debe tener la forma X.Y.Z y, si la
// actual también la tiene, ser mayor. La fecha del registro se asigna al guardarlo.
func (p *Project) ChangeProductVersion(version, reason string, actor kernel.Actor) (ProductVersionChange, error) {
	if p.Status != StatusWorking {
		return ProductVersionChange{}, ErrProductVersionNotWorking
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ProductVersionChange{}, ErrProductVersionReasonRequired
	}
	version = strings.TrimSpace(version)
	next, ok := parseVersion(version)
	if !ok {
		return ProductVersionChange{}, ErrProductVersionFormat
	}
	if current, ok := parseVersion(p.ProductVersion); ok && !greater(next, current) {
		return ProductVersionChange{}, ErrProductVersionNotGreater
	}

	change := ProductVersionChange{
		ID:           uuid.New(),
		ProjectID:    p.ID,
		From:         p.ProductVersion,
		To:           version,
		ActorName:    actor.Name,
		ActorProfile: string(actor.Profile),
		Reason:       reason,
	}
	p.ProductVersion = version
	return change, nil
}
