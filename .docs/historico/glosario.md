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
| **Versión de producto** | `ProductVersion` | Versión del producto de un proyecto, por ejemplo `1.0.0`. En `planning` es texto libre y cambia con `update_project`. En `working` se sube con `change_product_version`: forma `X.Y.Z`, mayor que la actual y con motivo; cada cambio deja un registro inmutable, consultable con `list_product_version_changes`. Fuera de esas dos fases no cambia. No es control de concurrencia. |
| **Etapa** | `Stage` | Parte de un proyecto que agrupa tareas, con un orden dentro de él. Es la raíz de su propio agregado y referencia a su proyecto por ID. |
| **Status de una etapa** | `Status` | Estado de una etapa: `planning` (se definen sus tareas), `stand_by` (lista para iniciarse), `in_dev` (sus tareas se están trabajando), `done` (terminada y cerrada), `stopped` (detenida por una causa mayor) o `cancelled` (ya no se hará). Solo cambia mediante las transiciones permitidas, y solo pasa a `done` si sus tareas están en `done` o `cancelled`, con al menos una en `done`; las retiradas no cuentan. |
| **Tarea** | `Task` | Unidad de trabajo de una etapa. Es la raíz de su propio agregado y referencia a su etapa por ID. |
| **Status de una tarea** | `Status` | Estado de una tarea: `backlog` (definida, aún no liberada), `ready` (liberada para desarrollo), `in_dev` (en desarrollo), `in_review` (en revisión de código), `changes_requested` (con correcciones pendientes), `in_qa` (en verificación de QA), `pending_close` (aprobada, falta el cierre del TL), `done` (verificada y cerrada) o `cancelled` (ya no se hará). |
| **Cancelar o retirar** | `cancelled`, `retire` | `cancelled` es un status reversible, para trabajo ya iniciado o que puede volver. `retire` es definitivo, para lo que nunca debió estar: no se deshace ni aparece en las consultas. |
| **Versión 1** | | Las etapas de la planeación original con PP: las que no tienen registro `add`. Al cerrar, cancelar o retirar la última abierta, si al menos una quedó en `done`, el proyecto pasa solo de `in_dev` a `working`. |
| **Cambio** | `Change` | Registro inmutable de una etapa o tarea, de tipo `add`, `edit` o `retire`, con su contenido completo, su motivo, quién lo pidió y su fecha. El original no es un cambio: queda en la fila de la etapa o tarea. En una etapa en planning propio, el registro `add` se escribe al salir de esa fase. Es una entidad del agregado `Stage` o `Task`. Nunca se edita ni se borra. |
| **Planning propio** | `InOwnPlanning` | Fase de una etapa nacida con `add_stage` o `import_stage` con el proyecto en `working`: se edita libremente, sin registros, y su registro `add` se escribe al salir. No se reabre. |
| **Bug** | `Bug` | Fallo reportado en un proyecto en `working`, hallado en pruebas de rutina o en el uso del sistema, con sus pasos para reproducirlo y la versión de producto donde apareció. Lo registran PP o el usuario. No tiene dev asignado. Es la raíz de su propio agregado, referencia a su proyecto por ID y no pertenece a ninguna etapa. Un fallo hallado en desarrollo no es bug: entra como tarea. |
| **Status de un bug** | `Status` | Estado de un bug: `reported` (reportado, aún sin confirmar), `confirmed` (reproducido, pendiente de corregir), `in_dev` (en corrección), `in_review` (en revisión de código), `changes_requested` (con correcciones pendientes), `in_qa` (en verificación de QA), `pending_close` (aprobado, falta el cierre del TL), `done` (corregido y verificado) o `rejected` (QA no lo reprodujo en `reported`; no se corregirá). |
| **Retirar** | `Retire` | Sacar una etapa o tarea de las consultas registrando un cambio, sin borrar nada. Sigue en el historial y no se deshace. |
| **Hallazgo** | `Finding` | Algo que CR o QA encontró en una tarea o un bug, bloqueante o no. Nace en `open` y termina en `fixed` o `dismissed`. No cambia el status del elemento ni frena transiciones. Es la raíz de su propio agregado y referencia a su tarea o bug por ID. |
| **Status de un hallazgo** | `Status` | Estado de un hallazgo: `open` (pendiente), `fixed` (corregido y verificado por CR, leyendo el código, o por QA) o `dismissed` (descartado por el TL). No se reabre. |
| **Hallazgo bloqueante** | `Blocking` | Hallazgo que impide aprobar: se devuelve de inmediato. Los no bloqueantes se devuelven juntos al terminar. |
| **Resolución** | `Resolution` | Forma de cerrar un hallazgo, `fixed` o `dismissed`, con motivo obligatorio. |
| **Comentario** | `Comment` | Constancia de una decisión que no es una transición, o de un bloqueo, sobre un proyecto, una etapa, una tarea o un bug. No cambia el status del elemento, no se edita ni se borra, y referencia a su elemento por ID. |
| **Plan QAP** | `Plan` | Plan de pruebas que QAP diseña para una etapa y vincula a tareas de ella. El TL de la etapa lo aprueba y QA lo ejecuta. No frena transiciones, y los bugs no tienen plan. Es la raíz de su propio agregado y referencia a su etapa y sus tareas por ID. |
| **Status de un plan** | `Status` | Estado de un plan QAP: `draft` (QAP lo edita), `submitted` (espera la decisión del TL), `approved` (el TL aprobó su última versión) o `returned` (devuelto con correcciones). Un plan devuelto exige una versión nueva antes de reenviarse. |
| **Versión de un plan** | `Version` | Registro inmutable del contenido de un plan QAP: camino feliz, casos de error, motivo, autor y fecha. Es una entidad del agregado `Plan`. Toda versión nueva deja el plan en `draft`. |
| **Camino feliz** | `HappyPath` | Casos de un plan QAP en los que todo sale bien. Obligatorio para enviarlo. |
| **Casos de error** | `ErrorCases` | Casos de un plan QAP en los que algo debe rechazarse o fallar. Obligatorios para enviarlo. |
| **Correcciones** | `ReviewNotes` | Lo que el TL pide cambiar al devolver un plan QAP. Son obligatorias al devolver, siguen visibles al revisar y aprobar las limpia. |
| **Development Planning** | `ProfileDP` | Perfil que define el cómo de un grupo de tareas o de un bug antes de que Dev empiece. Registra su plan en PMT; no escribe código, no cambia status y no aprueba nada. |
| **Plan de desarrollo** | `DevPlan` | Cómo se hará un grupo de tareas de una etapa, o un bug. DP lo registra y Dev lo sigue; CR revisa también contra él. No tiene status ni aprobación y no frena transiciones: vale su versión más reciente. Es la raíz de su propio agregado y referencia a su etapa y sus tareas, o a su bug, por ID. |
| **Versión de un plan de desarrollo** | `Version` | Registro inmutable del contenido de un plan de desarrollo: alcance, decisiones técnicas, estados de pantalla, casos límite, manejo de errores y criterios de aceptación, todos obligatorios, con motivo en las versiones nuevas, autor y fecha. Es una entidad del agregado `DevPlan`. |
| **Agente** | `Agent` | Trabajador de IA que opera en herdr y consulta y registra el estado en PMT por MCP. |
| **Aviso** | | Mensaje que se envía a un agente con `herdr agent prompt`. Es la única forma en que los agentes se comunican. No se guarda en PMT: lo que deba quedar registrado va en PMT. |
| **Sesión** | | Conversación de un agente. Dura un solo encargo y arranca limpia: el estado se lee en PMT. |
| **Vigía** | | Observa a los agentes con herramientas de herdr y da aviso al TL si todos llevan un rato quietos. Queda fuera de PMT. |
| **Perfil** | `Profile` | Conjunto de responsabilidades de quien actúa: PP, PC, TL, DP, DevFrontend, DevBackend, CR, QAP o QA, y `User`, el usuario humano, que tiene lo de PP. |
| **Actor** | `Actor` | Quien realiza una operación, sea agente o usuario humano, con su perfil. Lo resuelve el servidor a partir del token. |
| **Token** | | Credencial de un agente o del usuario. Determina su `Actor`. PMT solo guarda su hash. |
| **Historial de status** | `StatusChange` | Registro inmutable de cada cambio de status: desde, hacia, actor, fecha y motivo. Sobrevive al elemento eliminado. |
