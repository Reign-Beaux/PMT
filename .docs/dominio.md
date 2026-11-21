# Dominio

Conceptos que PMT guarda y reglas que aplica. La documentación usa el término en español; el código, su nombre en inglés. Todas las reglas de este documento las valida el backend, sea por REST o por MCP.

## Conceptos

| Término | En código | Significado |
| --- | --- | --- |
| **Proyecto** | `Project` | Trabajo que se gestiona en PMT y se divide en etapas. Tiene status y versión de producto. |
| **Etapa** | `Stage` | Parte de un proyecto con un orden (`position`), un TL responsable y un equipo de devs. Agrupa tareas. |
| **Tarea** | `Task` | Unidad de trabajo de una etapa, con un dueño (`assignee`) cuando un dev la toma. |
| **Bug** | `Bug` | Fallo de un proyecto en `working`, hallado en pruebas de rutina o en el uso. Pertenece al proyecto, no a una etapa, y no tiene dueño. Guarda pasos para reproducirlo, la versión donde apareció y quién lo reportó. |
| **Cambio** | `Change` | Registro inmutable de tipo `add`, `edit` o `retire` sobre una etapa o tarea, con contenido completo, motivo, quién lo pidió y fecha. |
| **Historial de status** | `StatusChange` | Registro inmutable de cada cambio de status de un proyecto, etapa, tarea o bug: desde, hacia, actor, perfil, fecha y motivo. |
| **Cambio de versión** | `ProductVersionChange` | Registro inmutable de cada subida de la versión de producto en `working`. |
| **Hallazgo** | `Finding` | Algo que CR o QA encontró en una tarea o un bug, bloqueante o no. Nace `open` y termina `fixed` o `dismissed`. |
| **Comentario** | `Comment` | Constancia de una decisión que no es una transición, o de un bloqueo, sobre un proyecto, etapa, tarea o bug. |
| **Plan QAP** | `test_plans` | Plan de pruebas de una etapa, vinculado a tareas. QAP lo escribe, el TL lo aprueba y QA lo ejecuta. |
| **Plan de desarrollo** | `DevPlan` | Cómo se hará un grupo de tareas o un bug. DP lo escribe y Dev lo sigue. |
| **Agente** | `Agent` | Trabajador de IA con nombre, perfil y token. No se borra: se revoca. |
| **Perfil** | `Profile` | PP, PC, TL, DP, DevFrontend, DevBackend, CR, QAP, QA o `User`, el usuario humano. |
| **Actor** | `Actor` | Quien realiza una operación, con su perfil. El servidor lo obtiene del token. |

**Cancelar o retirar.** `cancelled` es un status reversible, para trabajo iniciado o que puede volver. Retirar (`retire`) es definitivo, para lo que nunca debió estar: deja un cambio y saca el elemento de las consultas, sin borrar nada.

**Etapa o tarea activa:** la que no está retirada ni cancelada.

**Versión 1:** las etapas de la planeación original, las que no tienen registro `add`.

## Proyectos

Nace en `planning`. `in_dev` significa que se desarrolla la versión 1; `working`, que la versión 1 terminó y el proyecto crece con etapas nuevas y bugs.

| Desde | Puede pasar a |
| --- | --- |
| `planning` | `stand_by`, `stopped`, `cancelled` |
| `stand_by` | `planning`, `in_dev`, `stopped`, `cancelled` |
| `in_dev` | `stopped`, `cancelled` |
| `working` | `stopped`, `cancelled` |
| `stopped` | `planning`, `stand_by`, `in_dev`, `working`, `cancelled` |
| `cancelled` | `planning` |

- Solo PP y `User` lo cambian. Detener, cancelar y reabrir exigen motivo.
- **Automático `stand_by → in_dev`:** al iniciar la primera etapa con `start_stage`.
- **Automático `in_dev → working`:** cuando todas las etapas de la versión 1 quedan en `done`, `cancelled` o retiradas, con al menos una en `done`. Nunca es manual. Las etapas agregadas después no cuentan.
- **Versión de producto:** en `planning` es texto libre y cambia con `update_project`. En `working` se sube con `change_product_version`: forma `X.Y.Z`, mayor que la actual y con motivo, y queda en el historial. En otros status no cambia.

## Etapas

Nace en `planning`.

| Desde | Puede pasar a |
| --- | --- |
| `planning` | `stand_by`, `stopped`, `cancelled` |
| `stand_by` | `planning`, `in_dev`, `stopped`, `cancelled` |
| `in_dev` | `done`, `stopped`, `cancelled` |
| `done` | `in_dev` |
| `stopped` | `planning`, `stand_by`, `in_dev`, `cancelled` |
| `cancelled` | `planning` |

- **Perfiles:** el TL de la etapa la inicia y la cierra. PP y `User` liberan, devuelven, cancelan y reabren. PP, `User` y el TL de la etapa detienen, reanudan y reabren una etapa en `done`. Detener, cancelar y reabrir exigen motivo.
- **Cierre:** solo pasa a `done` si todas sus tareas activas están en `done` o `cancelled`, con al menos una en `done`. Una etapa vacía o toda cancelada se cancela, no se cierra.
- **Asignación:** PP o `User` le ponen o cambian el TL con `assign_stage_lead` y lo quitan con `remove_stage_lead`, solo con la etapa sin iniciar (`planning` o `stand_by`); quitarlo no cambia el equipo. El TL arma su equipo con `add_stage_member` y `remove_stage_member`.

## Tareas

Nace en `backlog`.

| Desde | Acción | Perfil | Hacia |
| --- | --- | --- | --- |
| `backlog` | Liberar | TL | `ready` |
| `ready` | Tomar | Dev del equipo | `in_dev` |
| `in_dev` | Entregar | Dev dueño | `in_review` |
| `in_review` | Aprobar | CR | `in_qa` |
| `in_review` | Pedir corrección | CR | `changes_requested` |
| `in_review` | Aprobar un cambio interno, con motivo | CR | `pending_close` |
| `changes_requested` | Retomar | Dev dueño | `in_dev` |
| `in_qa` | Aprobar, con evidencia en el motivo | QA | `pending_close` |
| `in_qa` | Pedir corrección | QA | `changes_requested` |
| `pending_close` | Cerrar | TL | `done` |
| `pending_close` | Devolver, con motivo | TL | `in_review`, `in_qa` |
| cualquiera menos `done` | Cancelar, con motivo | TL, PP, `User` | `cancelled` |
| `cancelled` | Reabrir, con motivo | TL, PP, `User` | `backlog` |

- **Tomar:** `ready → in_dev` deja la tarea a nombre del dev, que debe ser del equipo. Después solo ese dev la mueve. El TL puede cambiar el dueño con `assign_task`.
- **Lotes:** `change_tasks_status` mueve varias tareas de una etapa al mismo status, todas o ninguna.
- **Revisar una tarea entregada:** `revise_task` sobre una tarea en `in_review`, `in_qa` o `pending_close` la regresa a `changes_requested`. Una tarea en `done` no se revisa: se agrega otra con `add_task`.

## Bugs

Solo existen con el proyecto en `working`. Nacen en `reported`.

| Desde | Puede pasar a | Quién |
| --- | --- | --- |
| `reported` | `confirmed`, `rejected` (con motivo) | QA |
| `confirmed` | `in_dev` | Dev |
| `in_dev` | `in_review` | Dev |
| `in_review` | `in_qa`, `changes_requested`, `pending_close` (con motivo) | CR |
| `changes_requested` | `in_dev` | Dev |
| `in_qa` | `pending_close` (con evidencia), `changes_requested` | QA |
| `pending_close` | `done`, o con motivo `in_review` o `in_qa` | TL |
| `rejected` | `reported` (con motivo) | QA o TL |

- Solo PP y `User` crean y corrigen bugs (`create_bug`, `update_bug`). No se eliminan.
- Solo QA rechaza, y solo un bug `reported` que no reproduce. Uno que ya pasó de `reported` es real y se corrige.
- Un bug no tiene dueño ni etapa: el TL indica en un comentario a qué dev le toca.
- Un fallo hallado durante el desarrollo no es bug: entra como tarea.

## Reglas entre status

Detener o cancelar un padre congela lo de abajo, sin cambiar su status.

| Operación | Exige |
| --- | --- |
| Avanzar una tarea | Proyecto en `in_dev` o `working` y etapa en `in_dev` |
| Cancelar o reabrir una tarea | Proyecto ni `stopped` ni `cancelled`, y etapa ni `done` ni `cancelled` |
| Liberar una etapa (`planning → stand_by`) | Al menos una tarea activa |
| Pasar una etapa a `in_dev` | Proyecto en `in_dev` o `working` |
| Mover una etapa con el proyecto `stopped` o `cancelled` | Solo hacia `stopped` o `cancelled` |
| Liberar el proyecto (`planning → stand_by`) | Al menos una etapa activa y ninguna en `planning` |
| Proyecto `stand_by → in_dev` | Al menos una etapa en `stand_by` |
| Proyecto a `working` | Etapas de la versión 1 en `done`, `cancelled` o retiradas, con una en `done` |
| Mover un bug | Proyecto en `working` |

## Planning y cambios

**En `planning`** todo se edita y elimina libremente: `create_*`, `update_*` y `delete_*` de etapas y tareas, y `delete_project`. Fuera de `planning` responden `el proyecto no está en planning`. `update_project` es la excepción: siempre cambia nombre y descripción.

**Fuera de `planning`** etapas y tareas cambian solo con registros inmutables, que dan trazabilidad:

| Cambio | Comandos | Efecto |
| --- | --- | --- |
| `add` | `add_stage`, `add_task` | Nace al final, versión 1 |
| `edit` | `revise_stage`, `revise_task` | Nueva versión de nombre o título y descripción |
| `retire` | `retire_stage`, `retire_task` | Sale de las consultas, sin borrarse |

- Todo cambio exige motivo y quién lo pidió.
- La fila original no se toca. Las lecturas devuelven la última versión.
- Lo retirado responde `NotFound` en todas las operaciones y no se deshace. Una etapa con tareas sin retirar no se retira.
- El historial se consulta con `list_stage_changes` y `list_task_changes`.

**Planning propio de una etapa.** Una etapa creada con `add_stage` o `import_stage` con el proyecto en `working` se planea sola, sin mover el proyecto: sus tareas se crean, editan y eliminan libremente, sin registros. Al salir de `planning` se escribe su registro `add` con el contenido definitivo, y desde ahí sigue la regla de cambios. No se reabre.

## Hallazgos

- CR y QA los registran con `create_finding`, sobre una tarea o un bug, indicando si son bloqueantes.
- CR y QA los cierran como `fixed`. El TL de la etapa los descarta como `dismissed`; en un bug, o si la tarea ya no existe, cualquier TL. Ambos cierres exigen motivo. No se reabren.
- No cambian el status del elemento ni frenan transiciones: devolver es decisión de CR o QA.
- `list_findings` consulta por tarea, por bug o por etapa, y con `pending=true` solo los abiertos.

## Comentarios

- Todos los perfiles menos PC comentan con `add_comment`, incluido `User`.
- No se editan ni se borran, y no cambian el status del elemento.

## Planes QAP

| Desde | Operación | Quién | Hacia |
| --- | --- | --- | --- |
| Nuevo | `create_test_plan` | QAP | `draft`, versión 1 |
| `draft`, `returned`, `approved` | `revise_test_plan`, con motivo | QAP | `draft`, versión nueva |
| `draft` | `submit_test_plan` | QAP | `submitted` |
| `submitted` | `approve_test_plan` | TL de la etapa | `approved` |
| `submitted` | `return_test_plan`, con correcciones | TL de la etapa | `returned` |

- Cada versión guarda camino feliz y casos de error, obligatorios para enviarlo.
- Un plan devuelto exige versión nueva antes de reenviarse.
- No frena transiciones. Los bugs no tienen plan QAP.

## Planes de desarrollo

- DP los crea con `create_dev_plan`, para una etapa con al menos una de sus tareas, o para un bug. Los versiona con `revise_dev_plan`, con motivo.
- Cada versión trae seis campos obligatorios: `scope`, `technical_decisions`, `screen_states`, `edge_cases`, `error_handling` y `acceptance_criteria`. Si uno no aplica, se escribe el motivo.
- No tienen status ni aprobación: vale la versión más reciente. No frenan transiciones.

## Permisos

- **Todo comando de escritura exige perfil.** Uno sin permiso recibe `forbidden` antes de que se lea la base.
- **Planeación** (`create_*`, `update_*`, `delete_*`, `add_*`, `retire_*` de proyectos, etapas y tareas, `revise_task`, `import_*`, `change_product_version`, `release_project`, `assign_stage_lead`, `remove_stage_lead`): solo PP y `User`. `revise_stage` también el TL de esa etapa.
- **PC** solo lee: no escribe nada, ni comentarios.
- **`User`** tiene lo de PP, pero no aprueba como CR, QA ni TL, ni gestiona hallazgos o planes.
- **Un TL** solo actúa sobre su etapa y sus tareas.
- Las lecturas (`list_*`, `get_*`) son para todos.

## Inmutabilidad

Cambios, historiales de status y de versión, comentarios y versiones de planes no se editan ni se borran: triggers de PostgreSQL rechazan `UPDATE`, `DELETE` y `TRUNCATE`. Los historiales, hallazgos, comentarios y planes no tienen llave foránea a su elemento, así que sobreviven aunque se elimine en `planning`.
