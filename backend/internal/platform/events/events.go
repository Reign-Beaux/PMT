// Package events publica los cambios confirmados en la base de datos a los
// suscriptores del servidor, y los entrega a los clientes WebSocket.
package events

// Channel es el canal de NOTIFY de PostgreSQL.
const Channel = "pmt_changes"

// Recursos que la UI muestra y que un aviso puede invalidar.
const (
	ResourceProject      = "project"
	ResourceStage        = "stage"
	ResourceTask         = "task"
	ResourceBug          = "bug"
	ResourceComment      = "comment"
	ResourceStatusChange = "status_change"
)

// Tipos de ChangeEvent.
const (
	TypeResync     = "resync"
	TypeInvalidate = "invalidate"
)

// ChangeEvent es el contrato hacia los suscriptores, el mismo que recibe el
// navegador: {"type":"resync"} o {"type":"invalidate","resource":"<recurso>"}.
type ChangeEvent struct {
	Type     string `json:"type"`
	Resource string `json:"resource,omitempty"`
}

// ResyncEvent es el evento que pide recargar todo.
func ResyncEvent() ChangeEvent { return ChangeEvent{Type: TypeResync} }

func validResource(resource string) bool {
	switch resource {
	case ResourceProject, ResourceStage, ResourceTask, ResourceBug, ResourceComment, ResourceStatusChange:
		return true
	}
	return false
}
