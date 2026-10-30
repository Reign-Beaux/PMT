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
| `api/features/<feature>/` | Contratos públicos de la feature: OpenAPI para REST y esquemas de las herramientas MCP. Es la fuente de verdad de ambos. |
| `api/features/system/` | Contratos técnicos: salud, disponibilidad y versión. |
| `cmd/pmt-server/` | Punto de entrada del servidor. |
| `internal/app/` | Configuración, conexión de dependencias, arranque y cierre. Sin reglas de negocio. |
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
| Paquete de la slice | Verbo de la operación | `create`, `list` |
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

Las dependencias entre features van, en general, en este sentido:

```text
reviews, qa → tasks → stages → projects
tasks → agents
todas → activity
```

Cerrar una etapa necesita conocer sus tareas, en sentido contrario a la flecha `tasks → stages`. Ese caso se resuelve con una interfaz definida por la slice de `stages` e implementada mediante una query de `tasks`, conectada en `feature.go` o `app`. Nunca hay ciclos de importación.

## Reglas de dependencia

```text
cmd → app
app → feature.go de cada feature, kernel, platform
feature.go → sus slices, store, sqlcgen, SDK de MCP
handler.go → domain de su feature, kernel
rest.go → handler.go, platform/rest
mcp.go → handler.go, platform/mcp, SDK de MCP
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

## Identidad y permisos

- Cada agente tiene su propio token, que determina su `Actor`: agente y perfil.
- El usuario humano también es un `Actor`, con su propio perfil.
- El `Actor` se pasa explícitamente a cada handler.
- Cada agente recibe por MCP solo las herramientas de su perfil, lo que además reduce su contexto.
- Ese filtrado no sustituye la autorización: la validación real ocurre siempre en `domain` o en el handler.

Las responsabilidades de cada perfil están definidas en `perfiles.md`.

## Estados de una tarea

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

Dev corresponde a DevFrontend o DevBackend, según la tarea. El bloqueo es un atributo aparte, no un estado, para que al desbloquear la tarea siga donde estaba.

## Transacciones

- Los commands usan la interfaz `kernel.Transactor`, con `WithinTx(ctx, fn)`, implementada en `platform/postgres`.
- Un command que cruza features comparte la misma transacción mediante `ctx`.
- La auditoría se escribe en la misma transacción que el cambio, por lo que no puede haber cambios sin registro.

## Concurrencia

- Los agregados que varios agentes pueden modificar a la vez tienen una columna `version`, y sus escrituras envían `expected_version`. `Project` no la tiene: si dos escrituras coinciden, prevalece la última.
- Si la versión no coincide, el agente recibe un error de conflicto y debe releer antes de reintentar.
- Las operaciones críticas aceptan `idempotency_key`, única por actor, para que un reintento no duplique acciones.

## Eventos

La primera versión no tiene bus de eventos. Los agentes piden trabajo con una herramienta como `my_next_task`. La tabla de `activity` puede servir después como fuente de eventos si hace falta notificarlos.

## Ejemplo: QA aprueba una tarea

La operación es la slice `qa/commands/approve/`.

1. `mcp.go` recibe la llamada y `platform/mcp` resuelve el `Actor`.
2. `handler.go` abre la transacción y valida que el `Actor` sea QA y que el plan esté aprobado.
3. Registra la aprobación y pide a `tasks`, mediante una interfaz, la transición a `pending_close`.
4. `tasks` valida la tabla de transiciones y la `version`, y guarda.
5. `activity` registra la acción y la transacción confirma todo junto.

## Estado actual

Este documento describe el destino de la arquitectura. Hoy existe lo siguiente:

- **Implementado:** configuración, conexión a PostgreSQL, servidor REST, servidor MCP en `/mcp`, errores de negocio la feature `projects` con su CRUD por REST y la herramienta MCP `create_project`, y la feature `stages` con su CRUD por REST y la herramienta MCP `create_stage`.
- **Pendiente:** `Actor`, `Profile`, `Transactor`, autenticación, auditoría, filtrado de herramientas por perfil y las demás features.
- **Excepción temporal:** los commands de `projects` y `stages` no tienen autorización, transacción ni auditoría. Deben incorporarlas cuando existan `agents` y `activity`. Hasta entonces, el servidor escucha por defecto solo en `127.0.0.1` y no debe exponerse fuera de la máquina local.
- **Carencia conocida:** `store` no tiene pruebas automáticas contra PostgreSQL.

## Orden de construcción

1. `projects` y `stages`. Hechos, salvo la excepción temporal.
2. `agents` con autenticación por token, y `Actor` y `Profile` en `kernel`.
3. `tasks`, con su tabla de transiciones.
4. `activity`, integrada a las transiciones.
5. Primera operación completa por MCP, con una prueba de dos agentes en conflicto.
6. `reviews` y `qa`.

Una operación debe funcionar de entrada a salida antes de extender el patrón a otras.

## Procedimiento para implementar una operación

1. Decide si es un command o una query, y a qué feature pertenece.
2. Define entradas, resultados, errores y reglas, y documenta el contrato en `api/features/<feature>/`.
3. Crea la carpeta de la slice en `commands/<operacion>/` o `queries/<operacion>/`.
4. Si la operación necesita reglas nuevas, impleméntalas en `domain/`.
5. Implementa `handler.go`, con sus interfaces mínimas, autorización y límites transaccionales.
6. Añade la persistencia en `store/` y las migraciones necesarias.
7. Añade `rest.go` o `mcp.go`.
8. Registra la slice en `feature.go`.
9. Ejecuta `go test` y comprueba reglas de negocio, persistencia y correspondencia con el contrato.

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
