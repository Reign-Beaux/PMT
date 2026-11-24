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

REST y MCP llaman a los mismos handlers, así que las reglas de negocio no se duplican. Todo pide credenciales salvo las sondas de salud, el login, las invitaciones, el inicio de sesión del CLI, las descargas del CLI y los archivos del frontend ([Identidad](#identidad)), y cada operación comprueba antes la membresía del proyecto ([Membresía](#membresía)) y, si escribe, los bloqueos por persona ([Bloqueos](#bloqueos)).

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
| Pruebas | `testing` estándar: 238 archivos y unas 780 pruebas |

## Estructura

```text
backend/
├── api/features/<feature>/     contratos: <feature>.openapi.yaml (REST) y <feature>.mcp.json (MCP)
├── cmd/pmt-server/             punto de entrada; `pmt-server agent ...` administra tokens de agente
├── cmd/pmt/                    el CLI `pmt` ([CLI pmt](#cli-pmt))
├── scripts/build-cli.sh        compila `pmt` para los cuatro sistemas y escribe SHA256SUMS
├── internal/
│   ├── app/                    armado de features, rutas, arranque y cierre
│   ├── cli/                    subcomandos del CLI, credenciales, cliente HTTP, .pmt/ y herdr
│   ├── kernel/                 Actor, Profile, errores de negocio
│   ├── platform/
│   │   ├── rest/               servidor HTTP, autenticación por cookie o Bearer, errores
│   │   ├── mcp/                servidor MCP, autenticación por Bearer, errores de herramienta
│   │   ├── events/             listener de NOTIFY y WebSocket /api/events
│   │   ├── web/                frontend compilado con fallback SPA
│   │   ├── downloads/          /install.sh y /descargas del CLI, desde cli.dist
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
| `profiles` | Texto de los perfiles de los agentes (`catalog/`, incrustado), servido con la herramienta `get_profile` y en `GET /api/profiles` y `GET /api/profiles/{profile}`; da también las instrucciones cortas del MCP de cada sesión. Sin tablas |

## Herramientas MCP

Hay 67 herramientas, documentadas en `api/features/<feature>/<feature>.mcp.json`. La prueba `internal/app/contracts_test.go` falla si una herramienta y su documentación difieren. Cada agente recibe la lista completa; la membresía la comprueba `app` y los permisos de perfil, el handler.

| Feature | Herramientas |
| --- | --- |
| `projects` | `create_project`, `list_projects`, `update_project`, `delete_project`, `change_project_status`, `change_product_version`, `list_project_status_changes`, `list_product_version_changes`, `list_project_members`, `add_project_member`, `update_project_member`, `remove_project_member` |
| `stages` | `create_stage`, `list_stages`, `update_stage`, `delete_stage`, `change_stage_status`, `add_stage`, `revise_stage`, `retire_stage`, `list_stage_changes`, `list_stage_status_changes`, `assign_stage_lead`, `remove_stage_lead`, `add_stage_member`, `remove_stage_member`, `release_stage` |
| `tasks` | `create_task`, `list_tasks`, `update_task`, `delete_task`, `change_task_status`, `change_tasks_status`, `add_task`, `revise_task`, `retire_task`, `list_task_changes`, `list_task_status_changes`, `assign_task` |
| `workflow` | `release_project`, `start_stage`, `get_project_progress` |
| `bugs` | `create_bug`, `update_bug`, `change_bug_status`, `list_bugs`, `list_bug_status_changes`, `release_bug` |
| `reviews` | `create_finding`, `close_finding`, `list_findings` |
| `qa` | `create_test_plan`, `revise_test_plan`, `submit_test_plan`, `approve_test_plan`, `return_test_plan`, `list_test_plans`, `list_test_plan_versions` |
| `devplans` | `create_dev_plan`, `revise_dev_plan`, `list_dev_plans`, `list_dev_plan_versions` |
| `activity` | `add_comment`, `list_comments` |
| `imports` | `import_project`, `import_stage` |
| `profiles` | `get_profile` |

**REST** expone lo mismo bajo `/api/`, salvo `imports`, que es solo MCP. Los miembros están en `/api/projects/{id}/members` y `/api/projects/{id}/members/{username}`; liberar, en `POST /api/stages/{id}/release` y `POST /api/bugs/{id}/release`. La fuente de verdad son los `*.openapi.yaml`. Además hay `GET` de un proyecto, una etapa y una tarea por su ID, `GET /api/profiles` y `GET /api/profiles/{profile}` ([Perfiles servidos](#perfiles-servidos)), y las rutas de `auth` ([auth.openapi.yaml](../backend/api/features/auth/auth.openapi.yaml)):

| Ruta | Quién |
| --- | --- |
| `POST /api/auth/login`, `POST /api/auth/logout` | Pública |
| `GET /api/invitations/{token}`, `POST /api/invitations/{token}/accept` | Pública: el token del enlace es la credencial |
| `GET /api/auth/me`, `POST /api/auth/me/password` | El usuario; cambiar la contraseña solo con la cookie |
| `GET`, `POST /api/tokens`, `DELETE /api/tokens/{id}` | El usuario, sobre sus propios tokens |
| `GET`, `POST /api/invitations`, `GET /api/users`, `POST /api/users/{id}/disable` y `/enable` | El dueño de la instancia |

## Perfiles servidos

`internal/features/profiles/catalog` incrusta con `embed` los perfiles, `comun.md` y `pc-comun.md` ([perfiles.md](perfiles.md#dónde-viven-los-perfiles)) y arma al arrancar el texto servido de cada perfil: el rol más lo común, tomando la parte servida de cada tramo `[[local||servido]]`.

- **Instrucciones por sesión.** El SDK fija las instrucciones por servidor. En vez de un servidor por perfil, `platform/mcp.Instructions` agrega un middleware de recepción al único servidor: al responder `initialize` (y `server/discover`) toma el actor que dejó `mcp.Authenticate` y le pide a `profiles.Instructions` las suyas. Las herramientas se registran una sola vez y la membresía no cambia. Lo instala `routes` en `internal/app/app.go`. Las instrucciones son cortas, muy por debajo de los ~4 000 caracteres que muestra Claude Code (los perfiles servidos miden hasta ~9 000): nombran el perfil y piden, como primer paso, llamar a `get_profile` y leer `.pmt/entorno.md` ([texto](perfiles.md#cómo-se-sirven)). Sin perfil (`User`) o con token de agente (`kernel.Actor.Agent`, que fija `app.credentials`) no hay: el kit ya da el perfil completo por archivo.
- **`get_profile`.** Sin argumentos, devuelve como texto (Markdown, sin `structuredContent`) el perfil servido de la sesión, el mismo de `GET /api/profiles/{profile}`. Cualquier perfil, también con token de agente. Sin perfil declarado responde `invalid` con cómo declararlo (`?profile=`). No recibe IDs, así que la guardia de membresía y de bloqueos la deja pasar; empieza con `get_`, así que cuenta como lectura.
- **REST.** `GET /api/profiles` lista `name` (como se declara en la sesión), `profile` y `description`; `GET /api/profiles/{profile}` devuelve el texto en `text/markdown`, o 404 si no existe (`user` tampoco). Cualquier credencial y cualquier perfil. Contrato en [profiles.openapi.yaml](../backend/api/features/profiles/profiles.openapi.yaml). No reciben IDs de proyecto, así que la guardia de membresía las deja pasar.

## Identidad

Hay dos clases de actor ([dominio.md](dominio.md#conceptos)):

- **Usuario:** una persona con correo, nombre, contraseña y `username`. Actúa como `Actor{Name: username, Profile: <perfil de la sesión>}`: con la cookie, siempre `User`; con su token, el perfil que declare la sesión, o `User` si no declara ninguno. Puede usar cualquier perfil; las reglas de permisos son las mismas que para un agente de ese perfil.
- **Agente:** fila de `agents` con nombre y perfil fijo, administrada con `pmt-server agent`. Sigue igual que antes: si la sesión declara un perfil, debe ser el suyo. Actúa con la membresía del dueño de la instancia.

El actor lleva también `UserID`, el usuario cuya membresía usa, y `UserName`, su `username` (el nombre con que toma una etapa o un bug): lo pone `auth` para la cookie y el token de usuario, y `app.credentials` pone el del dueño de la instancia (`auth.Feature.InstanceOwnerID`) para un token de agente. Se eligió agregarlo al `Actor` en vez de resolverlo por `username` en cada operación porque los agentes no tienen usuario propio: así la membresía se consulta igual para los dos, sin otra búsqueda por petición.

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
- Abiertas sin credenciales: `/healthz`, `/readyz`, las rutas públicas de `auth` (`auth.IsPublic`: login, logout, consultar y aceptar una invitación, y `POST /api/cli/login` y `/api/cli/login/poll`), `/install.sh`, `/descargas/` y los archivos del frontend.
- Si una ruta protegida llegara sin actor, `ActorFrom` devuelve el actor vacío, sin perfil. `permissions_test.go` exige que toda herramienta de escritura lo rechace sin tocar la base.

**Secretos.** Sesiones, tokens de API e invitaciones son 32 bytes aleatorios; PMT solo guarda su SHA-256. Las contraseñas se guardan con argon2id. El token de API y el enlace de invitación solo se muestran al crearlos.

**Cookie.** `HttpOnly`, `SameSite=Lax`, `Path=/` y `Secure` según `auth.cookie_secure`. Cambiar la contraseña cierra las demás sesiones del usuario; desactivarlo borra todas y revoca sus tokens.

**Tokens de agente.** Viven en `~/.config/pmt/tokens/<nombre>`, con permiso 600. Los clientes MCP mandan `Authorization: Bearer ${PMT_TOKEN}`. Un token de usuario se crea en la pantalla Conectar del frontend y se usa igual, con `?profile=` en la URL del MCP ([operacion.md](operacion.md#usuarios-y-acceso)).

## CLI pmt

`pmt` es un binario aparte, en `cmd/pmt` sobre `internal/cli`, del mismo módulo. Usa `flag` de la biblioteca estándar: con pocos subcomandos no hace falta una librería, y `cobra` solo está en el módulo como dependencia indirecta de `sqlc`.

| Comando | Qué hace |
| --- | --- |
| `pmt login [--servidor URL] [--sin-navegador]` | Inicia sesión por el navegador y guarda el token. El servidor por omisión es `https://pmt.saulantonio.com`; el que se use queda guardado para el siguiente `login`. Si había otro token, lo revoca |
| `pmt logout` | Revoca el token en el servidor (`DELETE /api/tokens/{id}` con el mismo token) y borra el archivo. Sin conexión no borra nada; con el token ya inválido, solo borra |
| `pmt whoami` | Usuario y servidor, con `GET /api/auth/me` |
| `pmt version` | Versión incrustada al compilar (`-X pmt/internal/cli.Version=…`), sistema y arquitectura |
| `pmt pp [--configurar]` | Levanta a PP en herdr para el proyecto de la carpeta actual ([pmt pp](#pmt-pp)) |
| `pmt equipo [--configurar] [agente…]` | Levanta en herdr los agentes que falten del equipo, o solo los nombrados ([pmt equipo](#pmt-equipo)) |
| `pmt avisar <agente> "mensaje"` | Envía un aviso al agente y confirma que llegó ([comandos del equipo](#comandos-del-equipo)) |
| `pmt levantar <agente> "encargo"` | Reinicia al agente y le da el encargo; con el TL arranca el vigía |
| `pmt reiniciar <agente>` | Deja al agente con una sesión vacía |
| `pmt relanzar-dp <medium\|high>` | Guarda el effort de DP en `config.yaml` y lo relanza |
| `pmt vigia [--pulso 15s] [--quieto 150s]` | Avisa al TL si todos los agentes llevan un rato quietos |

**Credenciales:** `~/.config/pmt/credenciales.yaml` con `servidor`, `username`, `token` y `token_id`, permiso 600, carpeta 700. Se escribe a un temporal y se renombra. No usa `~/.config/pmt/tokens/`, que es del kit. El token nunca se imprime.

**Login por navegador** (como `gh auth login`), en la feature `auth`:

1. `POST /api/cli/login` (público) con `machine` (el nombre de la máquina, de 1 a 60 caracteres) devuelve `code` (`ABCD-1234`: cuatro letras sin I ni O y cuatro dígitos), `poll_secret` (32 bytes aleatorios que solo conoce el CLI), `url` (`app.public_url` + `/cli?codigo=<código>`), `expires_at` (10 minutos) e `interval` (2 s).
2. El CLI abre el navegador con `xdg-open` u `open` (en Linux, solo con `DISPLAY` o `WAYLAND_DISPLAY`) y siempre imprime la URL, para abrirla en otro equipo. Muestra el código para compararlo.
3. La pantalla `/cli` del frontend ([frontend.md](frontend.md)) muestra el código y la máquina con `GET /api/cli/login/{code}` y llama a `POST /api/cli/login/{code}/authorize` o `/cancel`. Son operaciones de cuenta: sin perfil declarado. Solo un código pendiente y vigente se autoriza o se cancela; si no, `conflict`.
4. El CLI sondea `POST /api/cli/login/poll` (público) con su secreto cada 2 s: `pending`, `authorized` (con `token`, `token_id` y `username`), `cancelled` o `expired`. Responde con `Cache-Control: no-store`. Un fallo de red se reintenta hasta que caduca el código.

- **Hashes y uso único:** se guarda solo el SHA-256 del secreto. El token de API (`pmt <máquina>`, como los de la cuenta) se crea en el primer sondeo después de autorizar, en la misma transacción que deja el código `consumed`; así nunca queda en la base en ninguna forma recuperable. Después el código responde `expired`. Si el usuario se desactivó entre autorizar y sondear, responde `cancelled`. Autorizado pero sin sondear antes de los 10 minutos, caduca sin token.
- **Contra abusos:** pedir un código es público, así que hay un tope de **20 códigos pendientes y vigentes en toda la instancia**; el 21.º responde `conflict` («hay demasiados inicios de sesión pendientes; intenta en unos minutos»). Cada petición borra los vencidos. El secreto de sondeo no se puede adivinar, y autorizar exige sesión.
- La tabla es `cli_logins` (migración `00029`).

**Descargas.** `platform/downloads` sirve, sin credenciales, `GET /install.sh` (`internal/platform/downloads/install.sh`, incrustado, con `app.public_url` ya puesto) y `GET /descargas/{pmt-linux-amd64|pmt-linux-arm64|pmt-darwin-amd64|pmt-darwin-arm64|SHA256SUMS}` desde `cli.dist`. Cualquier otro nombre responde 404. La carpeta se revisa en cada petición, así que basta compilar los binarios sin reiniciar. Si `cli.dist` está vacío, no existe o no tiene `SHA256SUMS`, el servidor arranca igual con un aviso en el log: `/install.sh` responde un script que dice «este servidor no tiene los binarios del CLI» y sale con error, y `/descargas/` responde 404.

`install.sh` es POSIX `sh`: detecta sistema (`uname -s`) y arquitectura (`uname -m`), descarga con `curl` o `wget`, verifica la suma con `sha256sum` o `shasum -a 256` contra `SHA256SUMS`, instala en `~/.local/bin/pmt` sin `sudo` y avisa si esa carpeta no está en el `PATH`. Si la suma no coincide no instala nada.

### pmt pp

Hace por PP lo que hacían `iniciar.sh` y `lanzar.sh` del kit, sin `.agentes.env`, sin tokens de agente y sin rutas de esta máquina. Archivos: `pp.go` (el flujo, las preguntas y lo que comparte con `pmt equipo`: workspace, prefijo, pane del agente y preparar el lanzamiento), `project.go` (`.pmt/config.yaml`, raíz y `.gitignore`), `entorno.go` (`.pmt/entorno.md`), `launch.go` (argumentos de Claude y Codex y `mcp-<perfil>.json`), `launcher.go` (el lanzador) y `herdr.go` (el comando `herdr`, detrás de una interfaz para las pruebas).

1. **Requisitos:** `HERDR_ENV=1` (si no, dice cómo abrir herdr), credenciales de `pmt login` y la herramienta elegida en el `PATH`.
2. **Configuración:** raíz = la carpeta actual o la más cercana hacia arriba con `.pmt/`. Sin `.pmt/config.yaml`, o con `--configurar`, pregunta proyecto (`GET /api/projects`, owner o editor, más «nuevo»), herramienta, modelo y effort ([formato](perfiles.md#formato-de-pmtconfigyaml)). Agrega `.pmt/` al `.gitignore` si hay repo git, sin duplicar. Si el proyecto es «nuevo», busca uno del usuario creado después y pregunta si es ese.
3. **Entorno:** regenera `.pmt/entorno.md` ([formato](perfiles.md#formato-de-pmtentornomd)) y crea `.pmt/para-usuario.md`.
4. **herdr:** busca el workspace de la config por etiqueta o ID y lo crea si falta (su primera pestaña pasa a llamarse `pp`). El agente se llama `pp`; si otro workspace ya tiene un agente con ese nombre, pone en la config el prefijo `<workspace>-`, como `HERDR_PREFIJO` del kit. Si en el workspace ya corre PP (un agente con su nombre, o la pestaña `pp` con algo en primer plano además del shell, tras esperar hasta 10 s a que un shell nuevo quede libre), no lo relanza y enfoca su pestaña.
5. **Perfil:** `GET /api/profiles/pp` a `.pmt/perfiles/pp.md`. Si falla y hay copia, usa la copia y avisa; con 401 pide `pmt login`.
6. **Lanzar:** en la pestaña `pp` (la crea si falta) corre con `herdr pane run` `pmt _lanzar pp <raíz>`. `_lanzar` es un subcomando interno que corre dentro del pane, como `lanzar.sh`: lee el token de las credenciales y los de MCP extra de `~/.config/pmt/`, los pone en el entorno (`PMT_TOKEN`, `MCP_TOKEN_<NOMBRE>`) y se reemplaza (`syscall.Exec`) por Claude o Codex. Luego `pmt pp` espera hasta un minuto a que herdr detecte al agente, lo nombra con `herdr agent rename` y enfoca la pestaña.

**Por qué `pane run` y no `herdr agent start`:** `agent start` lanza el ejecutable canónico de la herramienta con sus argumentos y no tiene forma de pasar variables de entorno; el token tendría que ir en la línea de comandos. `tab create --env` sí las pasa, pero el token quedaría en los argumentos de `herdr` y en el estado de herdr, y no sirve con una pestaña que ya existe. Con `pane run` del lanzador, lo que se escribe en el pane (y queda en su historial) no lleva secretos, y como `_lanzar` hace `exec`, el proceso que herdr ve es `claude` o `codex`, que es como lo detecta.

**El token nunca se escribe** en `.pmt/`, en argumentos (`ps`) ni en el pane: solo vive en el entorno del proceso del agente (y de lo que este lance).

| | Claude | Codex |
| --- | --- | --- |
| Perfil | `--append-system-prompt-file .pmt/perfiles/pp.md` | `-c developer_instructions="<texto>"` |
| MCP de PMT | `.pmt/mcp-pp.json` con `<servidor>/mcp?profile=pp` y `Authorization: Bearer ${PMT_TOKEN:-}`, más `--strict-mcp-config` | `-c mcp_servers.pmt.url=…?profile=pp` y `-c mcp_servers.pmt.bearer_token_env_var="PMT_TOKEN"` |
| MCP extra | en el mismo JSON, con `${MCP_TOKEN_<NOMBRE>:-}` | `-c mcp_servers.<nombre>.url` y `bearer_token_env_var` |
| Opciones | `--dangerously-skip-permissions --autocompact 400k --model <m> --effort <e>` | `-m <m>`, `model_reasoning_effort`, `allow_login_shell=false`, `notice.hide_rate_limit_model_nudge=true`, `projects={"<ruta>"={trust_level="trusted"}}`, `CODEX_HOME` de `codex_home` |

Respecto a `lanzar.sh` se quitó `--settings '{"enabledPlugins":{"engram@engram":false}}'` de Claude (el plugin `engram` solo existe en esta máquina) y la ruta fija `~/.codex-flow` de Codex (pasa a `codex_home`). La confianza de Codex va como tabla en línea porque Codex 0.171 no toma la ruta entre comillas dentro de una clave con puntos (`projects."<ruta>".trust_level`), y preguntaba por la carpeta. Claude sí pregunta la primera vez por una carpeta nueva: `pmt` no escribe en `~/.claude.json`.

### pmt equipo

Hace por los demás perfiles lo que `iniciar.sh` del kit, con el mismo mecanismo que `pmt pp`. Archivo: `team.go`.

1. **Agentes:** sin nombres, `pc tl dp devback devfront cr qap qa`; con nombres, esos (también `pp`, y los nombres de sesión como `devbackend`). Un nombre desconocido es error de uso.
2. **Requisitos y configuración:** los de `pmt pp`. Sin `.pmt/config.yaml` pregunta primero lo de `pmt pp` (proyecto y PP). Si falta algo del equipo (descripción, cadena, consulta, puertos o alguno de los ocho perfiles) pregunta solo eso; con `--configurar`, todo con los valores actuales. La descripción propone la del proyecto en PMT (`GET /api/projects/{id}`); la cadena no se muestra al volver a preguntar (Enter la conserva); la consulta debe llevar `<consulta>`; los puertos proponen 8090 y 5173. Los perfiles se muestran en una tabla con lo propuesto (Claude, `opus`, `medium` para TL, DP y CR; Codex, `gpt-6.1-terra`, `high` para PC, DevBackend, DevFrontend, QAP y QA) y se aceptan todos con Enter o se responden uno por uno. Los MCP extra de un perfil se conservan. La ruta no se pregunta: es la raíz.
3. **Entorno, workspace y prefijo:** como `pmt pp`. Si crea el workspace, su primera pestaña lleva el nombre del primer agente. El prefijo se pone si otro workspace ya tiene un agente con el nombre de alguno de los que se van a levantar.
4. **Por agente**, en orden: si ya corre en el workspace (por su nombre o por su pestaña, como en `pmt pp`), no lo toca. Si no, descarga su perfil, escribe `mcp-<perfil>.json` con Claude, valida sus MCP extra, crea su pestaña si falta y escribe en ella `pmt _lanzar <perfil> <raíz>`. Después espera a cada uno (hasta un minuto) y lo nombra.

### Comandos del equipo

`avisar`, `levantar`, `reiniciar`, `relanzar-dp` y `vigia` hacen lo que los scripts del kit del mismo nombre. Archivos: `commands.go` y `watcher.go`.

- **Proyecto y agentes:** la raíz es la carpeta más cercana hacia arriba con `.pmt/`, así que corren desde cualquier subcarpeta. El workspace es el de `config.yaml`, que debe existir. Un agente se nombra como en el kit (`tl`, `devback`), por su nombre de sesión o con el prefijo de `config.yaml` (`demo-tl`, como lo muestra `entorno.md`). Su pane es el del agente de herdr con su nombre en el workspace o, si no, el de la pestaña con su etiqueta; ya no hay `panes.env`. Si el agente no está en `config.yaml` o no tiene pestaña, error claro; `reiniciar`, `levantar` y `relanzar-dp` también fallan si en su pane solo está el shell.
- **`avisar`:** como `avisar.sh`. Sin agente en primer plano, sin compositor visible (`❯` de Claude, `›` de Codex) o con el agente `blocked`, no envía. Envía con `herdr agent prompt`; en Codex, un mensaje con `$` va con `pane send-text`, Esc si se abrió el selector de menciones y Enter aparte. Confirma (unos 4 s, mirando el pane) si el mensaje dejó el compositor y aparece arriba, o si el agente pasó a trabajar. Si sigue en el compositor manda Enter; si no hay rastro y el agente está quieto, reenvía una vez; ocupado y sin rastro no reenvía, para no duplicar. Sin confirmar, limpia el compositor, agrega `<fecha> aviso sin confirmar a <agente>: <mensaje>` a `.pmt/para-usuario.md` y sale con 1.
- **`reiniciar`:** espera hasta 90 s (cada 3 s) mientras el agente está `working` sin `Failed` en el título. En Claude manda `ctrl+u` y `/new`; en Codex, que tras `/new` descarta el primer mensaje, lo relanza: `ctrl+u`, `/quit`, espera hasta 10 s al shell, vuelve a descargar el perfil y corre el lanzador. Después lo nombra.
- **`levantar`:** reinicia; con el TL, si el vigía no corre (pid de `.pmt/vigia.pid` vivo), lo arranca en segundo plano en su propia sesión (`setsid`), con sus errores en `.pmt/vigia.log`; luego avisa con el encargo y lo nombra.
- **`relanzar-dp`:** guarda el effort en `perfiles.dp.effort` de `config.yaml` (el kit usaba `.agentes/dp.effort`) y relanza a DP; el lanzador lo toma de ahí, así que se conserva al reiniciar.
- **`vigia`:** como `vigia.sh`. Cada pulso (15 s) mira el estado de los agentes de `config.yaml`; si alguno está `blocked`, o `para-usuario.md` tiene una línea nueva, termina. Si nadie trabaja durante 150 s y desde el último aviso trabajó alguien que no sea PP, PC o TL, avisa al TL con `avisar`. Escribe su pid en `.pmt/vigia.pid` (lo borra al salir) y su bitácora, con fecha, en `.pmt/vigia.log` y en la salida. Si ya hay un vigía vivo, no arranca otro.

**Distinto del kit:** solo Claude y Codex, así que no están los casos de mcode y Qwen (`/retry` del vigía ante `Failed`, `pane run` para mcode, el `*` y `>` de Qwen en el compositor). El vigía y las esperas se cambian con `--pulso` y `--quieto` en vez de variables de entorno. Tras relanzar, en vez de esperar 12 s fijos, espera a que herdr detecte al agente.

## Membresía

Cada proyecto tiene miembros en `project_members (project_id, user_id, role)`, con un solo `owner` (índice único parcial) y `editor` o `viewer` ([dominio.md](dominio.md#miembros-y-roles)).

La comprobación vive en un solo lugar, `internal/app/membership.go`, y corre antes de cada herramienta MCP (`platform/mcp.Guard`) y de cada ruta REST (`platform/rest.Guard`, dentro de `Authenticate`):

1. Saca los IDs que recibe la operación por su nombre: `project_id`, `stage_id`, `task_id`, `task_ids`, `bug_id`, `finding_id`, `plan_id` (de pruebas o de desarrollo según la herramienta o la ruta) e `item_id` con su `item_kind`. En REST los toma de la ruta (el `{id}` según su primer segmento: `/api/stages/{id}` es una etapa), de la query y del cuerpo JSON, que deja intacto para el handler.
2. Para cada uno, `projects.Membership.RoleOf` consulta con `pmt_project_of` su proyecto y el rol del actor.
3. Si el elemento no existe o el actor no es miembro, responde el `not_found` de ese tipo, el mismo que daría el handler (`etapa no encontrada`). Si es viewer y llegó con token, `forbidden` lea o escriba (`projects.ErrViewerOnlyWeb`). Si es viewer con la cookie y la operación escribe (herramienta que no empieza con `list_` ni `get_`, o ruta que no es `GET`), `forbidden`. Si no, sigue al handler, que aplica sus reglas de perfil.

**Canal de la credencial.** `kernel.Actor.Channel` dice si el actor llegó con la cookie (`ChannelSession`) o con un token (`ChannelToken`). Lo fijan `rest.Authenticate` y `mcp.Authenticate`, que son los únicos que ven la credencial; los autenticadores de `auth` y `agents` no lo conocen. El valor vacío cuenta como token, así que lo que solo se permite desde el sitio web exige `ChannelSession` explícito. `projects.Role.Reaches(actor)` es la regla: owner y editor llegan con cualquiera; el viewer, solo con la cookie. La usan la guardia, `/api/events` (`guard.visible`) y `list_projects` (`domain.ReachableRoles`).

**Quién puede trabajar.** `assign_stage_lead`, `add_stage_member` y `assign_task` preguntan a `projects.Workers` (slice `projects/queries/checkworker`) si el nombre es un usuario activo; si lo es y no es owner ni editor del proyecto de la etapa, responde `conflict`. Si no es usuario, siguen validándolo como agente. El proyecto de la tarea llega en `kernel.Team.ProjectID`.

`list_projects` filtra en su consulta por miembro y por los roles que alcanzan con la credencial, y devuelve en cada proyecto `role`, el rol de quien consulta; `GET /api/projects/{id}` también lo trae (`projects.Members.Member`; sin usuario en el actor, como en las comprobaciones internas de otras features, queda vacío y se omite). Es lo que usa el frontend para mostrar el rol y ocultar al editor y al viewer las acciones del owner; `create_project` e `import_project` dejan al actor como owner en la misma sentencia o transacción. Al arrancar, después de crear al dueño, `projects.AdoptOrphans` lo deja como owner de los proyectos sin owner: los que existían cuando la migración `00027` no encontró usuarios.

Un ID que no es UUID no se revisa: el handler lo rechaza como inválido.

## Bloqueos

Las reglas están en [dominio.md](dominio.md#bloqueos-por-persona). Se reparten así:

- **Datos.** `stages` y `bugs` tienen `taken_by` (ID del usuario), `taken_by_name` (su `username`, copiado porque no cambia, para mostrarlo sin leer la tabla de `auth`) y `taken_at`, los tres nulos o los tres con valor (migración `00028`). En el dominio son `kernel.Hold`, embebido en `Stage` y `Bug` y en sus `Result`, que lo publican como `taken_by` y `taken_at`; `platform/postgres.HoldOf` y `HoldParams` lo traducen a columnas.
- **Tomar y soltar.** `Stage.ChangeStatus` y `Bug.ChangeStatus` (dominio) ajustan el `Hold` según el status destino; como `start_stage`, el cierre de `workflow` y `change_*_status` pasan por ellos, el `UPDATE` del status guarda el bloqueo en la misma sentencia. `taken_at` solo cambia cuando lo toma otra persona. Los triggers de `stages` y `bugs` emiten el aviso de cambio de siempre.
- **Escrituras.** `internal/app/membership.go`, después de la membresía y solo si la operación escribe, pregunta por cada ID a `projects.Membership.HoldOf`, que usa la función SQL `pmt_hold_of` (la etapa de una etapa, tarea, plan QAP, plan de desarrollo o hallazgo de tarea; el bug de un bug, plan o hallazgo de bug). Si la tiene otra persona, responde `stages.ErrTakenBy` o `bugs.ErrTakenBy`. Solo se salvan `release_stage`, `release_bug` y sus rutas. Los IDs de proyecto no se revisan.
- **Liberar.** Slices `stages/commands/release` y `bugs/commands/release`; el owner lo confirman con `projects.Owners` (`projects/queries/checkowner`). El `store` bloquea la fila, exige que siga en manos de quien la tenía al leerla y escribe el registro de status con el mismo status.
- **Arranque.** Si la `00028` no encontró usuarios, `app.adoptOrphans`, en el mismo arranque en que adopta los proyectos sin owner, deja tomadas por el dueño las etapas y los bugs en curso (`stages.AdoptHolds`, `bugs.AdoptHolds`).
- **Concurrencia.** La comprobación corre antes del handler, fuera de su transacción: una escritura que llega justo mientras otra persona toma la etapa puede pasar. Tomar sí es seguro, porque los cambios de status bloquean la fila y comparan el status de origen. Por eso la comprobación de perfil del handler ya no es lo primero: un no miembro recibe `not_found` aunque su perfil tampoco pudiera.

## Errores

- Los errores de negocio se crean en `domain` con `kernel.Invalid`, `NotFound`, `Conflict` o `Forbidden`.
- REST responde 400, 404, 409 o 403 con `{"error": "mensaje"}`. Cualquier otro error se registra en el log y responde 500 con un mensaje genérico.
- `kernel.Unauthorized` es la credencial ausente o inválida: REST responde 401.
- MCP antepone el tipo para que el agente lo distinga sin interpretar el texto: `invalid: `, `not_found: `, `conflict: `, `forbidden: ` o `unauthorized: `. Cualquier otro sale como `error interno`.
- `store` traduce las violaciones de llave foránea y de unicidad a errores de negocio.

## Datos

- 29 migraciones en `internal/platform/postgres/migrations/`, de `00001_create_projects` a `00029_create_cli_logins`. Las de usuarios son `00023_create_users_and_sessions`, `00024_create_api_tokens` y `00025_create_invitations`; `00026` quita las llaves foráneas de TL, equipo y dueño de tarea hacia `agents`. `00027` crea `project_members`, deja al dueño de la instancia como owner de los proyectos existentes, crea `pmt_project_of` y agrega el proyecto a los avisos de cambios. `00028` agrega los bloqueos a `stages` y `bugs`, deja lo que ya se trabajaba tomado por el dueño de la instancia y crea `pmt_hold_of`. `00029` crea `cli_logins`, los inicios de sesión del CLI ([CLI pmt](#cli-pmt)). Van embebidas en el binario y `postgres.Migrate` (goose provider) aplica las que falten al arrancar `pmt-server`, antes de abrir el puerto. Si una falla, el servidor no arranca. `pmt-server agent` no migra.
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
- Un `frontend.dist` o un `cli.dist` relativos se resuelven contra el directorio del archivo de configuración.

| Llave | Variable | Regla |
| --- | --- | --- |
| `server.addr` | `SERVER_ADDR` | obligatoria |
| `database.connection_string` | `DATABASE_CONNECTION_STRING` | obligatoria |
| `frontend.dist` | `FRONTEND_DIST` | opcional; si falta el compilado, el servidor arranca igual (ver Frontend servido) |
| `cli.dist` | `CLI_DIST` | opcional; carpeta con los binarios de `pmt` y `SHA256SUMS`. Si falta, el servidor arranca igual (ver [CLI pmt](#cli-pmt)) |
| `app.public_url` | `APP_PUBLIC_URL` | obligatoria; base de los enlaces de invitación, de la pantalla `/cli` y de `install.sh` |
| `auth.cookie_secure` | `AUTH_COOKIE_SECURE` | `true` marca la cookie como `Secure` (solo HTTPS) |
| `owner.email` | `OWNER_EMAIL` | dueño de la instancia; se crea al arrancar si no hay usuarios |
| `owner.password` | `OWNER_PASSWORD` | al menos 8 caracteres |
| `owner.username` | `OWNER_USERNAME` | su `username`; no puede ser nombre de un agente |

Si faltan `owner.email` u `owner.password` no se crea el dueño (avisa en el log si solo falta uno). Si hay usuarios, `owner.*` no se lee. Un dueño mal configurado con la base vacía detiene el arranque.

Como en DBP, una llave solo se puede sobrescribir con su variable si existe en el archivo: por eso los archivos llevan todas, aunque sea con `changeme # override: ...`.

| Archivo | En git | Contenido |
| --- | --- | --- |
| `backend/config/config.dev.yaml` | no | `127.0.0.1:8080`, la base `pmt`, `../../frontend/dist`, `cli.dist` `../../dist/cli`, `public_url` `http://127.0.0.1:8080`, cookie sin `Secure` y el dueño `saul` |
| `backend/config/config.prod.yaml` | sí | `0.0.0.0:8080`, `https://pmt.saulantonio.com`, cookie `Secure`, `frontend.dist: ""` porque la imagen fija `FRONTEND_DIST`, `cli.dist` `/app/cli`, y `changeme # override: ...` en la base y en `owner.*` |
| `frontend/e2e/harness/config.e2e.yaml` | sí | El backend de pruebas e2e: `127.0.0.1:8090`, `public_url` `http://localhost:5173`, y la base y el dueño de pruebas por variable ([frontend.md](frontend.md)) |

## Salud

`GET /healthz` responde 200 mientras el proceso atienda. `GET /readyz` responde 200 si la base contesta un ping en 2 s y 503 si no. Ninguna pasa por la autenticación.

## Imagen

El `Dockerfile` de la raíz compila el frontend con pnpm, `./cmd/pmt-server` con Go y el CLI con `scripts/build-cli.sh` (versión en `--build-arg PMT_VERSION`, por omisión `dev`), y deja en alpine el binario, `config/config.prod.yaml`, el `dist` en `/app/web` y los cuatro binarios de `pmt` con `SHA256SUMS` en `/app/cli`, con `ENV=prod FRONTEND_DIST=/app/web`. Escucha en 8080 y su `HEALTHCHECK` consulta `/readyz`. Al correrla hay que dar `DATABASE_CONNECTION_STRING` y, la primera vez, `OWNER_EMAIL`, `OWNER_PASSWORD` y `OWNER_USERNAME`. `docker-compose.prod.yml` la levanta con PostgreSQL 17 y pasa `SOURCE_COMMIT` (de Coolify) como `PMT_VERSION` ([despliegue.md](despliegue.md)).

## Pruebas del servidor

- Cada slice prueba su handler y sus adaptadores con dobles de la persistencia. `store` tiene pruebas propias.
- `internal/app/contracts_test.go`: las herramientas registradas coinciden con su documentación `.mcp.json`.
- `internal/app/permissions_test.go`: toda herramienta que no empiece con `list_` ni `get_` rechaza a PC y al actor vacío sin tocar la base. Una herramienta nueva de solo lectura debe seguir esa convención de nombre.
- `internal/app/membership_test.go`: recorre toda herramienta registrada y toda ruta que registran los `rest.go` de las features (leídas del código), con los IDs que aceptan según su esquema o su `*.openapi.yaml`, y exige `not_found` a quien no es miembro y `forbidden` a un viewer con token, sin llegar al handler; solo se salvan `create_project`, `import_project`, `list_projects`, `get_profile`, `GET`/`POST /api/projects` y las dos rutas de perfiles. También exige que contrato y código tengan las mismas rutas, que el viewer con cookie solo lea, y prueba owner, editor, viewer por token y por cookie, no miembro, token de agente y la visibilidad de los eventos.
- `internal/app/holds_test.go`: recorre toda herramienta y ruta de escritura sobre una etapa tomada y un bug tomado por otra persona y exige `conflict` con su nombre sin llegar al handler; solo se salvan `release_*` y las escrituras que solo reciben el proyecto, que deben ser exactamente las de su lista. También prueba la misma persona con otro perfil o con un token de agente, una etapa libre, lecturas y que la membresía va primero.
- `internal/app/routes_test.go`: 401 sin credenciales, cookie y token de usuario, token de agente, usuario desactivado, rutas públicas (incluidas las del CLI, `/install.sh` y `/descargas/`) y reparto de rutas.
- `internal/app/profiles_test.go`: `initialize` con `?profile=tl`, `qa`, `pc` y `devbackend` trae instrucciones cortas que nombran el perfil, `get_profile` y `.pmt/entorno.md`; sin perfil, con `user` o con token de agente, ninguna. `get_profile` con perfil devuelve su texto servido; sin perfil, `invalid` con `?profile=`; con token de agente, el de su perfil fijo. Las rutas de perfiles con token y cookie, sin distinguir mayúsculas, 404 y 401, y que REST y `get_profile` sirven lo mismo.
- `internal/features/profiles/feature_test.go` y `queries/get/handler_test.go`: las instrucciones de los nueve perfiles miden menos de 4 000 caracteres y piden `get_profile` antes que `.pmt/entorno.md`; ninguna sin perfil ni con token de agente; `HandleActor` con y sin perfil.
- `internal/features/profiles/catalog/catalog_test.go`: los nueve perfiles cargan, el texto servido no deja `{{…}}` ni tramos sin resolver, no nombra scripts del kit, menciona `.pmt/entorno.md`, y lo común abre pidiendo leerlo.
- `internal/features/auth/feature_test.go`: el recorrido de login, tokens, invitación, desactivación y cambio de contraseña por HTTP, sobre el repositorio en memoria de `auth/authtest`. `clilogin_test.go`: el login del CLI autorizado, cancelado, caducado, autorizado sin sondear a tiempo, uso único, tope de pendientes, perfil declarado, usuario desactivado y entradas inválidas; `domain/clilogin_test.go`, el código y los estados.
- `internal/cli/cli_test.go`: credenciales con permisos 600 y 700, servidor por omisión, alternativo e inválido, cada respuesta del sondeo, reintento por red, navegador que no abre, logout y whoami con token revocado o sin conexión. `internal/cli/pp_test.go`: raíz del proyecto, `config.yaml` de ida y vuelta, `.gitignore` sin duplicar, `entorno.md` con datos pendientes y completos, argumentos de Claude y Codex sin tokens, `mcp-pp.json`, MCP extra inválidos, detección de proyecto nuevo, y `pmt pp` con herdr y servidor falsos: primera vez, requisitos, PP que ya corre, proyecto nuevo adoptado, `--configurar`, perfil sin servidor con y sin copia, y prefijo por choque de nombre. `internal/cli/team_test.go`, con los mismos falsos (herdr simula la pantalla del agente): `pmt equipo` la primera vez, con nombres, con un agente que ya corre y con `--configurar` (sin mostrar la cadena); `avisar` confirmado, con reintento por Enter y por reenvío, ocupado sin rastro, sin agente, bloqueado, fallido con su línea en `para-usuario.md` y con `$` en Codex; `reiniciar` en Claude (con la espera de 90 s) y en Codex; `levantar tl` con el vigía; `relanzar-dp`; el vigía que avisa al TL y termina con una línea nueva o con un agente bloqueado; y errores claros desde una subcarpeta. `internal/platform/downloads`: con y sin binarios, nombres no permitidos y binarios compilados después de arrancar.
- Las pruebas de adaptadores REST y MCP de otras features actúan como usuario con `asUser` (`usuario_test.go`) o `platformmcp.ActAs`.

```bash
cd backend && go vet ./... && go test -race ./...
```

## Lo que no existe

Para no buscarlo:

- **Auditoría general** en `activity`: solo hay historiales de status, de cambios y de versión, y comentarios.
- **`kernel.Transactor`:** cada `store` abre su propia transacción.
- **Filtrado de herramientas MCP por perfil:** todos reciben las 67.
- **`idempotency_key`:** un reintento repite la operación.
- **Envío de correos:** el dueño copia el enlace de invitación y lo manda él.
- **Tomar o reasignar a mano** una etapa o un bug: el owner solo los libera.
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
