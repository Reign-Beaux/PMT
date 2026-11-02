// Package kernel contiene los conceptos compartidos entre features.
package kernel

import (
	"errors"
	"fmt"
)

// ErrorKind clasifica un error de negocio sin depender de ningún transporte.
type ErrorKind int

const (
	// KindInvalid indica que la entrada no cumple una regla.
	KindInvalid ErrorKind = iota + 1
	// KindNotFound indica que el recurso no existe.
	KindNotFound
	// KindConflict indica que la operación choca con el estado actual.
	KindConflict
	// KindForbidden indica que el perfil de quien actúa no permite la operación.
	KindForbidden
)

// Error es un error de negocio con un mensaje apto para quien hace la petición.
type Error struct {
	Kind    ErrorKind
	Message string
	cause   error
}

func (e *Error) Error() string {
	return e.Message
}

// Unwrap devuelve el error original de uno creado con Prefix.
func (e *Error) Unwrap() error {
	return e.cause
}

// Prefix antepone un contexto al mensaje de un error de negocio y conserva su
// tipo, para saber qué elemento de un lote falló. Otro error se envuelve igual.
func Prefix(prefix string, err error) error {
	var business *Error
	if errors.As(err, &business) {
		return &Error{Kind: business.Kind, Message: prefix + business.Message, cause: err}
	}
	return fmt.Errorf("%s%w", prefix, err)
}

// Invalid crea un error de entrada inválida.
func Invalid(message string) *Error {
	return &Error{Kind: KindInvalid, Message: message}
}

// NotFound crea un error de recurso inexistente.
func NotFound(message string) *Error {
	return &Error{Kind: KindNotFound, Message: message}
}

// Conflict crea un error de conflicto con el estado actual.
func Conflict(message string) *Error {
	return &Error{Kind: KindConflict, Message: message}
}

// Forbidden crea un error de permiso.
func Forbidden(message string) *Error {
	return &Error{Kind: KindForbidden, Message: message}
}

// Errores de autorización y de motivo comunes a las features con status.
var (
	// ErrTransitionForbidden indica que el perfil no puede hacer la transición.
	ErrTransitionForbidden = Forbidden("tu perfil no puede hacer esta transición")
	// ErrNotStageLead indica que quien actúa no es el TL de la etapa.
	ErrNotStageLead = Forbidden("no eres el TL de la etapa")
	// ErrReasonRequired indica que la transición exige un motivo.
	ErrReasonRequired = Invalid("el motivo es obligatorio en esta transición")
)
