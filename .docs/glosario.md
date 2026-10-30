# Glosario

Lenguaje ubicuo de PMT. La documentación usa el término en español y el código usa su nombre en inglés.

## Términos de diseño

| Término | Significado | En Go | Dónde vive |
| --- | --- | --- | --- |
| **Entidad** | Concepto del negocio con identidad propia, que perdura aunque cambien sus datos. | `struct` con ID | `domain/` |
| **Objeto de valor** | Concepto definido solo por su valor, sin identidad propia. | Tipo o `struct` sin ID | `domain/` o `kernel/` |
| **Agregado** | Grupo de entidades y objetos de valor que cambia como una unidad y protege sus reglas. | `struct` y sus métodos | `domain/` |
| **Raíz del agregado** | Entidad por la que se accede y se modifica todo el agregado. | `struct` principal | `domain/` |
| **Registro** | Fila de una tabla, representada en Go. No contiene reglas de negocio. | `struct` generado por `sqlc` | `store/sqlcgen/` |
| **Feature** | Conjunto de todo lo relacionado con un agregado: dominio, persistencia y operaciones. | Carpeta | `internal/features/<feature>/` |
| **Vertical slice** | Una operación completa en su propia carpeta, desde la entrada hasta la lógica. | Paquete | `commands/<operacion>/` o `queries/<operacion>/` |
| **Command** | Operación que cambia estado. | Slice | `commands/` |
| **Query** | Operación que solo lee, sin cambiar estado. | Slice | `queries/` |
| **Handler** | Código que ejecuta una operación. | `handler.go` | Dentro de cada slice |

## Términos del negocio

| Término | En código | Significado |
| --- | --- | --- |
| **Proyecto** | `Project` | Trabajo que se gestiona en PMT y se divide en etapas. Es la raíz de su propio agregado; sus etapas son agregados aparte. |
| **Status de un proyecto** | `Status` | Estado de un proyecto: `planning`, `stand_by`, `in_dev`, `working`, `stopped` o `cancelled`. `in_dev` significa que se desarrolla la primera versión; `working`, que esa versión ya terminó. Solo cambia mediante las transiciones permitidas. |
| **Versión de producto** | `ProductVersion` | Versión del producto de un proyecto, de texto libre, por ejemplo `1.0.0`. No es control de concurrencia. |
| **Etapa** | `Stage` | Parte de un proyecto que agrupa tareas, con un orden dentro de él. Es la raíz de su propio agregado y referencia a su proyecto por ID. |
| **Status de una etapa** | `Status` | Estado de una etapa: `planning` (se definen sus tareas), `stand_by` (lista, esperando su turno), `in_dev` (sus tareas se están trabajando), `done` (terminada y cerrada), `stopped` (detenida por una causa mayor) o `cancelled` (ya no se hará). Solo cambia mediante las transiciones permitidas, y solo pasa a `done` si todas sus tareas están en `done`. |
| **Tarea** | `Task` | Unidad de trabajo de una etapa. Es la raíz de su propio agregado y referencia a su etapa por ID. |
| **Status de una tarea** | `Status` | Estado de una tarea: `backlog` (definida, aún no liberada), `ready` (liberada para desarrollo), `in_progress` (en desarrollo), `in_review` (en revisión de código), `changes_requested` (con correcciones pendientes), `in_qa` (en verificación de QA), `pending_close` (aprobada por QA, falta el cierre), `done` (verificada y cerrada) o `cancelled` (ya no se hará). |
| **Agente** | `Agent` | Trabajador de IA que opera en herdr y se comunica con PMT por MCP. |
| **Perfil** | `Profile` | Conjunto de responsabilidades de quien actúa: PP, PC, TL, DevFrontend, DevBackend, CR, QAP o QA. |
| **Actor** | `Actor` | Quien realiza una operación, sea agente o usuario humano, con su perfil. |
