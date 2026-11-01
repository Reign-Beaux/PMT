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
| **Status de una etapa** | `Status` | Estado de una etapa: `planning` (se definen sus tareas), `stand_by` (lista, esperando su turno), `in_dev` (sus tareas se están trabajando), `done` (terminada y cerrada), `stopped` (detenida por una causa mayor) o `cancelled` (ya no se hará). Solo cambia mediante las transiciones permitidas, y solo pasa a `done` si sus tareas están en `done` o `cancelled`, con al menos una en `done`; las retiradas no cuentan. |
| **Tarea** | `Task` | Unidad de trabajo de una etapa. Es la raíz de su propio agregado y referencia a su etapa por ID. |
| **Status de una tarea** | `Status` | Estado de una tarea: `backlog` (definida, aún no liberada), `ready` (liberada para desarrollo), `in_dev` (en desarrollo), `in_review` (en revisión de código), `changes_requested` (con correcciones pendientes), `in_qa` (en verificación de QA), `pending_close` (aprobada, falta el cierre del TL), `done` (verificada y cerrada) o `cancelled` (ya no se hará). |
| **Cancelar o retirar** | `cancelled`, `retire` | `cancelled` es un status reversible, para trabajo ya iniciado o que puede volver. `retire` es definitivo, para lo que nunca debió estar: no se deshace ni aparece en las consultas. |
| **Versión 1** | | Las etapas de la planeación original con PP: las que no tienen registro `add`. Al cerrar, cancelar o retirar la última abierta, si al menos una quedó en `done`, el proyecto pasa solo de `in_dev` a `working`. |
| **Cambio** | `Change` | Registro inmutable de una etapa o tarea, de tipo `add`, `edit` o `retire`, con su contenido completo, su motivo, quién lo pidió y su fecha. El original no es un cambio: queda en la fila de la etapa o tarea. En una etapa en planning propio, el registro `add` se escribe al salir de esa fase. Es una entidad del agregado `Stage` o `Task`. Nunca se edita ni se borra. |
| **Planning propio** | `InOwnPlanning` | Fase de una etapa nacida con `add_stage` o `import_stage` con el proyecto en `working`: se edita libremente, sin registros, y su registro `add` se escribe al salir. No se reabre. |
| **Bug** | `Bug` | Fallo reportado en un proyecto, con sus pasos para reproducirlo y la versión de producto donde apareció. Es la raíz de su propio agregado, referencia a su proyecto por ID y no pertenece a ninguna etapa. |
| **Status de un bug** | `Status` | Estado de un bug: `reported` (reportado, aún sin confirmar), `confirmed` (reproducido, pendiente de corregir), `in_dev` (en corrección), `in_review` (en revisión de código), `changes_requested` (con correcciones pendientes), `in_qa` (en verificación de QA), `pending_close` (aprobado, falta el cierre del TL), `done` (corregido y verificado) o `rejected` (no se corregirá). |
| **Retirar** | `Retire` | Sacar una etapa o tarea de las consultas registrando un cambio, sin borrar nada. Sigue en el historial y no se deshace. |
| **Hallazgo** | `Finding` | Algo que CR o QA encontró en una tarea o un bug, bloqueante o no. Nace en `open` y termina en `fixed` o `dismissed`. No cambia el status del elemento ni frena transiciones. |
| **Agente** | `Agent` | Trabajador de IA que opera en herdr y se comunica con PMT por MCP. |
| **Perfil** | `Profile` | Conjunto de responsabilidades de quien actúa: PP, PC, TL, DevFrontend, DevBackend, CR, QAP o QA. |
| **Actor** | `Actor` | Quien realiza una operación, sea agente o usuario humano, con su perfil. Lo resuelve el servidor a partir del token. |
| **Token** | | Credencial de un agente o del usuario. Determina su `Actor`. PMT solo guarda su hash. |
| **Historial de status** | `StatusChange` | Registro inmutable de cada cambio de status: desde, hacia, actor, fecha y motivo. Sobrevive al elemento eliminado. |
