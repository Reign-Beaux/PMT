# Arquitectura del backend

El backend es un **monolito modular en Go organizado por vertical slices**. Un solo proceso, `pmt-server`, es dueño de la base PostgreSQL y sirve todo:

```text
Navegador ── REST /api ─────────┐
           ── WebSocket /api/events ─┤
           ── frontend /  ──────┤
                                ├── pmt-server ── commands y queries ── PostgreSQL
Agentes ───── MCP /mcp ─────────┘                       ▲
                                                        │ NOTIFY pmt_changes
                      listener de eventos ◄─────────────┘
```

REST y MCP llaman a los mismos handlers, así que las reglas de negocio no se duplican.

| Necesidad | Elección |
| --- | --- |
| Lenguaje | Go 1.27 |
| Base de datos | PostgreSQL |
| Acceso a datos | `pgx/v5` y `sqlc` |
| Identificadores | UUID (`github.com/google/uuid`) |
| Configuración | Viper |
| Migraciones | `goose` como librería: embebidas y aplicadas al arrancar |
| MCP | SDK oficial de Go (`modelcontextprotocol/go-sdk`), Streamable HTTP |
| WebSocket | `github.com/coder/websocket` |
| YAML de importación | `gopkg.in/yaml.v3` |
| Pruebas | `testing` estándar: 179 archivos y unas 600 pruebas |

## Estructura

```text
backend/
├── api/features/<feature>/     contratos: <feature>.openapi.yaml (REST) y <feature>.mcp.json (MCP)
├── cmd/pmt-server/             punto de entrada; `pmt-server agent ...` administra tokens
├── internal/
│   ├── app/                    armado de features, rutas, arranque y cierre
│   ├── kernel/                 Actor, Profile, errores de negocio
│   ├── platform/
│   │   ├── rest/               servidor HTTP, autenticación, errores
│   │   ├── mcp/                servidor MCP, autenticación, errores de herramienta
│   │   ├── events/             listener de NOTIFY y WebSocket /api/events
│   │   ├── web/                frontend compilado con fallback SPA
│   │   ├── config/             carga de la configuración; options/ con una sección por archivo
│   │   ├── health/             /healthz y /readyz
│   │   └── postgres/           conexión, errores, migrate.go y migraciones/
│   └── features/<feature>/
│       ├── feature.go          construye y registra las slices
│       ├── domain/             agregado, reglas, transiciones, permisos
│       ├── store/              persistencia; queries/ (SQL) y sqlcgen/ (generado)
│       ├── commands/<op>/      operación que escribe
│       └── queries/<op>/       operación que lee
├── config/                     config.dev.yaml (fuera de git) y config.prod.yaml
├── sqlc.yaml
└── bin/                        binario y log, fuera de git
```

Cada slice es un paquete con `handler.go` (la operación, su autorización y las interfaces mínimas que usa), `handler_test.go`, y `rest.go` o `mcp.go` según por dónde se expone.

### Convenciones

| Elemento | Nombre | Ejemplo |
| --- | --- | --- |
| Paquete de la slice | Verbo sin separadores | `create`, `changestatus` |
| Entrada y salida | `Command` o `Query`; `Result` o `Item` | `create.Command` |
| Persistencia que pide | `Repository` en commands, `Reader` en queries | `update.Repository` |
| Operación | `Handler.Handle` | `h.Handle(ctx, cmd)` |
| Adaptadores | `serve` (REST), `callTool` (MCP) | |
| Herramienta MCP | Verbo y agregado en `snake_case` | `create_project` |

### Reglas de dependencia

- Una slice no importa otra slice. Lo que comparten sube a `domain/`.
- `domain` solo importa `kernel`. `handler.go` no importa `store`, `platform`, `pgx` ni `net/http`.
- `platform` no importa features. `kernel` no importa nada.
- Una feature no lee las tablas de otra: la slice que consume define una interfaz, la otra feature la cumple con una query (los `Checker`), y `registerFeatures` en `internal/app/app.go` las conecta.
- **Excepciones:** `imports` y `workflow` no tienen agregado propio y usan el `domain` y el `store` de `projects`, `stages` y `tasks` sobre una sola transacción. Nadie las importa, así que no hay ciclos.

## Features

| Feature | Contiene |
| --- | --- |
| `projects` | Proyectos, su status, su versión de producto y sus historiales |
| `stages` | Etapas, su status, cambios, TL y equipo |
| `tasks` | Tareas, su status, lotes, cambios y dueño |
| `bugs` | Bugs de un proyecto en `working` |
| `reviews` | Hallazgos de CR y QA |
| `qa` | Planes QAP con versiones y tareas vinculadas |
| `devplans` | Planes de desarrollo con versiones y tareas vinculadas |
| `activity` | Comentarios |
| `imports` | `import_project` e `import_stage` desde YAML ([importar.md](importar.md)) |
| `workflow` | Operaciones sobre varios agregados: `release_project`, `start_stage`, el cierre y el retiro de etapas, y `get_project_progress` |
| `agents` | Agentes y tokens. Sin REST ni MCP: solo `pmt-server agent` |

## Herramientas MCP

Hay 59 herramientas, documentadas en `api/features/<feature>/<feature>.mcp.json`. La prueba `internal/app/contracts_test.go` falla si una herramienta y su documentación difieren. Cada agente recibe la lista completa; los permisos los aplica el handler.

| Feature | Herramientas |
| --- | --- |
| `projects` | `create_project`, `list_projects`, `update_project`, `delete_project`, `change_project_status`, `change_product_version`, `list_project_status_changes`, `list_product_version_changes` |
| `stages` | `create_stage`, `list_stages`, `update_stage`, `delete_stage`, `change_stage_status`, `add_stage`, `revise_stage`, `retire_stage`, `list_stage_changes`, `list_stage_status_changes`, `assign_stage_lead`, `add_stage_member`, `remove_stage_member` |
| `tasks` | `create_task`, `list_tasks`, `update_task`, `delete_task`, `change_task_status`, `change_tasks_status`, `add_task`, `revise_task`, `retire_task`, `list_task_changes`, `list_task_status_changes`, `assign_task` |
| `workflow` | `release_project`, `start_stage`, `get_project_progress` |
| `bugs` | `create_bug`, `update_bug`, `change_bug_status`, `list_bugs`, `list_bug_status_changes` |
| `reviews` | `create_finding`, `close_finding`, `list_findings` |
| `qa` | `create_test_plan`, `revise_test_plan`, `submit_test_plan`, `approve_test_plan`, `return_test_plan`, `list_test_plans`, `list_test_plan_versions` |
| `devplans` | `create_dev_plan`, `revise_dev_plan`, `list_dev_plans`, `list_dev_plan_versions` |
| `activity` | `add_comment`, `list_comments` |
| `imports` | `import_project`, `import_stage` |

**REST** expone lo mismo bajo `/api/`, salvo `imports`, que es solo MCP. La fuente de verdad son los `*.openapi.yaml`. Además hay `GET` de un proyecto, una etapa y una tarea por su ID.

## Identidad

- Cada agente tiene un token: 32 bytes aleatorios en hexadecimal. PMT solo guarda su SHA-256, y el perfil del agente no cambia.
- `/api/` y `/mcp` pasan por la autenticación, que resuelve el `Actor` y lo pasa a cada handler. El actor nunca es un argumento de la herramienta ni del cuerpo.
- Token inválido o revocado: 401. **Sin cabecera, o con `Bearer` vacío, actúa el usuario**: `saul`, perfil `User`. Así entra el navegador.
- Los tokens viven en `~/.config/pmt/tokens/<nombre>`, con permiso 600. Los clientes MCP mandan `Authorization: Bearer ${PMT_TOKEN}`.
- El token evita errores y deja rastro; no protege contra un proceso con acceso a esta máquina. Por eso el servidor escucha en `127.0.0.1`.

## Errores

- Los errores de negocio se crean en `domain` con `kernel.Invalid`, `NotFound`, `Conflict` o `Forbidden`.
- REST responde 400, 404, 409 o 403 con `{"error": "mensaje"}`. Cualquier otro error se registra en el log y responde 500 con un mensaje genérico.
- MCP antepone el tipo para que el agente lo distinga sin interpretar el texto: `invalid: `, `not_found: `, `conflict: ` o `forbidden: `. Cualquier otro sale como `error interno`.
- `store` traduce las violaciones de llave foránea y de unicidad a errores de negocio.

## Datos

- 22 migraciones en `internal/platform/postgres/migrations/`, de `00001_create_projects` a `00022_create_product_version_changes`. Van embebidas en el binario y `postgres.Migrate` (goose provider) aplica las que falten al arrancar `pmt-server`, antes de abrir el puerto. Si una falla, el servidor no arranca. `pmt-server agent` no migra.
- `sqlc.yaml` tiene una entrada por feature con consultas, todas sobre el esquema de las migraciones. Tras cambiar migraciones o consultas: `go tool sqlc generate`. El código de `sqlcgen/` no se edita a mano.
- **Transacciones:** las operaciones que escriben varias filas usan `pgx.BeginFunc` en su `store`, y `imports` y `workflow` comparten una `pgx.Tx`. Los cambios de status bloquean con `FOR UPDATE` y comparan el status de origen; si cambió entre leer y guardar, responden `el status cambió, vuelve a consultar`. Los lotes bloquean por orden de ID.
- **Concurrencia:** los agregados no tienen control de versión. Fuera de los casos anteriores, gana la última escritura.
- **Inmutables:** cambios, historiales, comentarios y versiones de planes están protegidos por triggers que rechazan `UPDATE`, `DELETE` y `TRUNCATE`.

## Tiempo real

1. Triggers en `projects`, `stages`, `stage_members`, `tasks`, `bugs`, `comments`, los historiales de status, `stage_changes`, `task_changes` y `product_version_changes` hacen `pg_notify('pmt_changes', {"resource": ...})` (migraciones 00021 y 00022). PostgreSQL entrega el aviso al confirmar la transacción.
2. `platform/events.Listener` escucha el canal en una conexión propia, fuera del pool, y reintenta si se cae.
3. `GET /api/events` abre un WebSocket por cliente. Envía `{"type":"invalidate","resource":"<recurso>"}` por cada cambio, y `{"type":"resync"}` al conectar o cuando el listener se recupera. Hace ping cada 30 s.

Los recursos son `project`, `stage`, `task`, `bug`, `comment` y `status_change`. PMT no avisa a los agentes: eso lo hacen ellos con herdr.

## Frontend servido

`platform/web` sirve el compilado del frontend (`frontend.dist`) en toda ruta que no sea `/api`, `/mcp`, `/healthz` ni `/readyz`, con fallback a `index.html` para las rutas del SPA. Si `frontend.dist` está vacío, no existe o no tiene `index.html`, el servidor arranca igual: registra un aviso en el log, las rutas de la interfaz responden 404 "frontend no compilado", y `/api`, `/mcp`, `/healthz` y `/readyz` funcionan normal.

## Configuración

`internal/platform/config` la carga una sola vez al arrancar, igual que en DBP, y `main.go` se la pasa a `app.Run` y a `app.RunAgent`:

- `CONFIG_FILE` da la ruta del archivo. Si falta, se lee `config/config.<ENV>.yaml` relativo al directorio de arranque, con `ENV` por omisión `dev`.
- Cada llave se sobrescribe con una variable de entorno de su mismo nombre en mayúsculas y con `_` en lugar de `.` (Viper `AutomaticEnv`).
- Cada sección se lee a su estructura en `options/` y se valida ahí.
- Un `frontend.dist` relativo se resuelve contra el directorio del archivo de configuración.

| Llave | Variable | Regla |
| --- | --- | --- |
| `server.addr` | `SERVER_ADDR` | obligatoria |
| `database.connection_string` | `DATABASE_CONNECTION_STRING` | obligatoria |
| `frontend.dist` | `FRONTEND_DIST` | opcional; si falta el compilado, el servidor arranca igual (ver Frontend servido) |

| Archivo | En git | Contenido |
| --- | --- | --- |
| `backend/config/config.dev.yaml` | no | `127.0.0.1:8080`, la base `pmt` y `../../frontend/dist` |
| `backend/config/config.prod.yaml` | sí | `0.0.0.0:8080`, `changeme # override: DATABASE_CONNECTION_STRING` y `frontend.dist: ""`, porque la imagen fija `FRONTEND_DIST` |
| `frontend/e2e/harness/config.e2e.yaml` | sí | El backend de pruebas e2e: `127.0.0.1:8090` y la base por variable ([frontend.md](frontend.md)) |

## Salud

`GET /healthz` responde 200 mientras el proceso atienda. `GET /readyz` responde 200 si la base contesta un ping en 2 s y 503 si no. Ninguna pasa por la autenticación.

## Imagen

El `Dockerfile` de la raíz compila el frontend con pnpm y `./cmd/pmt-server` con Go, y deja en alpine el binario, `config/config.prod.yaml` y el `dist` en `/app/web`, con `ENV=prod FRONTEND_DIST=/app/web`. Escucha en 8080 y su `HEALTHCHECK` consulta `/readyz`. Al correrla hay que dar `DATABASE_CONNECTION_STRING`.

## Pruebas del servidor

- Cada slice prueba su handler y sus adaptadores con dobles de la persistencia. `store` tiene pruebas propias.
- `internal/app/contracts_test.go`: las herramientas registradas coinciden con su documentación `.mcp.json`.
- `internal/app/permissions_test.go`: toda herramienta que no empiece con `list_` ni `get_` rechaza a PC sin tocar la base. Una herramienta nueva de solo lectura debe seguir esa convención de nombre.
- `internal/app/routes_test.go`: autenticación y reparto de rutas.

```bash
cd backend && go vet ./... && go test -race ./...
```

## Lo que no existe

Para no buscarlo:

- **Auditoría general** en `activity`: solo hay historiales de status, de cambios y de versión, y comentarios.
- **`kernel.Transactor`:** cada `store` abre su propia transacción.
- **Filtrado de herramientas MCP por perfil:** todos reciben las 59.
- **`idempotency_key`:** un reintento repite la operación.
- **Login del usuario humano:** el navegador entra sin token, como `saul`.
- **Multi-tenencia:** una sola organización.

## Agregar una operación

1. Decide si es un command o una query, y de qué feature.
2. Define el contrato en `api/features/<feature>/`.
3. Crea la slice en `commands/<op>/` o `queries/<op>/`, con las reglas nuevas en `domain/`.
4. Agrega la persistencia en `store/` y, si hace falta, una migración y `go tool sqlc generate`.
5. Agrega `rest.go` o `mcp.go` y regístrala en `feature.go`. Una feature nueva se registra también en `registerFeatures`.
6. Si es herramienta MCP, actualiza su `.mcp.json`. Si solo lee, su nombre empieza con `list_` o `get_`.
7. `go vet ./... && go test -race ./...`.
