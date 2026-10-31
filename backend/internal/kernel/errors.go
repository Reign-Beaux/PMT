// Package kernel contiene los conceptos compartidos entre features.
package kernel

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
}

func (e *Error) Error() string {
	return e.Message
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
