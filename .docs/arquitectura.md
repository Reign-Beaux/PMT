# Arquitectura del backend

El backend es un **monolito modular en Go organizado por vertical slices**. Un solo proceso, `pmt-server`, es dueño de la base PostgreSQL y sirve todo:

```text
Navegador ── REST /api (cookie) ┐
           ── WebSocket /api/events ─┤
           ── frontend /  ──────┤
                                ├── pmt-server ── commands y queries ── PostgreSQL
Agentes ───── MCP /mcp (token) ─┘                       ▲
                                                        │ NOTIFY pmt_changes
                      listener de eventos ◄─────────────┘
```

REST y MCP llaman a los mismos handlers, así que las reglas de negocio no se duplican. Todo pide credenciales salvo las sondas de salud, el login, las invitaciones y los archivos del frontend ([Identidad](#identidad)), y cada operación comprueba antes la membresía del proyecto ([Membresía](#membresía)).

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
| Contraseñas | argon2id (`golang.org/x/crypto/argon2`) |
| Pruebas | `testing` estándar: 220 archivos y unas 690 pruebas |

## Estructura

```text
backend/
├── api/features/<feature>/     contratos: <feature>.openapi.yaml (REST) y <feature>.mcp.json (MCP)
├── cmd/pmt-server/             punto de entrada; `pmt-server agent ...` administra tokens de agente
├── internal/
│   ├── app/                    armado de features, rutas, arranque y cierre
│   ├── kernel/                 Actor, Profile, errores de negocio
│   ├── platform/
│   │   ├── rest/               servidor HTTP, autenticación por cookie o Bearer, errores
│   │   ├── mcp/                servidor MCP, autenticación por Bearer, errores de herramienta
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
- **Membresía:** la comprueba `internal/app/membership.go` para todas las features, con `projects.Membership` y los errores `NotFound` de cada `domain`. La función SQL `pmt_project_of` (migración `00027`) resuelve el proyecto de cualquier elemento leyendo las tablas de varias features; la usan esa comprobación y los avisos de cambios.
- **Excepciones:** `imports` y `workflow` no tienen agregado propio y usan el `domain` y el `store` de `projects`, `stages` y `tasks` sobre una sola transacción. Nadie las importa, así que no hay ciclos.

## Features

| Feature | Contiene |
| --- | --- |
| `projects` | Proyectos, su status, su versión de producto, sus historiales y sus miembros |
| `stages` | Etapas, su status, cambios, TL y equipo |
| `tasks` | Tareas, su status, lotes, cambios y dueño |
| `bugs` | Bugs de un proyecto en `working` |
| `reviews` | Hallazgos de CR y QA |
| `qa` | Planes QAP con versiones y tareas vinculadas |
| `devplans` | Planes de desarrollo con versiones y tareas vinculadas |
| `activity` | Comentarios |
| `imports` | `import_project` e `import_stage` desde YAML ([importar.md](importar.md)) |
| `workflow` | Operaciones sobre varios agregados: `release_project`, `start_stage`, el cierre y el retiro de etapas, y `get_project_progress` |
| `agents` | Agentes y sus tokens. Sin REST ni MCP: solo `pmt-server agent` |
| `auth` | Usuarios, sesiones del navegador, invitaciones y tokens de API de usuario. Solo REST |

## Herramientas MCP

Hay 64 herramientas, documentadas en `api/features/<feature>/<feature>.mcp.json`. La prueba `internal/app/contracts_test.go` falla si una herramienta y su documentación difieren. Cada agente recibe la lista completa; la membresía la comprueba `app` y los permisos de perfil, el handler.

| Feature | Herramientas |
| --- | --- |
| `projects` | `create_project`, `list_projects`, `update_project`, `delete_project`, `change_project_status`, `change_product_version`, `list_project_status_changes`, `list_product_version_changes`, `list_project_members`, `add_project_member`, `update_project_member`, `remove_project_member` |
| `stages` | `create_stage`, `list_stages`, `update_stage`, `delete_stage`, `change_stage_status`, `add_stage`, `revise_stage`, `retire_stage`, `list_stage_changes`, `list_stage_status_changes`, `assign_stage_lead`, `remove_stage_lead`, `add_stage_member`, `remove_stage_member` |
| `tasks` | `create_task`, `list_tasks`, `update_task`, `delete_task`, `change_task_status`, `change_tasks_status`, `add_task`, `revise_task`, `retire_task`, `list_task_changes`, `list_task_status_changes`, `assign_task` |
| `workflow` | `release_project`, `start_stage`, `get_project_progress` |
| `bugs` | `create_bug`, `update_bug`, `change_bug_status`, `list_bugs`, `list_bug_status_changes` |
| `reviews` | `create_finding`, `close_finding`, `list_findings` |
| `qa` | `create_test_plan`, `revise_test_plan`, `submit_test_plan`, `approve_test_plan`, `return_test_plan`, `list_test_plans`, `list_test_plan_versions` |
| `devplans` | `create_dev_plan`, `revise_dev_plan`, `list_dev_plans`, `list_dev_plan_versions` |
| `activity` | `add_comment`, `list_comments` |
| `imports` | `import_project`, `import_stage` |

**REST** expone lo mismo bajo `/api/`, salvo `imports`, que es solo MCP. Los miembros están en `/api/projects/{id}/members` y `/api/projects/{id}/members/{username}`. La fuente de verdad son los `*.openapi.yaml`. Además hay `GET` de un proyecto, una etapa y una tarea por su ID, y las rutas de `auth` ([auth.openapi.yaml](../backend/api/features/auth/auth.openapi.yaml)):

| Ruta | Quién |
| --- | --- |
| `POST /api/auth/login`, `POST /api/auth/logout` | Pública |
| `GET /api/invitations/{token}`, `POST /api/invitations/{token}/accept` | Pública: el token del enlace es la credencial |
| `GET /api/auth/me`, `POST /api/auth/me/password` | El usuario; cambiar la contraseña solo con la cookie |
| `GET`, `POST /api/tokens`, `DELETE /api/tokens/{id}` | El usuario, sobre sus propios tokens |
| `GET`, `POST /api/invitations`, `GET /api/users`, `POST /api/users/{id}/disable` y `/enable` | El dueño de la instancia |

## Identidad

Hay dos clases de actor ([dominio.md](dominio.md#conceptos)):

- **Usuario:** una persona con correo, nombre, contraseña y `username`. Actúa como `Actor{Name: username, Profile: <perfil de la sesión>}`: con la cookie, siempre `User`; con su token, el perfil que declare la sesión, o `User` si no declara ninguno. Puede usar cualquier perfil; las reglas de permisos son las mismas que para un agente de ese perfil.
- **Agente:** fila de `agents` con nombre y perfil fijo, administrada con `pmt-server agent`. Sigue igual que antes: si la sesión declara un perfil, debe ser el suyo. Actúa con la membresía del dueño de la instancia.

El actor lleva también `UserID`, el usuario cuya membresía usa: lo pone `auth` para la cookie y el token de usuario, y `app.credentials` pone el del dueño de la instancia (`auth.Feature.InstanceOwnerID`) para un token de agente. Se eligió agregarlo al `Actor` en vez de resolverlo por `username` en cada operación porque los agentes no tienen usuario propio: así la membresía se consulta igual para los dos, sin otra búsqueda por petición.

Un `username` no puede coincidir con el nombre de un agente, revocado o no, ni al revés: lo validan `auth` (al aceptar una invitación y al crear el dueño) y `pmt-server agent create`, cada uno con el `Checker` de la otra feature.

**Credenciales.** `platform/rest.Authenticate` y `platform/mcp.Authenticate` resuelven el actor y lo pasan a cada handler; nunca es un argumento de la herramienta ni del cuerpo.

| Credencial | Dónde | Resuelve |
| --- | --- | --- |
| Cookie `pmt_session` | `/api/`, incluido el WebSocket `/api/events` | El usuario de una sesión vigente (30 días) |
| `Authorization: Bearer pmt_<64 hex>` | `/api/` y `/mcp` | El usuario dueño del token, con el perfil declarado |
| `Authorization: Bearer <64 hex>` | `/api/` y `/mcp` | El agente del token, con su perfil fijo |

**Perfil por sesión.** Solo con token:

| Transporte | Cómo se declara |
| --- | --- |
| `/mcp` | Parámetro de la URL `profile` (`/mcp?profile=tl`), para que Claude, Codex y Qwen lo configuren sin cabeceras extra |
| `/api/` y `/api/events` | Cabecera `X-PMT-Profile` |

- Valores sin distinguir mayúsculas: `pp`, `pc`, `tl`, `dp`, `devfrontend`, `devbackend`, `cr`, `qap`, `qa`; `user` equivale a no declarar. Los interpreta `kernel.ParseProfile`.
- `platform/rest` y `platform/mcp` leen el perfil y lo pasan a `kernel.Authenticator.AuthenticateToken`; `app.credentials` lo aplica con `kernel.AsUser` (token de usuario) o `kernel.AsAgent` (token de agente).
- Un valor desconocido, o un perfil distinto del de un token de agente, responde **400** con el motivo (en `/mcp`, como texto plano: el middleware del SDK responde 400 a los errores `auth.ErrOAuth`). Sin credenciales sigue siendo 401 antes de mirar el perfil.
- Con la cookie, `X-PMT-Profile` se ignora: el frontend es de consulta y siempre actúa como `User`.
- `GET /api/auth/me` devuelve el usuario y `profile`, el perfil efectivo de la petición, con cualquier perfil. Las demás operaciones de `auth` (tokens, invitaciones, usuarios, contraseña) siguen siendo del perfil `User`: un usuario con perfil declarado recibe 403.
- **TL y equipo.** `assign_stage_lead`, `add_stage_member` y `assign_task` aceptan el nombre de un agente activo (con la validación de perfil de siempre) o el `username` de un usuario activo, sin revisar perfil: lo comprueba cada operación cuando actúa. Lo consultan con `auth.Checker.Active`. Por eso `stages.lead`, `stage_members.agent` y `tasks.assignee` ya no tienen llave foránea a `agents` (migración `00026`).

- Con cabecera `Authorization` manda el token; sin ella, la cookie. `/mcp` solo acepta tokens.
- Sin credenciales, con `Bearer` vacío, o con una cookie o un token inválidos, vencidos, revocados o de un usuario desactivado: **401**. No hay acceso anónimo: el antiguo «sin token actúa `saul`» ya no existe.
- Abiertas sin credenciales: `/healthz`, `/readyz`, las rutas públicas de `auth` (`auth.IsPublic`) y los archivos del frontend.
- Si una ruta protegida llegara sin actor, `ActorFrom` devuelve el actor vacío, sin perfil. `permissions_test.go` exige que toda herramienta de escritura lo rechace sin tocar la base.

**Secretos.** Sesiones, tokens de API e invitaciones son 32 bytes aleatorios; PMT solo guarda su SHA-256. Las contraseñas se guardan con argon2id. El token de API y el enlace de invitación solo se muestran al crearlos.

**Cookie.** `HttpOnly`, `SameSite=Lax`, `Path=/` y `Secure` según `auth.cookie_secure`. Cambiar la contraseña cierra las demás sesiones del usuario; desactivarlo borra todas y revoca sus tokens.

**Tokens de agente.** Viven en `~/.config/pmt/tokens/<nombre>`, con permiso 600. Los clientes MCP mandan `Authorization: Bearer ${PMT_TOKEN}`. Un token de usuario se crea en la pantalla Cuenta del frontend y se usa igual, con `?profile=` en la URL del MCP ([operacion.md](operacion.md#usuarios-y-acceso)).

## Membresía

Cada proyecto tiene miembros en `project_members (project_id, user_id, role)`, con un solo `owner` (índice único parcial) y `editor` o `viewer` ([dominio.md](dominio.md#miembros-y-roles)).

La comprobación vive en un solo lugar, `internal/app/membership.go`, y corre antes de cada herramienta MCP (`platform/mcp.Guard`) y de cada ruta REST (`platform/rest.Guard`, dentro de `Authenticate`):

1. Saca los IDs que recibe la operación por su nombre: `project_id`, `stage_id`, `task_id`, `task_ids`, `bug_id`, `finding_id`, `plan_id` (de pruebas o de desarrollo según la herramienta o la ruta) e `item_id` con su `item_kind`. En REST los toma de la ruta (el `{id}` según su primer segmento: `/api/stages/{id}` es una etapa), de la query y del cuerpo JSON, que deja intacto para el handler.
2. Para cada uno, `projects.Membership.RoleOf` consulta con `pmt_project_of` su proyecto y el rol del actor.
3. Si el elemento no existe o el actor no es miembro, responde el `not_found` de ese tipo, el mismo que daría el handler (`etapa no encontrada`). Si es viewer y llegó con token, `forbidden` lea o escriba (`projects.ErrViewerOnlyWeb`). Si es viewer con la cookie y la operación escribe (herramienta que no empieza con `list_` ni `get_`, o ruta que no es `GET`), `forbidden`. Si no, sigue al handler, que aplica sus reglas de perfil.

**Canal de la credencial.** `kernel.Actor.Channel` dice si el actor llegó con la cookie (`ChannelSession`) o con un token (`ChannelToken`). Lo fijan `rest.Authenticate` y `mcp.Authenticate`, que son los únicos que ven la credencial; los autenticadores de `auth` y `agents` no lo conocen. El valor vacío cuenta como token, así que lo que solo se permite desde el sitio web exige `ChannelSession` explícito. `projects.Role.Reaches(actor)` es la regla: owner y editor llegan con cualquiera; el viewer, solo con la cookie. La usan la guardia, `/api/events` (`guard.visible`) y `list_projects` (`domain.ReachableRoles`).

**Quién puede trabajar.** `assign_stage_lead`, `add_stage_member` y `assign_task` preguntan a `projects.Workers` (slice `projects/queries/checkworker`) si el nombre es un usuario activo; si lo es y no es owner ni editor del proyecto de la etapa, responde `conflict`. Si no es usuario, siguen validándolo como agente. El proyecto de la tarea llega en `kernel.Team.ProjectID`.

`list_projects` filtra en su consulta por miembro y por los roles que alcanzan con la credencial; `create_project` e `import_project` dejan al actor como owner en la misma sentencia o transacción. Al arrancar, después de crear al dueño, `projects.AdoptOrphans` lo deja como owner de los proyectos sin owner: los que existían cuando la migración `00027` no encontró usuarios.

Un ID que no es UUID no se revisa: el handler lo rechaza como inválido. Por eso la comprobación de perfil del handler ya no es lo primero: un no miembro recibe `not_found` aunque su perfil tampoco pudiera.

## Errores

- Los errores de negocio se crean en `domain` con `kernel.Invalid`, `NotFound`, `Conflict` o `Forbidden`.
- REST responde 400, 404, 409 o 403 con `{"error": "mensaje"}`. Cualquier otro error se registra en el log y responde 500 con un mensaje genérico.
- `kernel.Unauthorized` es la credencial ausente o inválida: REST responde 401.
- MCP antepone el tipo para que el agente lo distinga sin interpretar el texto: `invalid: `, `not_found: `, `conflict: `, `forbidden: ` o `unauthorized: `. Cualquier otro sale como `error interno`.
- `store` traduce las violaciones de llave foránea y de unicidad a errores de negocio.

## Datos

- 27 migraciones en `internal/platform/postgres/migrations/`, de `00001_create_projects` a `00027_create_project_members`. Las de usuarios son `00023_create_users_and_sessions`, `00024_create_api_tokens` y `00025_create_invitations`; `00026` quita las llaves foráneas de TL, equipo y dueño de tarea hacia `agents`. `00027` crea `project_members`, deja al dueño de la instancia como owner de los proyectos existentes, crea `pmt_project_of` y agrega el proyecto a los avisos de cambios. Van embebidas en el binario y `postgres.Migrate` (goose provider) aplica las que falten al arrancar `pmt-server`, antes de abrir el puerto. Si una falla, el servidor no arranca. `pmt-server agent` no migra.
- `sqlc.yaml` tiene una entrada por feature con consultas, todas sobre el esquema de las migraciones. Tras cambiar migraciones o consultas: `go tool sqlc generate`. El código de `sqlcgen/` no se edita a mano.
- **Transacciones:** las operaciones que escriben varias filas usan `pgx.BeginFunc` en su `store`, y `imports` y `workflow` comparten una `pgx.Tx`. Los cambios de status bloquean con `FOR UPDATE` y comparan el status de origen; si cambió entre leer y guardar, responden `el status cambió, vuelve a consultar`. Los lotes bloquean por orden de ID.
- **Concurrencia:** los agregados no tienen control de versión. Fuera de los casos anteriores, gana la última escritura.
- **Inmutables:** cambios, historiales, comentarios y versiones de planes están protegidos por triggers que rechazan `UPDATE`, `DELETE` y `TRUNCATE`.

## Tiempo real

1. Triggers en `projects`, `project_members`, `stages`, `stage_members`, `tasks`, `bugs`, `comments`, los historiales de status, `stage_changes`, `task_changes` y `product_version_changes` hacen `pg_notify('pmt_changes', {"resource", "project_id"})` (migraciones 00021, 00022 y 00027). Un cambio de miembros lleva además `user_id`, el usuario afectado. PostgreSQL entrega el aviso al confirmar la transacción.
2. `platform/events.Listener` escucha el canal en una conexión propia, fuera del pool, y reintenta si se cae.
3. `GET /api/events` abre un WebSocket por cliente. Envía `{"type":"invalidate","resource":"<recurso>"}` por cada cambio de un proyecto que ve quien abrió la conexión, y `{"type":"resync"}` al conectar o cuando el listener se recupera. Hace ping cada 30 s.
4. **Filtro por conexión:** entrega el aviso si el actor es miembro del proyecto (consulta `app.guard.visible` y lo recuerda hasta el siguiente cambio de proyecto, que incluye los de miembros) o si es el `user_id` afectado, para que quien deja de ser miembro se entere. Un aviso sin proyecto no se entrega. El navegador sigue recibiendo solo `type` y `resource`.

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
| `app.public_url` | `APP_PUBLIC_URL` | obligatoria; base de los enlaces de invitación |
| `auth.cookie_secure` | `AUTH_COOKIE_SECURE` | `true` marca la cookie como `Secure` (solo HTTPS) |
| `owner.email` | `OWNER_EMAIL` | dueño de la instancia; se crea al arrancar si no hay usuarios |
| `owner.password` | `OWNER_PASSWORD` | al menos 8 caracteres |
| `owner.username` | `OWNER_USERNAME` | su `username`; no puede ser nombre de un agente |

Si faltan `owner.email` u `owner.password` no se crea el dueño (avisa en el log si solo falta uno). Si hay usuarios, `owner.*` no se lee. Un dueño mal configurado con la base vacía detiene el arranque.

Como en DBP, una llave solo se puede sobrescribir con su variable si existe en el archivo: por eso los archivos llevan todas, aunque sea con `changeme # override: ...`.

| Archivo | En git | Contenido |
| --- | --- | --- |
| `backend/config/config.dev.yaml` | no | `127.0.0.1:8080`, la base `pmt`, `../../frontend/dist`, `public_url` `http://127.0.0.1:8080`, cookie sin `Secure` y el dueño `saul` |
| `backend/config/config.prod.yaml` | sí | `0.0.0.0:8080`, `https://pmt.saulantonio.com`, cookie `Secure`, `frontend.dist: ""` porque la imagen fija `FRONTEND_DIST`, y `changeme # override: ...` en la base y en `owner.*` |
| `frontend/e2e/harness/config.e2e.yaml` | sí | El backend de pruebas e2e: `127.0.0.1:8090`, `public_url` `http://localhost:5173`, y la base y el dueño de pruebas por variable ([frontend.md](frontend.md)) |

## Salud

`GET /healthz` responde 200 mientras el proceso atienda. `GET /readyz` responde 200 si la base contesta un ping en 2 s y 503 si no. Ninguna pasa por la autenticación.

## Imagen

El `Dockerfile` de la raíz compila el frontend con pnpm y `./cmd/pmt-server` con Go, y deja en alpine el binario, `config/config.prod.yaml` y el `dist` en `/app/web`, con `ENV=prod FRONTEND_DIST=/app/web`. Escucha en 8080 y su `HEALTHCHECK` consulta `/readyz`. Al correrla hay que dar `DATABASE_CONNECTION_STRING` y, la primera vez, `OWNER_EMAIL`, `OWNER_PASSWORD` y `OWNER_USERNAME`.

## Pruebas del servidor

- Cada slice prueba su handler y sus adaptadores con dobles de la persistencia. `store` tiene pruebas propias.
- `internal/app/contracts_test.go`: las herramientas registradas coinciden con su documentación `.mcp.json`.
- `internal/app/permissions_test.go`: toda herramienta que no empiece con `list_` ni `get_` rechaza a PC y al actor vacío sin tocar la base. Una herramienta nueva de solo lectura debe seguir esa convención de nombre.
- `internal/app/membership_test.go`: recorre toda herramienta registrada y toda ruta que registran los `rest.go` de las features (leídas del código), con los IDs que aceptan según su esquema o su `*.openapi.yaml`, y exige `not_found` a quien no es miembro y `forbidden` a un viewer con token, sin llegar al handler; solo se salvan `create_project`, `import_project`, `list_projects` y `GET`/`POST /api/projects`. También exige que contrato y código tengan las mismas rutas, que el viewer con cookie solo lea, y prueba owner, editor, viewer por token y por cookie, no miembro, token de agente y la visibilidad de los eventos.
- `internal/app/routes_test.go`: 401 sin credenciales, cookie y token de usuario, token de agente, usuario desactivado, rutas públicas y reparto de rutas.
- `internal/features/auth/feature_test.go`: el recorrido de login, tokens, invitación, desactivación y cambio de contraseña por HTTP, sobre el repositorio en memoria de `auth/authtest`.
- Las pruebas de adaptadores REST y MCP de otras features actúan como usuario con `asUser` (`usuario_test.go`) o `platformmcp.ActAs`.

```bash
cd backend && go vet ./... && go test -race ./...
```

## Lo que no existe

Para no buscarlo:

- **Auditoría general** en `activity`: solo hay historiales de status, de cambios y de versión, y comentarios.
- **`kernel.Transactor`:** cada `store` abre su propia transacción.
- **Filtrado de herramientas MCP por perfil:** todos reciben las 64.
- **`idempotency_key`:** un reintento repite la operación.
- **Envío de correos:** el dueño copia el enlace de invitación y lo manda él.
- **Bloqueos por persona y pantallas de miembros:** fases 1d y 1e del [plan](plan_produccion.md).
- **Transferir el owner** de un proyecto.
- **Multi-tenencia:** una sola organización.

## Agregar una operación

1. Decide si es un command o una query, y de qué feature.
2. Define el contrato en `api/features/<feature>/`.
3. Crea la slice en `commands/<op>/` o `queries/<op>/`, con las reglas nuevas en `domain/`.
4. Agrega la persistencia en `store/` y, si hace falta, una migración y `go tool sqlc generate`.
5. Agrega `rest.go` o `mcp.go` y regístrala en `feature.go`. Una feature nueva se registra también en `registerFeatures`.
6. Si es herramienta MCP, actualiza su `.mcp.json`. Si solo lee, su nombre empieza con `list_` o `get_`.
7. `go vet ./... && go test -race ./...`.
