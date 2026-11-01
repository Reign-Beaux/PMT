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
| `internal/kernel/` | Conceptos compartidos con la misma semántica en varias features: `Actor`, `Profile`, `Team`, `Transactor`, identificadores y errores de negocio (`Invalid`, `NotFound`, `Conflict`), que `platform/rest` traduce a HTTP. |
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

- Los errores de negocio se crean con `kernel.Invalid`, `kernel.NotFound` o `kernel.Conflict` y se declaran en `domain/`; `kernel.Forbidden` marca los de permiso. Los de transición comunes a varias features viven en `kernel`.
- Cada adaptador los entrega a `platform/rest.WriteError`, que responde 400, 403, 404 o 409 con el formato `{"error": "mensaje"}`.
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
| `agents` | Agentes, perfiles y tokens. Sin REST ni MCP: se administra con `pmt-server agent`. |
| `reviews` | Hallazgos de CR y QA sobre tareas y bugs. |
| `qa` | Planes QAP, casos y ejecuciones de QA. |
| `activity` | Auditoría, comentarios y bloqueos. |
| `bugs` | Bugs de un proyecto. No pertenecen a ninguna etapa. |
| `imports` | Carga de un proyecto o una etapa completos desde un fichero. No tiene agregado propio; ver [Feature de composición](#features-de-composición-imports-y-workflow). |
| `workflow` | Operaciones que mueven varios agregados en una transacción: `release_project`, `start_stage` y el cierre y el retiro de una etapa. No tiene agregado propio. |

Las dependencias entre features van, en general, en este sentido:

```text
reviews, qa → tasks → stages → projects
reviews → bugs → projects
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

### Features de composición: `imports` y `workflow`

`imports` es una excepción a las reglas anteriores. Crea un proyecto con sus etapas y tareas, o una etapa con sus tareas, como una unidad, sin agregado propio, y por eso:

- `imports/domain` y los handlers de `importproject` e `importstage` usan el `domain` de `projects`, `stages` y `tasks`, y aplican sus constructores para validar.
- `imports/store` usa el `store` de esas tres features sobre una sola `pgx.Tx`, porque aceptan `sqlcgen.DBTX`. No usa `kernel.Transactor`, aún pendiente.
- `imports/file` lee el fichero con `os` e interpreta el YAML con `gopkg.in/yaml.v3`. Lo comparten ambas slices, que no se importan entre sí; es un paquete propio porque `domain` no conoce formatos de fichero.

Ninguna otra feature importa `imports`, así que no hay ciclos. El formato del fichero está en `importar_proyectos.md`.

`workflow` usa la misma excepción, y solo estas dos features la tienen:

- `workflow/domain` declara qué mueve cada operación. Las reglas siguen en el `domain` de cada feature, incluidos su tabla de perfiles y su ámbito.
- `workflow/store` usa el `store` de `projects`, `stages` y `tasks` sobre una sola `pgx.Tx`. `pgx.Tx` tiene `Begin`, así que `NewStatusChanges(tx)` funciona: sus `BeginFunc` internos son savepoints y conservan el `FOR UPDATE` y la comparación de origen.
- `stages/commands/changestatus` cierra una etapa con `CloseStage`, y `stages/commands/retire` la retira con `RetireStage`, que `app` conecta con `workflow.NewCloser`: `stages` no importa `workflow`.
- El bloqueo sigue siempre el orden proyecto, etapa y tareas por ID.

## Identidad y permisos

- Cada agente tiene su propio token, que determina su `Actor`: agente y perfil.
- El usuario humano también es un `Actor`, con su propio token y el perfil `User`.
- El `Actor` se pasa explícitamente a cada handler, en el `Command`. Nunca es argumento de una herramienta ni del cuerpo REST.
- Cada agente recibe por MCP solo las herramientas de su perfil, lo que además reduce su contexto.
- Ese filtrado no sustituye la autorización: la validación real ocurre siempre en `domain` o en el handler.

Las responsabilidades de cada perfil están definidas en `perfiles.md`.

### Tokens

`agents` guarda en una fila nombre, perfil, `token_hash` y `revoked_at`. El token son 32 bytes aleatorios en hexadecimal y solo se guarda su SHA-256, suficiente por su entropía. El perfil no cambia. Un agente no se borra, se revoca, porque el historial conserva su nombre.

Los tokens se administran solo por línea de comandos, para que ningún agente pueda emitirlos. Se ejecutan desde `backend/`, leen `config.yaml` o `DATABASE_URL` sin escribirlos, y `create` y `rotate` escriben solo el token en la salida estándar:

```text
pmt-server agent create --name NOMBRE --profile PERFIL
pmt-server agent rotate --name NOMBRE
pmt-server agent revoke --name NOMBRE
pmt-server agent list
```

`internal/app/app.go` monta `/api/` y `/mcp` con `rest.Authenticate` y `mcp.Authenticate`. Ambas reciben un `kernel.Authenticator`, que implementa `agents/queries/authenticate`, de modo que `platform` no importa features. MCP usa `auth.RequireBearerToken` del SDK y lleva el `Actor` en `TokenInfo`, de donde lo lee `platformmcp.ActorFrom(req)`. REST lo lleva en el contexto, de donde lo lee `rest.ActorFrom(ctx)`.

**Los clientes:** `.mcp.json` está versionado y compartido, así que solo contiene `Authorization: Bearer ${PMT_TOKEN:-}`. Cada sesión se lanza con su token en `PMT_TOKEN` y la cabecera se fija al arrancar. Los tokens viven fuera del repo, en `~/.config/pmt/tokens/<nombre>`, con permiso 600. REST recibe la misma cabecera.

**Sin token:** un token inválido o revocado responde 401. Sin cabecera, o con `Bearer` vacío, que es lo que envía Claude Code si falta `PMT_TOKEN`, actúa el usuario: `kernel.Owner`, `Actor{Name: "saul", Profile: User}`, sin fila en `agents`. Por eso `saul` no puede ser TL, miembro de un equipo ni dueño de una tarea.

El token evita errores y deja rastro; no detiene a un agente hostil, porque cualquier proceso con shell en esta máquina puede leer esos ficheros.

## Estados de un proyecto

Todo proyecto nace en `planning`. `in_dev` significa que se desarrolla la primera versión; `working`, que esa versión ya terminó. Una tabla de transiciones en `projects/domain` define los cambios permitidos:

| Desde | Puede pasar a |
| --- | --- |
| `planning` | `stand_by`, `stopped`, `cancelled` |
| `stand_by` | `planning`, `in_dev`, `stopped`, `cancelled` |
| `in_dev` | `stopped`, `cancelled` |
| `working` | `stopped`, `cancelled` |
| `stopped` | `planning`, `stand_by`, `in_dev`, `working`, `cancelled` |
| `cancelled` | `planning` |

Un valor desconocido es `kernel.Invalid`. Una transición no permitida, incluida la del status a sí mismo, es `kernel.Conflict`. El status solo cambia con la slice `projects/commands/changestatus`, no con `PUT /api/projects/{id}`.

**Perfiles:** PP y `User` hacen todas las transiciones. PC solo detiene y reanuda. Detener, cancelar y reabrir exigen motivo.

Cambiar el status depende de las etapas; ver [Reglas entre status](#reglas-entre-status).

**Movimientos automáticos.** Ocurren en la transacción que los provoca, sin consultar la tabla de perfiles. El actor es quien hizo la operación: el TL en `start_stage` y al cerrar, y quien canceló o retiró la etapa en esos casos:

| Movimiento | Cuándo | Motivo |
| --- | --- | --- |
| `stand_by → in_dev` | `start_stage`, si el proyecto sigue en `stand_by`. `change_project_status` también lo hace, para PP y `User`, con la regla 12 | `inicio de la etapa <nombre>` |
| `in_dev → working` | Al pasar a `done` una etapa, o al cancelar o retirar una etapa de la versión 1, si las etapas de la **versión 1** cumplen la regla 13: todas en `done`, `cancelled` o retiradas, con al menos una en `done`. Si ninguna queda en `done`, el proyecto no cambia. Nunca es manual | `cierre de la versión 1` |

La versión 1 son las etapas de la planeación original con PP, las que no tienen registro `add`. Las agregadas no cuentan, aunque se agreguen antes de `working`, y una etapa en planning propio tampoco. Cancelar o retirar una etapa agregada no mueve el proyecto. `CloseStage` y `RetireStage` bloquean primero el proyecto, para que dos cierres simultáneos se vean entre sí. `stopped → working` sigue siendo manual y evalúa las mismas etapas; ver [Reglas entre status](#reglas-entre-status).

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

**Perfiles:** `stand_by → in_dev` e `in_dev → done` son del TL de la etapa; `done → in_dev`, de PP, `User` o ese TL, con motivo. PP y `User` liberan, devuelven, cancelan y reabren. PP, `User`, PC y el TL detienen y reanudan. Detener, cancelar y reabrir exigen motivo.

**Regla de cierre:** una etapa solo pasa a `done` si todas sus tareas activas están en `done` o `cancelled`, con al menos una en `done`; si no, la operación responde `kernel.Conflict`. Una etapa vacía o toda cancelada no cierra: se cancela. Reabrirla (`done` → `in_dev`) no consulta las tareas, pero cumple la regla 5. Como `stages` no importa `tasks`, la slice define la interfaz `changestatus.TaskChecker`, que implementa `tasks.Checker.EnsureClosable` sobre el recuento de `tasks/queries/checkdone`, conectada en `registerFeatures`. Todo paso a `done` o `cancelled` pasa por `CloseStage`, salvo la salida del planning propio; ver [movimientos automáticos](#estados-de-un-proyecto).

## Estados de una tarea

Toda tarea nace en `backlog`. Estos son sus status:

| Status | Significado |
| --- | --- |
| `backlog` | Definida, aún no liberada. |
| `ready` | Liberada para desarrollo. |
| `in_dev` | En desarrollo. |
| `in_review` | En revisión de código. |
| `changes_requested` | Con correcciones pendientes. |
| `in_qa` | En verificación de QA. |
| `pending_close` | Aprobada, falta el cierre del TL. |
| `done` | Verificada y cerrada. |
| `cancelled` | Ya no se hará. |

Una tabla de transiciones en `tasks/domain` define los cambios permitidos:

| Desde | Puede pasar a |
| --- | --- |
| `backlog` | `ready`, `cancelled` |
| `ready` | `in_dev`, `cancelled` |
| `in_dev` | `in_review`, `cancelled` |
| `in_review` | `in_qa`, `changes_requested`, `pending_close`, `cancelled` |
| `changes_requested` | `in_dev`, `cancelled` |
| `in_qa` | `pending_close`, `changes_requested`, `cancelled` |
| `pending_close` | `done`, `in_review`, `in_qa`, `cancelled` |
| `done` | ninguna |
| `cancelled` | `backlog` |

Un valor desconocido es `kernel.Invalid`. Una transición no permitida, incluida la del status a sí mismo, es `kernel.Conflict`. El status solo cambia con la slice `tasks/commands/changestatus` (`change_task_status`, `PUT /api/tasks/{id}/status`), no con `PUT /api/tasks/{id}`.

Cambiar el status depende del proyecto y de la etapa; ver [Reglas entre status](#reglas-entre-status). No es un cambio de contenido: no escribe en `task_changes`. Actualiza `status` y `updated_at`, deja su registro en el [historial de status](#historial-de-status) y devuelve la última versión. Una tarea retirada responde `NotFound`.

**Perfiles:** cada transición exige un perfil, en `domain/permissions.go`. Una prueba exige que toda transición tenga perfiles. `User` tiene lo de PP y PC; Dev es DevFrontend o DevBackend.

| Desde | Acción | Perfil | Hacia |
| --- | --- | --- | --- |
| `backlog` | Liberar | TL | `ready` |
| `ready` | Tomar | Dev del equipo | `in_dev` |
| `in_dev` | Entregar | Dev dueño | `in_review` |
| `in_review` | Aprobar | CR | `in_qa` |
| `in_review` | Pedir corrección | CR | `changes_requested` |
| `in_review` | Aprobar un cambio interno, con motivo | CR | `pending_close` |
| `in_qa` | Pedir corrección | QA | `changes_requested` |
| `changes_requested` | Retomar | Dev dueño | `in_dev` |
| `in_qa` | Aprobar | QA | `pending_close` |
| `pending_close` | Cerrar | TL | `done` |
| `pending_close` | Devolver, con motivo | TL | `in_review`, `in_qa` |
| cualquiera menos `done` | Cancelar, con motivo | TL, PP, `User` | `cancelled` |
| `cancelled` | Reabrir, con motivo | TL, PP, `User` | `backlog` |

La validación sigue este orden: status conocido, transición, perfil, ámbito, motivo y reglas entre status. El perfil responde `kernel.Forbidden` antes de consultar otra feature. Solo espacios cuenta como motivo vacío.

**Criterio de QA:** un cambio pasa por QA si altera algo observable desde fuera. Si es interno, CR lo lleva a `pending_close` con motivo. El TL decide si lo devuelve a CR o a QA, sin importar por dónde llegó; CR o QA deciden después si va a `changes_requested`.

`revise_task` también mueve el status: una tarea en `in_review`, `in_qa` o `pending_close` vuelve a `changes_requested`, porque si no se aprobaría algo distinto a lo pedido. Lo hace `Task.ReviseStatus`, con el status leído tras `FOR UPDATE`, conserva el dueño, deja un registro de status con quien revisó como actor y aplica aunque la tarea esté congelada. Una tarea en `done` responde `kernel.Conflict` con `una tarea en done no se revisa: usa add_task`. En los demás status no cambia.

## Asignación

Cada etapa tiene un TL responsable y un equipo de devs. `stages.lead`, `stage_members.agent` y `tasks.assignee` referencian `agents (name)`, que es único y no cambia: así las salidas muestran el nombre sin que otra feature lea `agents`. No hay historial de asignación: quién tomó una tarea queda en `task_status_changes`.

| Comando MCP | REST | Quién | Efecto |
| --- | --- | --- | --- |
| `assign_stage_lead` | `PUT /api/stages/{id}/lead` | PP, `User` | Fija o reemplaza el TL. El agente debe tener perfil TL |
| `add_stage_member` | `POST /api/stages/{id}/members` | TL de la etapa | Añade un dev |
| `remove_stage_member` | `DELETE /api/stages/{id}/members/{agent}` | TL de la etapa | `Conflict` si tiene tareas abiertas en la etapa |
| `assign_task` | `PUT /api/tasks/{id}/assignee` | TL de la etapa | Pone o cambia el dueño de una tarea ya tomada. `Conflict` en `backlog`, `ready`, `done` y `cancelled` |

- **Tomar no es un comando:** `ready → in_dev` deja `assignee` con el nombre del dev, que debe ser del equipo. Después solo ese dev la mueve. `cancelled → backlog` lo limpia.
- Un dev puede tener varias tareas sin límite fijo. CR, QA y QAP actúan por perfil.
- Un TL solo actúa sobre su etapa y sus tareas. PP, PC y `User` no tienen ámbito.
- Solo se asignan agentes activos. Lo ya asignado a un agente revocado no cambia: PP o el TL lo reasignan.
- Las cuatro operaciones aplican en cualquier status y en el planning propio, porque no son cambios de contenido. Lo retirado responde `NotFound`. `stage_members` se borra en cascada con la etapa, para no impedir `delete_stage`.
- `import_*` crea sin TL, sin equipo y sin dueño.
- `tasks` pide `Team` a `stages.Checker`; `stages` pide `HasOpenAssigned` a `tasks.Checker`, y ambas piden `Profile` a `agents.Checker`, sobre `agents/queries/lookup`.

## Reglas entre status

Las comprueba el handler después del dominio y antes de escribir. Ninguna escribe en los hijos: detener o cancelar un padre congela lo de abajo porque la regla lo consulta, sin cambiar su status. Una **etapa o tarea activa** es la que no está retirada ni cancelada. Todas responden `kernel.Conflict`.

| Reglas | Slice | Comprobación |
| --- | --- | --- |
| 1, 7, 8, 9 | `tasks/commands/changestatus` | Avanzar exige proyecto en `in_dev` o `working`, y etapa en `in_dev` |
| 2, 3, 8, 9 | Igual | Cancelar y reabrir exigen proyecto ni `stopped` ni `cancelled`, y etapa ni `done` ni `cancelled`. Con la etapa en `stopped` sí se puede |
| 4 | `stages/commands/changestatus` | `planning → stand_by`: al menos una tarea activa |
| 5 | Igual | `→ in_dev`, desde cualquier origen: proyecto en `in_dev` o `working` |
| 10 | Igual | Con proyecto `stopped` o `cancelled`: solo `→ stopped` o `→ cancelled` |
| 11 | `projects/commands/changestatus` | `planning → stand_by`: al menos una etapa activa y ninguna en `planning` |
| 12 | Igual | `stand_by → in_dev`: al menos una etapa en `stand_by` |
| 13, 14 | Igual | `→ working`: entre las etapas de la versión 1, al menos una en `done` y las demás sin retirar en `done` o `cancelled`, así que una detenida lo impide. Las agregadas no cuentan |
| 15 | `bugs/commands/changestatus` | Toda transición: proyecto en `working` |

- Las reglas 4, 11 y 12 solo valen desde el origen que nombran, y no desde `stopped`.
- `stopped → working` solo aplica la regla 13 si el proyecto nunca estuvo en `working`, según `project_status_changes`, y evalúa solo las etapas de la versión 1, igual que el paso automático. Un proyecto que llegó antes de la fase 1 no tiene ese registro y cumple la regla.
- Las etapas en planning propio cumplen las reglas 4 y 10 al salir: las comprobaciones van antes de `LeavePlanning`.
- Las reglas 16 y 17 ya estaban vigentes: los bugs no afectan al cierre, y `add_task` en una etapa `done` no la reabre.
- **Colaboración:** `projects.Checker` da `AcceptsWork` e `IsFrozen` (`queries/checkstatus`); `stages.Checker`, `EnsureWorkOpen` y `EnsureTaskCancellable` (`queries/checktasks`); `tasks.Checker`, `EnsureReleasable` (`queries/checkrelease`). `stages.NewProgress` da `EnsureReleasable`, `EnsureStartable` y `EnsureClosed` (`queries/checkprogress`); no depende de `projects`, así que `registerFeatures` lo construye antes y lo pasa a `projects.Register`.

## Lotes

Son todo o nada, en una transacción, con un registro de status por elemento.

| Comando MCP | REST | Quién | Efecto |
| --- | --- | --- | --- |
| `release_project` | `POST /api/projects/{id}/release` | PP, `User` | Pasa a `stand_by` un proyecto en `planning` y sus etapas en `planning`. Exige una etapa activa, y cada etapa cumple la regla 4; el error la nombra. Las demás etapas no se tocan, las tareas siguen en `backlog`, y una etapa en planning propio sale con su `add` |
| `start_stage` | `POST /api/stages/{id}/start` | TL de la etapa | Exige etapa en `stand_by` y proyecto en `stand_by`, `in_dev` o `working`. Pasa la etapa a `in_dev`, sus tareas en `backlog` a `ready` y el proyecto a `in_dev` si sigue en `stand_by`. `change_stage_status stand_by → in_dev` sigue disponible y no toca tareas |
| `change_tasks_status` | `PUT /api/stages/{stage_id}/tasks/status` | Según cada transición | Un `status` y un `reason` para varias tareas de una etapa, sin repetir. Cada una valida lo mismo que `change_task_status` y puede tener otro origen |

`change_tasks_status` es una slice de `tasks`; los otros dos, de `workflow`. Bloquea con `WHERE id = ANY($1) ORDER BY id FOR UPDATE`, para que dos lotes no se bloqueen entre sí, y decide con lo leído tras bloquear. Lista vacía o repetida: `Invalid`. Tarea de otra etapa o retirada: `NotFound`. El error conserva su tipo con `kernel.Prefix` y antepone `tarea <id>: `. Los bugs no tienen lote.

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
- **Lectura:** toda lectura devuelve la última versión: el contenido del cambio más reciente, o el de la fila si no tiene cambios. `GetStage`, `GetTask`, `ListStages`, `ListTasks`, `UpdateStageStatus` y `UpdateTaskStatus` lo resuelven con `LEFT JOIN LATERAL`. `status`, `position` y `updated_at` salen siempre de la fila. La fila solo se escribe en `planning` o en el planning propio, y sin cambios, porque `update_*` y `delete_*` rechazan lo que tiene historial.
- **Historial:** `list_stage_changes` (`GET /api/projects/{project_id}/stage-changes?stage_id=`) y `list_task_changes` (`GET /api/stages/{stage_id}/task-changes?task_id=`). Para cada elemento con al menos un cambio devuelven `original` (contenido de la fila, con `created_at`) y `changes` en orden de `version`. Se consulta por padre porque lo retirado no aparece en `list_*`. Sin registros responde lista vacía.
- **Retirado:** `GetStage`, `GetTask`, `ListStages` y `ListTasks` lo excluyen, así que responde `NotFound` en todas las operaciones y el retiro no se deshace. Su `status` y su `position` no cambian. `position` nunca se renumera ni se reutiliza.
- **Cierre:** las tareas retiradas no cuentan para pasar a `done`; las agregadas nacen en `backlog` y sí cuentan. Agregar a una etapa en `done` no la reabre.
- **Retirar una etapa:** con tareas activas responde `kernel.Conflict`. `stages` define `retire.TaskChecker`, que implementa `tasks.Checker` sobre `tasks/queries/hasactive`. El retiro pasa por `RetireStage`, de `workflow`, y puede disparar el paso a `working`; ver [movimientos automáticos](#estados-de-un-proyecto).
- **Status del proyecto:** `add_*`, `revise_*` y `retire_*` funcionan con cualquiera, salvo sobre una etapa en su planning propio. Así, un elemento con historial se sigue pudiendo cambiar si el proyecto vuelve a `planning`.
- **Inmutabilidad:** triggers de PostgreSQL rechazan `UPDATE`, `DELETE` y `TRUNCATE` en ambas tablas, y `ON DELETE RESTRICT` impide borrar un elemento con historial. No hay consultas, comandos ni rutas que los modifiquen.
- **Transacción:** `store/changes.go` escribe el registro, y en `retire_*` también `retired_at`, con `pgx.BeginFunc`, porque `kernel.Transactor` sigue pendiente. Bloquea el elemento con `FOR UPDATE` para numerar las versiones.

## Historial de status

Cada cambio de status de un proyecto, etapa, tarea o bug deja un registro inmutable: desde, hacia, actor, perfil, fecha y motivo. No es un cambio de contenido: `stage_changes` y `task_changes` guardan contenido, y estas tablas solo status. Crear un elemento no deja registro, y tampoco `import_*`, porque no hay cambio. Los datos anteriores no tienen relleno: un registro retroactivo inventaría actor y fecha, así que el historial empieza con el primer cambio.

| Tabla | MCP | REST |
| --- | --- | --- |
| `project_status_changes` | `list_project_status_changes` | `GET /api/projects/{id}/status-changes` |
| `stage_status_changes` | `list_stage_status_changes` | `GET /api/stages/{id}/status-changes` |
| `task_status_changes` | `list_task_status_changes` | `GET /api/tasks/{id}/status-changes` |
| `bug_status_changes` | `list_bug_status_changes` | `GET /api/bugs/{id}/status-changes` |

- **Sin llave foránea:** las tablas no referencian a su entidad, a diferencia de `stage_changes` y `task_changes`. El usuario pidió que en `planning` se pueda eliminar, y un cambio de status no debe impedirlo: el historial de status no bloquea ningún `delete_*`. Los registros sobreviven al elemento eliminado y se consultan por su ID, por eso `list_*_status_changes` no comprueba que exista y un ID sin registros responde lista vacía.
- **Actor:** se guarda una copia de nombre y perfil, sin FK a `agents`, para conservar el perfil que tenía al actuar.
- **Inmutabilidad:** triggers con `reject_change_mutation` rechazan `UPDATE`, `DELETE` y `TRUNCATE`.
- **Transacción:** `store/status_changes.go` hace `FOR UPDATE`, compara el status con el de origen, escribe el registro y actualiza. Si el status cambió entre leerlo y guardarlo, responde `kernel.Conflict` con `el status cambió, vuelve a consultar`, para que `desde` siempre sea cierto. Por eso la regla general de [Concurrencia](#concurrencia) no aplica a los cambios de status.
- **Motivo:** `reason` se exige en las transiciones que indica cada `domain/permissions.go`; en las demás es opcional.
- **Actor:** `saul`, con perfil `User`, marca lo hecho sin token. `sin-autenticar`, con perfil vacío, solo aparece en registros anteriores a los perfiles. Los registros anteriores a la migración 00015 conservan `in_progress`, que era el valor real al escribirlos.

## Planning propio de una etapa

Una etapa pedida con el proyecto en `working` se planea por sí sola, sin que el proyecto cambie de status. Está en **planning propio** si su fila tiene `add_reason` y `add_requested_by`: el status `planning` no basta, porque también lo tienen las etapas definidas en el `planning` del proyecto, que siguen sin poder editarse. `Stage.InOwnPlanning` hace la distinción.

`add_stage` e `import_stage` guardan motivo y solicitante en la fila y no escriben en `stage_changes`, porque el registro es inmutable y `ON DELETE RESTRICT` impediría editar o eliminar la etapa. Al salir, `Changes.LeavePlanning` escribe en una transacción el `add` versión 1 con el contenido definitivo y el registro de status, cambia el status y vuelve a `NULL` ambas columnas. El motivo del registro es el de `change_stage_status`, no `add_reason`. Por eso `changed_at` del `add` es la fecha de salida, y una etapa eliminada en su planning propio no deja rastro. Con el proyecto en otro status, `add_stage` escribe el `add` de inmediato.

Cualquier transición desde `planning` es la salida, y es definitiva: si la etapa vuelve al status `planning`, sigue bajo historial.

| Operación | En planning propio | Después de salir |
| --- | --- | --- |
| `update_stage`, `delete_stage` | Con proyecto en `working` | `el proyecto no está en planning`, o `la etapa tiene historial de cambios` si el proyecto volvió a `planning` |
| `create_task`, `update_task`, `delete_task` | Con proyecto en `working`, sin registros | Rechazadas, aunque el proyecto vuelva a `planning` |
| `add_task`, `revise_*`, `retire_*` | `la etapa está en su propio planning` | Permitidas |
| `change_stage_status` | Escribe el `add` al salir. Salir a `stand_by` exige una tarea activa | Sin cambios |

- Fuera de `working`, las operaciones permitidas responden `el proyecto no está en working`, aunque el proyecto vuelva a `planning`. `change_stage_status` sigue disponible, así que la etapa no queda atascada; con el proyecto en `stopped` o `cancelled` solo se detiene o se cancela.
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
| `change_bug_status` | `PUT /api/bugs/{id}/status` | Solo `working` |
| `list_bugs` | `GET /api/projects/{project_id}/bugs` | Cualquiera |

Fuera de `working` responden `el proyecto no está en working`. `update_bug` corrige el reporte y no cambia el status. No hay comando para eliminar un bug. `delete_project` se bloquea si el proyecto tiene bugs.

Todo bug nace en `reported`. Estos son sus status:

| Status | Significado |
| --- | --- |
| `reported` | Reportado, aún sin confirmar. |
| `confirmed` | Reproducido, pendiente de corregir. |
| `in_dev` | En corrección. |
| `in_review` | En revisión de código. |
| `changes_requested` | Con correcciones pendientes. |
| `in_qa` | En verificación de QA. |
| `pending_close` | Aprobado, falta el cierre del TL. |
| `done` | Corregido y verificado. |
| `rejected` | No se corregirá: no se reproduce, está duplicado o no es un bug. |

Una tabla de transiciones en `bugs/domain` define los cambios permitidos:

| Desde | Puede pasar a |
| --- | --- |
| `reported` | `confirmed`, `rejected` |
| `confirmed` | `in_dev`, `rejected` |
| `in_dev` | `in_review`, `rejected` |
| `in_review` | `in_qa`, `changes_requested`, `pending_close`, `rejected` |
| `changes_requested` | `in_dev`, `rejected` |
| `in_qa` | `pending_close`, `changes_requested`, `rejected` |
| `pending_close` | `done`, `in_review`, `in_qa`, `rejected` |
| `done` | ninguna |
| `rejected` | `reported` |

Un valor desconocido es `kernel.Invalid`. Una transición no permitida, incluida la del status a sí mismo, es `kernel.Conflict`. Un bug inexistente responde `NotFound`. El status solo cambia con la slice `bugs/commands/changestatus` (`change_bug_status`, `PUT /api/bugs/{id}/status`), no con `update_bug`. Cambiar el status exige el proyecto en `working` (regla 15) y devuelve el bug completo.

**Perfiles:** un bug no tiene etapa, así que todo es por perfil, sin asignación. QA confirma; Dev inicia, entrega y retoma; CR aprueba, pide corrección o lleva un cambio interno a `pending_close`; QA aprueba o pide corrección en `in_qa`; el TL cierra o devuelve a `in_review` o `in_qa`. QA o TL rechazan un bug `reported` y reabren uno `rejected`; solo TL rechaza desde los demás. Rechazar, reabrir, `in_review → pending_close` y devolver exigen motivo.

## Hallazgos

Un hallazgo es algo que CR o QA encontró en una tarea o un bug. Lo guarda `findings`, de la feature `reviews`, con `blocking`, quién lo registró y su perfil. Nace en `open` y termina en `fixed` o `dismissed`; no se reabre.

| Comando MCP | REST | Quién | Efecto |
| --- | --- | --- | --- |
| `create_finding` | `POST /api/findings` | CR, QA | Registra el hallazgo. El elemento debe existir, sin retirar |
| `close_finding` | `PUT /api/findings/{id}/close` | `fixed`: CR, QA. `dismissed`: TL de la etapa, o TL en bugs | Cierra con `resolution` y motivo obligatorio |
| `list_findings` | `GET /api/findings?item_kind=&item_id=&pending=` | Todos | Por fecha. Con `pending=true`, solo los abiertos, bloqueantes primero |

- **No frenan el flujo:** no tocan el status del elemento, y QA puede aprobar con hallazgos abiertos. CR y QA lo devuelven con `change_tasks_status` o `change_bug_status`.
- **Sin llave foránea:** igual que el [historial de status](#historial-de-status), para no impedir `delete_task`. La existencia se valida al crear, y `list_findings` no la comprueba: responde lista vacía.
- **Validación de `close_finding`:** resolución conocida, abierto, perfil, ámbito y motivo. Solo espacios cuenta como motivo vacío. Cerrar uno cerrado responde `kernel.Conflict`; el `UPDATE` exige `status = 'open'`, así que de dos cierres simultáneos el segundo también. `User` no registra ni cierra hallazgos: no tiene lo de CR, QA ni TL.
- **Tarea que ya no existe:** descartar exige saber su etapa, así que responde `NotFound` si la tarea se retiró o se eliminó. `fixed` no consulta la tarea.
- **Colaboración:** `tasks.Checker` da `StageOf`, `stages.Checker` da `Team` y `bugs.Checker` da `Check`, sobre `bugs/queries/get`. Todo se conecta en `registerFeatures`.

## Transacciones

- Los commands usan la interfaz `kernel.Transactor`, con `WithinTx(ctx, fn)`, implementada en `platform/postgres`.
- Un command que cruza features comparte la misma transacción mediante `ctx`.
- La auditoría se escribe en la misma transacción que el cambio, por lo que no puede haber cambios sin registro.

## Concurrencia

- Ningún agregado tiene control de versión: si dos escrituras coinciden, prevalece la última. La `product_version` de `Project` es la versión del producto, de texto libre, y no es control de concurrencia.
- Las operaciones críticas aceptan `idempotency_key`, única por actor, para que un reintento no duplique acciones.

## Eventos

PMT no tiene bus de eventos ni envía avisos. Sus comandos solo afectan la base de datos. Los agentes se avisan entre sí con hooks de herdr, fuera de PMT.

## Ejemplo: QA aprueba una tarea

La operación es la slice `tasks/commands/changestatusbatch/`, `change_tasks_status`, o `change_task_status` para una sola tarea.

1. `mcp.go` recibe la llamada y `platform/mcp` resuelve el `Actor`.
2. `handler.go` pide al store bloquear las tareas por orden de ID.
3. Con lo leído, `domain` valida para cada una la transición `in_qa → pending_close` y que el `Actor` sea QA.
4. El handler consulta a `stages` el equipo y las reglas entre status.
5. El store escribe un registro de status por tarea y el nuevo status, y la transacción confirma todo junto.

## Estado actual

Este documento describe el destino de la arquitectura. Hoy existe lo siguiente:

- **Implementado:** el historial de status, `Actor`, la feature `agents` con sus tokens, la autenticación de `/api/` y `/mcp`, los perfiles de todas las transiciones, la asignación de etapas y tareas, las reglas entre status, los lotes y la feature `workflow`; configuración, conexión a PostgreSQL, servidor REST, servidor MCP en `/mcp`, errores de negocio, la feature `projects` con su CRUD por REST, su `status` con tabla de transiciones, su `product_version` y las herramientas MCP `create_project`, `list_projects`, `update_project`, `delete_project` y `change_project_status`, la feature `stages` con su CRUD por REST, su `status` con tabla de transiciones y regla de cierre, y las herramientas MCP `create_stage`, `list_stages`, `update_stage`, `delete_stage` y `change_stage_status`, y la feature `tasks` con su CRUD por REST, su `status` con tabla de transiciones y las herramientas MCP `create_task`, `list_tasks`, `update_task`, `delete_task` y `change_task_status`, la feature `imports` con las herramientas MCP `import_project` e `import_stage`, y la feature `bugs` con su REST y las herramientas MCP `create_bug`, `update_bug`, `change_bug_status` y `list_bugs`, con `status` y tabla de transiciones, y la feature `reviews` con sus hallazgos, su REST y las herramientas MCP `create_finding`, `close_finding` y `list_findings`. Crear, editar y eliminar etapas y tareas, y eliminar proyectos, exige que el proyecto esté en `planning`; fuera de esa fase, etapas y tareas cambian con `add_*`, `revise_*` y `retire_*`, con historial en `list_stage_changes` y `list_task_changes`, salvo las etapas en su planning propio.
- **Pendiente:** `Transactor`, auditoría, filtrado de herramientas por perfil y las demás features.
- **Excepción temporal:** los commands de `projects`, `stages`, `tasks` y `bugs` no tienen auditoría ni, salvo los de cambios y de status, transacción, y solo los de status y asignación tienen autorización. Deben incorporarlas cuando exista `activity`. Hasta entonces, el servidor escucha por defecto solo en `127.0.0.1` y no debe exponerse fuera de la máquina local.

## Orden de construcción

1. `projects` y `stages`. Hechos, salvo la excepción temporal.
2. `agents` con autenticación por token, y `Actor` y `Profile` en `kernel`. Hecho.
3. Reglas por perfil en las transiciones, con asignación, reglas entre status y lotes. Hecho.
4. `activity`, integrada a las transiciones.
5. Primera operación completa por MCP, con una prueba de dos agentes en conflicto.
6. `reviews`. Hecho.
7. `qa`.

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
