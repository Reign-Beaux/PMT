# Arquitectura del backend de PMT

Este documento define la arquitectura del backend de PMT. Es obligatorio para los agentes y desarrolladores que trabajen en él. Cualquier excepción debe documentarse aquí.

## Visión general

El backend es un **monolito modular en Go organizado por vertical slices**. Un solo proceso sirve la API REST al frontend y el MCP a los agentes, y es el único dueño de la base de datos. Ambas entradas llaman a los mismos handlers, por lo que las reglas de negocio no se duplican.

```text
Frontend React ── REST ──┐
                         ├── pmt-server ── commands y queries ── PostgreSQL
Agentes en herdr ─ MCP ──┘
```

Tres ideas organizan el código:

- **Feature:** agrupa todo lo de un agregado: su dominio, su persistencia y sus operaciones.
- **Vertical slice:** cada operación vive en su propia carpeta, con todo lo que necesita desde la entrada hasta la lógica. Añadir una operación agrega una carpeta, no hace crecer un fichero existente.
- **Commands y queries:** las operaciones que cambian estado se separan de las que solo leen.

| Necesidad | Elección |
| --- | --- |
| Lenguaje | Go |
| Base de datos | PostgreSQL |
| Acceso a datos | `pgx` y `sqlc` |
| Identificadores | UUID con `github.com/google/uuid` |
| Configuración | Viper |
| Migraciones | `goose` |
| MCP | SDK oficial de MCP para Go, con Streamable HTTP |
| Pruebas unitarias | Paquete estándar `testing` con `go test` |

## Estructura

```text
backend/
├── api/
│   └── features/
│       ├── <feature>/
│       └── system/
├── cmd/
│   └── pmt-server/
├── internal/
│   ├── app/
│   ├── features/
│   │   └── <feature>/
│   │       ├── feature.go
│   │       ├── domain/
│   │       ├── store/
│   │       │   ├── queries/
│   │       │   └── sqlcgen/
│   │       ├── commands/
│   │       │   └── <operacion>/
│   │       │       ├── handler.go
│   │       │       ├── handler_test.go
│   │       │       ├── rest.go
│   │       │       └── mcp.go
│   │       └── queries/
│   │           └── <operacion>/
│   │               ├── handler.go
│   │               ├── handler_test.go
│   │               ├── rest.go
│   │               └── mcp.go
│   ├── kernel/
│   └── platform/
│       ├── rest/
│       ├── mcp/
│       └── postgres/
│           └── migrations/
├── scripts/
└── bin/
```

Por ejemplo, la feature `projects` queda así:

```text
internal/features/projects/
├── feature.go
├── domain/
│   └── project.go
├── store/
│   ├── projects.go
│   ├── queries/
│   └── sqlcgen/
├── commands/
│   ├── changestatus/
│   ├── create/
│   ├── update/
│   └── delete/
└── queries/
    ├── get/
    └── list/
```

Una slice solo incluye `rest.go` o `mcp.go` si esa operación se expone por ese transporte.

## Responsabilidades

| Ruta | Responsabilidad |
| --- | --- |
| `api/features/<feature>/` | Contratos públicos de la feature. `<feature>.openapi.yaml` es la fuente de verdad de REST. `<feature>.mcp.json` documenta las herramientas MCP, cuya fuente de verdad es el código. |
| `api/features/system/` | Contratos técnicos: salud, disponibilidad y versión. |
| `cmd/pmt-server/` | Punto de entrada del servidor. |
| `internal/app/` | Configuración, conexión de dependencias, arranque y cierre. `registerFeatures`, en `app.go`, registra cada feature. Sin reglas de negocio. |
| `internal/features/<feature>/feature.go` | Construye las slices de la feature y registra sus rutas y herramientas. |
| `internal/features/<feature>/domain/` | Agregado, invariantes, errores y reglas de negocio. Lo comparten todas las slices de la feature. |
| `internal/features/<feature>/store/` | Persistencia de la feature. Implementa las interfaces que piden las slices y traduce entre registros y dominio. |
| `internal/features/<feature>/store/queries/` | Consultas SQL de la feature para `sqlc`. |
| `internal/features/<feature>/store/sqlcgen/` | Código generado por `sqlc`. No se edita a mano. |
| `internal/features/<feature>/commands/<operacion>/` | Una operación que cambia estado. |
| `internal/features/<feature>/queries/<operacion>/` | Una operación que solo lee. |
| `internal/kernel/` | Conceptos compartidos con la misma semántica en varias features: `Actor`, `Profile`, `Transactor`, identificadores y errores de negocio (`Invalid`, `NotFound`, `Conflict`), que `platform/rest` traduce a HTTP. |
| `internal/platform/rest/` | Servidor HTTP, middleware y manejo técnico de errores. |
| `internal/platform/mcp/` | Servidor MCP, autenticación y filtrado de herramientas por perfil. Sin lógica de negocio. |
| `internal/platform/postgres/` | Conexión, transacciones y ejecución de migraciones. |
| `internal/platform/postgres/migrations/` | Cambios versionados del esquema de datos. |
| `scripts/` | Automatización de desarrollo y operación. |
| `bin/` | Binarios locales, excluidos del control de versiones. |

## Vertical slices

Cada slice es un paquete de Go con estos ficheros:

| Fichero | Contenido |
| --- | --- |
| `handler.go` | La operación: su entrada, su resultado, su autorización y las interfaces mínimas que necesita. |
| `handler_test.go` | Pruebas de la operación y de su entrada REST, con dobles de la persistencia. |
| `rest.go` | Adaptador REST: lee la petición, llama al handler y construye la respuesta. Es el equivalente a un controller. |
| `mcp.go` | Adaptador MCP: declara la herramienta y su esquema, y llama al handler. Sus pruebas van en `mcp_test.go`. |

### Convenciones de nombres

| Elemento | Nombre | Ejemplo |
| --- | --- | --- |
| Paquete de la slice | Verbo de la operación, sin separadores | `create`, `list`, `changestatus` |
| Entrada | `Command` en los commands; parámetros simples o `Query` en las queries | `create.Command` |
| Salida | `Result`, o `Item` para los elementos de una lista | `get.Result` |
| Persistencia requerida | `Repository` en los commands, `Reader` en las queries | `update.Repository` |
| Constructor | `New` | `create.New(repo)` |
| Operación | Método `Handle` del `Handler` | `h.Handle(ctx, cmd)` |
| Adaptador REST | Método `serve`, registrado por `Register(mux)` | `h.serve` |
| Adaptador MCP | Método `callTool`, registrado por `RegisterTool(server)` | `h.callTool` |
| Herramienta MCP | Verbo y agregado en `snake_case` | `create_project` |

En PMT, **handler** siempre significa el `Handle` de `handler.go`, que no conoce HTTP. La función `serve` de `rest.go` se llama **adaptador REST**, aunque en Go se la conozca como handler HTTP.

`Command`, `Result` e `Item` pueden llevar etiquetas `json`, que no crean dependencia con ningún transporte. `mcp.go` define sus propios `toolInput` y `toolOutput`, porque de ellos se infiere el esquema de la herramienta. Los UUID y las fechas se exponen ahí como `string`.

El esquema publicado por MCP se infiere, por lo que su fuente de verdad es el código. `api/features/<feature>/<feature>.mcp.json` lo documenta como un arreglo de herramientas con `name`, `description`, `inputSchema` y `outputSchema`. La prueba `internal/app/contracts_test.go` construye el servidor en memoria, sin PostgreSQL, y falla si una herramienta registrada y su documentación difieren o si una de ellas falta.

### Command o query

| | Command | Query |
| --- | --- | --- |
| Propósito | Cambia estado | Solo lee |
| Dominio | Carga y guarda el agregado; las reglas viven en `domain/` | No aplica reglas de negocio ni modifica nada |
| Resultado | Lo mínimo que necesite quien llama | Un tipo de lectura propio de la slice, con solo los datos que expone. Puede obtener los datos de `store` como agregado, siempre que no lo modifique |
| Transacción y auditoría | Sí | No |

### Reglas de las slices

- Una slice no importa otra slice. Si dos slices comparten una regla de negocio, esa regla sube a `domain/`.
- Cada handler define solo las interfaces que usa. `store` las implementa todas.
- La lógica de negocio no vive en `rest.go` ni en `mcp.go`.
- El código técnico compartido va a `platform/`, nunca a un paquete `common` o `utils` de la feature.
- Duplicar código pequeño entre slices es aceptable. Solo se extrae cuando la duplicación represente la misma regla.

## Errores

- Los errores de negocio se crean con `kernel.Invalid`, `kernel.NotFound` o `kernel.Conflict` y se declaran en `domain/`.
- Cada adaptador los entrega a `platform/rest.WriteError`, que responde 400, 404 o 409 con el formato `{"error": "mensaje"}`.
- `store` traduce a errores de negocio las violaciones de la base de datos, con `postgres.IsForeignKeyViolation` y `postgres.IsUniqueViolation`.
- Cualquier otro error se registra en el log y responde 500 con un mensaje genérico, sin exponer su detalle.
- `platform/mcp.ToolError` aplica la misma regla a las herramientas MCP.

## Configuración

- La configuración se lee con Viper desde `config.yaml`, en la raíz de `backend/`. Este archivo contiene secretos y nunca se versiona.
- `config.example.yaml` es la plantilla versionada.
- Las variables de entorno `DATABASE_URL` y `SERVER_ADDR` tienen prioridad sobre `database.url` y `server.addr` del archivo.
- `internal/app/` es el único lugar que lee la configuración.

## Datos

- Las migraciones se escriben en `internal/platform/postgres/migrations/` y se ejecutan con `go tool goose`, que recibe la URL por `DATABASE_URL`.
- `sqlc.yaml` tiene una entrada por cada feature con consultas. Todas leen el esquema de esas migraciones.
- Tras cambiar migraciones o consultas se ejecuta `go tool sqlc generate`.

## Features

Cada feature corresponde a un agregado principal:

| Feature | Contiene |
| --- | --- |
| `projects` | Proyectos. |
| `stages` | Etapas de un proyecto. |
| `tasks` | Tareas, estados, asignaciones y propuestas de cambio. |
| `agents` | Agentes, perfiles y credenciales. |
| `reviews` | Revisiones de CR y hallazgos. |
| `qa` | Planes QAP, casos y ejecuciones de QA. |
| `activity` | Auditoría, comentarios y bloqueos. |
| `bugs` | Bugs de un proyecto. No pertenecen a ninguna etapa. |
| `imports` | Carga de un proyecto o una etapa completos desde un fichero. No tiene agregado propio; ver [Feature de composición](#feature-de-composición-imports). |

Las dependencias entre features van, en general, en este sentido:

```text
reviews, qa → tasks → stages → projects
bugs → projects
tasks → agents
todas → activity
```

Cerrar una etapa necesita conocer sus tareas, en sentido contrario a la flecha `tasks → stages`. Ese caso se resuelve con una interfaz definida por la slice de `stages` e implementada mediante una query de `tasks`, conectada en `feature.go` o `app`. Nunca hay ciclos de importación.

## Reglas de dependencia

```text
cmd → app
app → feature.go de cada feature, kernel, platform, pgx, SDK de MCP
feature.go → sus slices, store, sqlcgen, SDK de MCP
handler.go → domain de su feature, kernel
rest.go → handler.go, platform/rest
mcp.go → handler.go, kernel, platform/mcp, SDK de MCP
store → domain de su feature, kernel, sqlcgen, pgx, platform/postgres
domain → kernel
kernel → nada
platform → kernel
```

- `domain` no importa transporte, persistencia ni frameworks de infraestructura.
- `handler.go` no importa `store`, `sqlcgen`, `platform`, `pgx` ni `net/http`.
- `platform` no importa features.
- `kernel` solo incorpora conceptos cuya semántica compartida esté comprobada.
- `rest.go` y `mcp.go` validan el formato de las entradas. Las invariantes de negocio se verifican en `domain` o en el handler.

### Colaboración entre features

1. La slice consumidora define la interfaz que necesita.
2. La feature proveedora expone un command o una query.
3. `feature.go` o `app` conecta ambas.

Ninguna feature lee las tablas ni importa los detalles internos de otra.

### Feature de composición: `imports`

`imports` es una excepción a las reglas anteriores. Crea un proyecto con sus etapas y tareas, o una etapa con sus tareas, como una unidad, sin agregado propio, y por eso:

- `imports/domain` y los handlers de `importproject` e `importstage` usan el `domain` de `projects`, `stages` y `tasks`, y aplican sus constructores para validar.
- `imports/store` usa el `store` de esas tres features sobre una sola `pgx.Tx`, porque aceptan `sqlcgen.DBTX`. No usa `kernel.Transactor`, aún pendiente.
- `imports/file` lee el fichero con `os` e interpreta el YAML con `gopkg.in/yaml.v3`. Lo comparten ambas slices, que no se importan entre sí; es un paquete propio porque `domain` no conoce formatos de fichero.

Ninguna otra feature importa `imports`, así que no hay ciclos. La excepción se limita a esta feature. El formato del fichero está en `importar_proyectos.md`.

## Identidad y permisos

- Cada agente tiene su propio token, que determina su `Actor`: agente y perfil.
- El usuario humano también es un `Actor`, con su propio perfil.
- El `Actor` se pasa explícitamente a cada handler.
- Cada agente recibe por MCP solo las herramientas de su perfil, lo que además reduce su contexto.
- Ese filtrado no sustituye la autorización: la validación real ocurre siempre en `domain` o en el handler.

Las responsabilidades de cada perfil están definidas en `perfiles.md`.

## Estados de un proyecto

Todo proyecto nace en `planning`. `in_dev` significa que se desarrolla la primera versión; `working`, que esa versión ya terminó. Una tabla de transiciones en `projects/domain` define los cambios permitidos:

| Desde | Puede pasar a |
| --- | --- |
| `planning` | `stand_by`, `stopped`, `cancelled` |
| `stand_by` | `planning`, `in_dev`, `stopped`, `cancelled` |
| `in_dev` | `working`, `stopped`, `cancelled` |
| `working` | `stopped`, `cancelled` |
| `stopped` | `planning`, `stand_by`, `in_dev`, `working`, `cancelled` |
| `cancelled` | `planning` |

Un valor desconocido es `kernel.Invalid`. Una transición no permitida, incluida la del status a sí mismo, es `kernel.Conflict`. El status solo cambia con la slice `projects/commands/changestatus`, no con `PUT /api/projects/{id}`.

## Estados de una etapa

Toda etapa nace en `planning`. Una tabla de transiciones en `stages/domain` define los cambios permitidos:

| Desde | Puede pasar a |
| --- | --- |
| `planning` | `stand_by`, `stopped`, `cancelled` |
| `stand_by` | `planning`, `in_dev`, `stopped`, `cancelled` |
| `in_dev` | `done`, `stopped`, `cancelled` |
| `done` | `in_dev` |
| `stopped` | `planning`, `stand_by`, `in_dev`, `cancelled` |
| `cancelled` | `planning` |

Un valor desconocido es `kernel.Invalid`. Una transición no permitida, incluida la del status a sí mismo, es `kernel.Conflict`. El status solo cambia con la slice `stages/commands/changestatus`, no con `PUT /api/stages/{id}`.

**Regla de cierre:** una etapa solo pasa a `done` si todas sus tareas están en `done`; si no, la operación responde `kernel.Conflict`. Una etapa sin tareas sí puede pasar a `done`, y reabrirla (`done` → `in_dev`) no consulta las tareas. Como `stages` no importa `tasks`, la slice define la interfaz `changestatus.TaskChecker`, que implementa `tasks.Checker` sobre la query `tasks/queries/checkdone`, conectada en `registerFeatures`.

## Estados de una tarea

Toda tarea nace en `backlog`. Estos son sus status:

| Status | Significado |
| --- | --- |
| `backlog` | Definida, aún no liberada. |
| `ready` | Liberada para desarrollo. |
| `in_progress` | En desarrollo. |
| `in_review` | En revisión de código. |
| `changes_requested` | Con correcciones pendientes. |
| `in_qa` | En verificación de QA. |
| `pending_close` | Aprobada por QA, falta el cierre. |
| `done` | Verificada y cerrada. |
| `cancelled` | Ya no se hará. |

`cancelled` aún no tiene transiciones: ninguna fila de la tabla siguiente entra en ese status ni sale de él.

Una tabla de transiciones en `tasks/domain` define qué perfil puede hacer cada cambio:

| Desde | Acción | Perfil | Hacia |
| --- | --- | --- | --- |
| `backlog` | Liberar | TL | `ready` |
| `ready` | Iniciar | Dev | `in_progress` |
| `in_progress` | Entregar | Dev | `in_review` |
| `in_review` | Aprobar | CR | `in_qa` |
| `in_review` | Pedir corrección | CR | `changes_requested` |
| `in_qa` | Pedir corrección | QA | `changes_requested` |
| `changes_requested` | Reentregar | Dev | `in_review` |
| `in_qa` | Aprobar | QA | `pending_close` |
| `pending_close` | Cerrar | TL | `done` |

Dev corresponde a DevFrontend o DevBackend, según la tarea.

## Regla de planning

Un proyecto solo admite estas operaciones, por REST o por MCP, mientras está en `planning`:

| Agregado | REST | MCP |
| --- | --- | --- |
| Proyecto | `DELETE /api/projects/{id}` | `delete_project` |
| Etapa | `PUT` y `DELETE /api/stages/{id}` | `update_stage`, `delete_stage` |
| Tarea | `PUT` y `DELETE /api/tasks/{id}` | `update_task`, `delete_task` |
| Crear etapa | `POST /api/projects/{project_id}/stages` | `create_stage` |
| Crear tarea | `POST /api/stages/{stage_id}/tasks` | `create_task` |

`update_project` es la excepción: fuera de `planning`, `PUT /api/projects/{id}` cambia `name` y `description`, y `Project.Update` responde `kernel.Conflict` si `product_version` difiere de la actual. Enviar el mismo valor es válido.

Fuera de `planning` responden `kernel.Conflict` con `el proyecto no está en planning`. Se comprueba después de confirmar que el elemento existe y antes de validar sus datos. Dentro de `planning` siguen vigentes `el proyecto tiene etapas o bugs, activos o retirados` y `la etapa tiene tareas, activas o retiradas`: lo retirado cuenta, porque sus filas siguen referenciando al padre. Una etapa o tarea con historial responde `la etapa tiene historial de cambios` o `la tarea tiene historial de cambios`, aunque el proyecto haya vuelto a `planning`.

Las etapas en su planning propio son la excepción a esta sección; ver [Planning propio de una etapa](#planning-propio-de-una-etapa).

La regla es `Project.EnsurePlanning`, en `projects/domain`, y la aplican los handlers de `commands/update`, `commands/delete` y `commands/create`. `projects` la usa directamente en `delete`. `stages` define `ProjectChecker`, que implementa `projects.Checker` sobre la query `projects/queries/checkplanning`. `tasks` define `StageChecker`, que implementa `stages.Checker` sobre la query `stages/queries/checktasks`: busca la etapa y pregunta por su proyecto, de modo que `tasks` no conoce `projects`. `checkplanning` solo lee y delega la decisión en `domain`. Todo se conecta en `registerFeatures`.

## Cambios de etapas y tareas

Fuera de `planning`, las etapas y tareas cambian mediante registros inmutables, que dan trazabilidad de cada decisión. `stages` y `tasks` conservan el contenido original: `revise_*` no toca su fila y `retire_*` solo marca `retired_at`. `stage_changes` y `task_changes` guardan solo cambios, cada uno con su contenido completo, su motivo, quién lo pidió y su fecha. Un cambio es una entidad del agregado `Stage` o `Task`, no un agregado propio.

| Tipo | `kind` | Efecto | MCP | REST |
| --- | --- | --- | --- | --- |
| Agregar | `add` | Nace al final, versión 1, salvo el planning propio | `add_stage`, `add_task` | `POST /api/projects/{project_id}/stages/add`, `POST /api/stages/{stage_id}/tasks/add` |
| Editar | `edit` | Nueva versión | `revise_stage`, `revise_task` | `POST /api/stages/{id}/revise`, `POST /api/tasks/{id}/revise` |
| Retirar | `retire` | Se marca `retired_at`, sin borrar nada | `retire_stage`, `retire_task` | `POST /api/stages/{id}/retire`, `POST /api/tasks/{id}/retire` |

- **Versiones:** `version` numera los cambios desde 1; una tarea con dos cambios tiene exactamente 2 registros. Lo agregado con `add_*` tiene su registro `add` como versión 1. Solo se versionan nombre o título y descripción.
- **Lectura:** toda lectura devuelve la última versión: el contenido del cambio más reciente, o el de la fila si no tiene cambios. `GetStage`, `GetTask`, `ListStages`, `ListTasks` y `UpdateStageStatus` lo resuelven con `LEFT JOIN LATERAL`. `status`, `position` y `updated_at` salen siempre de la fila. La fila solo se escribe en `planning` o en el planning propio, y sin cambios, porque `update_*` y `delete_*` rechazan lo que tiene historial.
- **Historial:** `list_stage_changes` (`GET /api/projects/{project_id}/stage-changes?stage_id=`) y `list_task_changes` (`GET /api/stages/{stage_id}/task-changes?task_id=`). Para cada elemento con al menos un cambio devuelven `original` (contenido de la fila, con `created_at`) y `changes` en orden de `version`. Se consulta por padre porque lo retirado no aparece en `list_*`. Sin registros responde lista vacía.
- **Retirado:** `GetStage`, `GetTask`, `ListStages` y `ListTasks` lo excluyen, así que responde `NotFound` en todas las operaciones y el retiro no se deshace. Su `status` y su `position` no cambian. `position` nunca se renumera ni se reutiliza.
- **Cierre:** las tareas retiradas no cuentan para pasar a `done`; las agregadas nacen en `backlog` y sí cuentan. Agregar a una etapa en `done` no la reabre.
- **Retirar una etapa:** con tareas activas responde `kernel.Conflict`. `stages` define `retire.TaskChecker`, que implementa `tasks.Checker` sobre `tasks/queries/hasactive`.
- **Status del proyecto:** `add_*`, `revise_*` y `retire_*` funcionan con cualquiera, salvo sobre una etapa en su planning propio. Así, un elemento con historial se sigue pudiendo cambiar si el proyecto vuelve a `planning`.
- **Inmutabilidad:** triggers de PostgreSQL rechazan `UPDATE`, `DELETE` y `TRUNCATE` en ambas tablas, y `ON DELETE RESTRICT` impide borrar un elemento con historial. No hay consultas, comandos ni rutas que los modifiquen.
- **Transacción:** `store/changes.go` escribe el registro, y en `retire_*` también `retired_at`, con `pgx.BeginFunc`, porque `kernel.Transactor` sigue pendiente. Bloquea el elemento con `FOR UPDATE` para numerar las versiones.

## Planning propio de una etapa

Una etapa pedida con el proyecto en `working` se planea por sí sola, sin que el proyecto cambie de status. Está en **planning propio** si su fila tiene `add_reason` y `add_requested_by`: el status `planning` no basta, porque también lo tienen las etapas definidas en el `planning` del proyecto, que siguen sin poder editarse. `Stage.InOwnPlanning` hace la distinción.

`add_stage` e `import_stage` guardan motivo y solicitante en la fila y no escriben en `stage_changes`, porque el registro es inmutable y `ON DELETE RESTRICT` impediría editar o eliminar la etapa. Al salir, `Changes.LeavePlanning` escribe en una transacción el `add` versión 1 con el contenido definitivo, cambia el status y vuelve a `NULL` ambas columnas. Por eso `changed_at` del `add` es la fecha de salida, y una etapa eliminada en su planning propio no deja rastro. Con el proyecto en otro status, `add_stage` escribe el `add` de inmediato.

Cualquier transición desde `planning` es la salida, y es definitiva: si la etapa vuelve al status `planning`, sigue bajo historial.

| Operación | En planning propio | Después de salir |
| --- | --- | --- |
| `update_stage`, `delete_stage` | Con proyecto en `working` | `el proyecto no está en planning`, o `la etapa tiene historial de cambios` si el proyecto volvió a `planning` |
| `create_task`, `update_task`, `delete_task` | Con proyecto en `working`, sin registros | Rechazadas, aunque el proyecto vuelva a `planning` |
| `add_task`, `revise_*`, `retire_*` | `la etapa está en su propio planning` | Permitidas |
| `change_stage_status` | Escribe el `add` al salir | Sin cambios |

- Fuera de `working`, las operaciones permitidas responden `el proyecto no está en working`, aunque el proyecto vuelva a `planning`. `change_stage_status` sigue disponible, así que la etapa no queda atascada.
- Las tareas creadas en esta fase quedan como originales en su fila, sin registros.
- Toda etapa con registro `add` rechaza `create_task`, `update_task` y `delete_task`, aunque el proyecto vuelva a `planning`. `add_reason` y `add_requested_by` aparecen en las salidas solo durante el planning propio.
- `tasks` lo consulta con `StageChecker`: `EnsureTasksEditable` en `create`, `update` y `delete`, y `EnsureHistoryRule` en `add`, `revise` y `retire`. `stages` pide `IsWorking` y `EnsureWorking` a `projects.Checker`, sobre la query `projects/queries/checkworking`.
- Si `update_*` coincide con la salida, el registro gana en la lectura. Aplica la regla de [Concurrencia](#concurrencia).

## Bugs

Un bug pertenece a un proyecto, sin etapa: `tasks` no cambia y los bugs no afectan al cierre de etapas. Guarda `steps_to_reproduce` y `product_version`, la versión donde apareció, de texto libre y sin compararla con la del proyecto.

| Comando | REST | Status del proyecto |
| --- | --- | --- |
| `create_bug` | `POST /api/projects/{project_id}/bugs` | Solo `working` |
| `update_bug` | `PUT /api/bugs/{id}` | Solo `working` |
| `list_bugs` | `GET /api/projects/{project_id}/bugs` | Cualquiera |

Fuera de `working` responden `el proyecto no está en working`. `update_bug` corrige el reporte y no cambia el status. No hay comando para eliminar un bug ni para cambiar su status. `delete_project` se bloquea si el proyecto tiene bugs.

Todo bug nace en `reported`. Estos son sus status, por ahora sin transiciones:

| Status | Significado |
| --- | --- |
| `reported` | Reportado, aún sin confirmar. |
| `confirmed` | Reproducido, pendiente de corregir. |
| `in_progress` | En corrección. |
| `in_review` | En revisión de código. |
| `changes_requested` | Con correcciones pendientes. |
| `in_qa` | En verificación de QA. |
| `done` | Corregido y verificado. |
| `rejected` | No se corregirá: no se reproduce, está duplicado o no es un bug. |

## Transacciones

- Los commands usan la interfaz `kernel.Transactor`, con `WithinTx(ctx, fn)`, implementada en `platform/postgres`.
- Un command que cruza features comparte la misma transacción mediante `ctx`.
- La auditoría se escribe en la misma transacción que el cambio, por lo que no puede haber cambios sin registro.

## Concurrencia

- Ningún agregado tiene control de versión: si dos escrituras coinciden, prevalece la última. La `product_version` de `Project` es la versión del producto, de texto libre, y no es control de concurrencia.
- Las operaciones críticas aceptan `idempotency_key`, única por actor, para que un reintento no duplique acciones.

## Eventos

La primera versión no tiene bus de eventos. Los agentes piden trabajo con una herramienta como `my_next_task`. La tabla de `activity` puede servir después como fuente de eventos si hace falta notificarlos.

## Ejemplo: QA aprueba una tarea

La operación es la slice `qa/commands/approve/`.

1. `mcp.go` recibe la llamada y `platform/mcp` resuelve el `Actor`.
2. `handler.go` abre la transacción y valida que el `Actor` sea QA y que el plan esté aprobado.
3. Registra la aprobación y pide a `tasks`, mediante una interfaz, la transición a `pending_close`.
4. `tasks` valida la tabla de transiciones y guarda.
5. `activity` registra la acción y la transacción confirma todo junto.

## Estado actual

Este documento describe el destino de la arquitectura. Hoy existe lo siguiente:

- **Implementado:** configuración, conexión a PostgreSQL, servidor REST, servidor MCP en `/mcp`, errores de negocio, la feature `projects` con su CRUD por REST, su `status` con tabla de transiciones, su `product_version` y las herramientas MCP `create_project`, `list_projects`, `update_project`, `delete_project` y `change_project_status`, la feature `stages` con su CRUD por REST, su `status` con tabla de transiciones y regla de cierre, y las herramientas MCP `create_stage`, `list_stages`, `update_stage`, `delete_stage` y `change_stage_status`, y la feature `tasks` con su CRUD por REST y las herramientas MCP `create_task`, `list_tasks`, `update_task` y `delete_task`, la feature `imports` con las herramientas MCP `import_project` e `import_stage`, y la feature `bugs` con su REST y las herramientas MCP `create_bug`, `update_bug` y `list_bugs`. Crear, editar y eliminar etapas y tareas, y eliminar proyectos, exige que el proyecto esté en `planning`; fuera de esa fase, etapas y tareas cambian con `add_*`, `revise_*` y `retire_*`, con historial en `list_stage_changes` y `list_task_changes`, salvo las etapas en su planning propio. El `status` de una tarea o de un bug se devuelve, pero aún no se puede cambiar.
- **Pendiente:** `Actor`, `Profile`, `Transactor`, autenticación, auditoría, filtrado de herramientas por perfil, la tabla de transiciones de `tasks` y las demás features.
- **Excepción temporal:** los commands de `projects`, `stages`, `tasks` y `bugs` no tienen autorización, auditoría ni, salvo los de cambios, transacción. Deben incorporarlas cuando existan `agents` y `activity`. Hasta entonces, el servidor escucha por defecto solo en `127.0.0.1` y no debe exponerse fuera de la máquina local.
- **Carencia conocida:** `store` no tiene pruebas automáticas contra PostgreSQL.

## Orden de construcción

1. `projects` y `stages`. Hechos, salvo la excepción temporal.
2. `agents` con autenticación por token, y `Actor` y `Profile` en `kernel`.
3. Tabla de transiciones de `tasks`. Su CRUD ya existe, adelantado a `agents`.
4. `activity`, integrada a las transiciones.
5. Primera operación completa por MCP, con una prueba de dos agentes en conflicto.
6. `reviews` y `qa`.

Una operación debe funcionar de entrada a salida antes de extender el patrón a otras.

## Procedimiento para implementar una operación

1. Decide si es un command o una query, y a qué feature pertenece.
2. Define entradas, resultados, errores y reglas, y documenta el contrato REST en `api/features/<feature>/`.
3. Crea la carpeta de la slice en `commands/<operacion>/` o `queries/<operacion>/`.
4. Si la operación necesita reglas nuevas, impleméntalas en `domain/`.
5. Implementa `handler.go`, con sus interfaces mínimas, autorización y límites transaccionales.
6. Añade la persistencia en `store/` y las migraciones necesarias.
7. Añade `rest.go` o `mcp.go`.
8. Registra la slice en `feature.go`.
9. Si la feature es nueva, regístrala en `registerFeatures`, en `internal/app/app.go`. Si no, sus rutas y herramientas no se publican.
10. Si añades o cambias una herramienta MCP, actualiza `api/features/<feature>/<feature>.mcp.json` con lo que publica el servidor.
11. Ejecuta `go test` y comprueba reglas de negocio, persistencia y correspondencia con el contrato.

## Criterios de aceptación

Una implementación está lista cuando:

- La operación puede seguirse desde su entrada hasta su resultado.
- Las dependencias respetan las reglas de este documento y las slices no se importan entre sí.
- Los permisos se validan en `domain` o en el handler, y no solo en el transporte.
- Los commands protegen contra conflictos cuando corresponde y dejan auditoría.
- Las queries no modifican estado.
- Las pruebas pasan.

## Supuestos

- **PostgreSQL:** elegida por su concurrencia y sus restricciones fuertes.
- **Una sola organización:** no hay multi-tenencia. Agregarla después obligaría a tocar casi todas las tablas.
- **Login del usuario humano:** el detalle queda pendiente de definir.
