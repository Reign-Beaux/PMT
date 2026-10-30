# Arquitectura del backend de PMT

Este documento define la arquitectura del backend de PMT. Es obligatorio para los agentes y desarrolladores que trabajen en él. Cualquier excepción debe documentarse aquí.

## Visión general

El backend es un **monolito modular en Go**. Un solo proceso sirve la API REST al frontend y el MCP a los agentes, y es el único dueño de la base de datos. Ambas entradas llaman a los mismos casos de uso, por lo que las reglas de negocio no se duplican.

```text
Frontend React ── REST ──┐
                         ├── pmt-server ── casos de uso ── PostgreSQL
Agentes en herdr ─ MCP ──┘
```

| Necesidad | Elección |
| --- | --- |
| Lenguaje | Go |
| Base de datos | PostgreSQL |
| Acceso a datos | `pgx` y `sqlc` |
| Configuración | Viper |
| Migraciones | `goose` |
| MCP | SDK oficial de MCP para Go, con Streamable HTTP |
| Pruebas unitarias | Paquete estándar `testing` con `go test` |

## Estructura

```text
backend/
├── api/
│   └── features/
│       ├── work/
│       ├── quality/
│       ├── agents/
│       ├── activity/
│       └── system/
├── cmd/
│   └── pmt-server/
├── internal/
│   ├── app/
│   ├── features/
│   │   └── <modulo>/
│   │       ├── domain/
│   │       ├── usecase/
│   │       ├── store/
│   │       │   ├── queries/
│   │       │   └── sqlcgen/
│   │       └── transport/
│   │           ├── rest/
│   │           └── mcp/
│   ├── kernel/
│   └── platform/
│       ├── rest/
│       ├── mcp/
│       └── postgres/
│           └── migrations/
├── scripts/
└── bin/
```

Todo módulo tiene las capas `domain`, `usecase`, `store` y `transport`, sin excepciones. Solo puede omitir un transporte que no use.

## Responsabilidades

| Ruta | Responsabilidad |
| --- | --- |
| `api/features/<modulo>/` | Contratos públicos del módulo: OpenAPI para REST y esquemas de las herramientas MCP. Es la fuente de verdad de ambos. |
| `api/features/system/` | Contratos técnicos: salud, disponibilidad y versión. |
| `cmd/pmt-server/` | Punto de entrada del servidor. |
| `internal/app/` | Configuración, composición de dependencias, registro de rutas y herramientas, arranque y cierre. Sin reglas de negocio. |
| `internal/features/<modulo>/domain/` | Entidades, invariantes, errores y reglas de negocio. |
| `internal/features/<modulo>/usecase/` | Operaciones de negocio, autorización y puertos que necesitan. |
| `internal/features/<modulo>/store/` | Implementación de los puertos de persistencia y traducción entre registros y dominio. |
| `internal/features/<modulo>/store/queries/` | Consultas SQL del módulo para `sqlc`. |
| `internal/features/<modulo>/store/sqlcgen/` | Código generado por `sqlc`. No se edita a mano. |
| `internal/features/<modulo>/transport/rest/` | Adaptación de peticiones REST a casos de uso. |
| `internal/features/<modulo>/transport/mcp/` | Declaración de las herramientas MCP del módulo y adaptación a casos de uso. |
| `internal/kernel/` | Conceptos compartidos con la misma semántica en varios módulos: `Actor`, `Profile` e identificadores. |
| `internal/platform/rest/` | Servidor HTTP, middleware y manejo técnico de errores. |
| `internal/platform/mcp/` | Servidor MCP, autenticación y filtrado de herramientas por perfil. Sin lógica de negocio. |
| `internal/platform/postgres/` | Conexión, transacciones y ejecución de migraciones. |
| `internal/platform/postgres/migrations/` | Cambios versionados del esquema de datos. |
| `scripts/` | Automatización de desarrollo y operación. |
| `bin/` | Binarios locales, excluidos del control de versiones. |

## Configuración

- La configuración se lee con Viper desde `config.yaml`, en la raíz de `backend/`. Este archivo contiene secretos y nunca se versiona.
- `config.example.yaml` es la plantilla versionada.
- Las variables de entorno `DATABASE_URL` y `SERVER_ADDR` tienen prioridad sobre `database.url` y `server.addr` del archivo.
- `internal/app/` es el único lugar que lee la configuración.

## Datos

- Las migraciones se escriben en `internal/platform/postgres/migrations/` y se ejecutan con `go tool goose`, que recibe la URL por `DATABASE_URL`.
- `sqlc.yaml` tiene una entrada por cada módulo con consultas. Todas leen el esquema de esas migraciones.
- Tras cambiar migraciones o consultas se ejecuta `go tool sqlc generate`.

## Módulos

| Módulo | Contiene |
| --- | --- |
| `work` | Proyectos, etapas, tareas, estados, asignaciones y propuestas de cambio. |
| `quality` | Hallazgos, revisiones de CR, planes QAP, casos y ejecuciones de QA. |
| `agents` | Agentes, perfiles y credenciales. |
| `activity` | Auditoría, comentarios y bloqueos. |

Proyectos, etapas y tareas forman un solo módulo a propósito: cerrar una etapa depende de sus tareas y una tarea depende de su etapa. Separarlos crearía un ciclo sin ningún beneficio.

Las dependencias entre módulos van en un solo sentido:

```text
quality → work → agents
quality, work, agents → activity
```

## Reglas de dependencia

```text
cmd → app
app → todo lo demás
transport → usecase, domain, kernel, platform
store → puertos de usecase, domain, kernel, platform
usecase → domain, kernel
domain → kernel
kernel → nada
platform → kernel
```

- `domain` no importa transporte, persistencia ni frameworks de infraestructura.
- `usecase` no importa implementaciones de `store`, `transport` o `platform`.
- `platform` no importa módulos de negocio.
- `kernel` solo incorpora conceptos cuya semántica compartida esté comprobada.
- `transport` valida el formato de las entradas. Las invariantes de negocio se verifican en `domain` o `usecase`.

### Colaboración entre módulos

1. El módulo consumidor define el puerto en su `usecase`.
2. El módulo proveedor expone un caso de uso.
3. `app` conecta ambos.

Ningún módulo lee las tablas ni importa los detalles internos de otro.

## Identidad y permisos

- Cada agente tiene su propio token, que determina su `Actor`: agente y perfil.
- El usuario humano también es un `Actor`, con su propio perfil.
- El `Actor` se pasa explícitamente a cada caso de uso.
- Cada agente recibe por MCP solo las herramientas de su perfil, lo que además reduce su contexto.
- Ese filtrado no sustituye la autorización: la validación real ocurre siempre en `domain` o `usecase`.

Las responsabilidades de cada perfil están definidas en `perfiles.md`.

## Estados de una tarea

Una tabla de transiciones en `work/domain` define qué perfil puede hacer cada cambio:

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

- `usecase` usa el puerto `Transactor.WithinTx(ctx, fn)`, implementado en `platform/postgres`.
- Una operación que cruza módulos comparte la misma transacción mediante `ctx`.
- La auditoría se escribe en la misma transacción que el cambio, por lo que no puede haber cambios sin registro.

## Concurrencia

- Los agregados que varios agentes pueden modificar a la vez tienen una columna `version`, y sus escrituras envían `expected_version`. `Project` no la tiene: si dos escrituras coinciden, prevalece la última.
- Si la versión no coincide, el agente recibe un error de conflicto y debe releer antes de reintentar.
- Las operaciones críticas aceptan `idempotency_key`, única por actor, para que un reintento no duplique acciones.

## Eventos

La primera versión no tiene bus de eventos. Los agentes piden trabajo con una herramienta como `my_next_task`. La tabla de `activity` puede servir después como fuente de eventos si hace falta notificarlos.

## Ejemplo: QA aprueba una tarea

1. `quality/transport/mcp` recibe la llamada y `platform/mcp` resuelve el `Actor`.
2. `quality/usecase` abre la transacción y valida que el `Actor` sea QA y que el plan esté aprobado.
3. Registra la aprobación y pide a `work`, mediante un puerto, la transición a `pending_close`.
4. `work` valida la tabla de transiciones y la `version`, y guarda.
5. `activity` registra la acción y la transacción confirma todo junto.

## Orden de construcción

1. Base: `kernel`, `platform` y `agents` con autenticación por token.
2. `work`, con tareas y su tabla de transiciones.
3. `activity`, integrada a las transiciones.
4. Primera operación completa por MCP, con una prueba de dos agentes en conflicto.
5. REST para el frontend.
6. `quality`.

Una operación debe funcionar de entrada a salida antes de extender el patrón a otras.

## Procedimiento para implementar una operación

1. Define entradas, resultados, errores y reglas, y documenta el contrato en `api/features/<modulo>/`.
2. Implementa entidades e invariantes en `domain/`.
3. Implementa el caso de uso en `usecase/`, con sus puertos mínimos, autorización y límites transaccionales.
4. Añade la persistencia en `store/` y las migraciones necesarias.
5. Añade las entradas en `transport/rest/` o `transport/mcp/`.
6. Conecta las dependencias en `app/`.
7. Ejecuta `go test` y comprueba reglas de negocio, persistencia y correspondencia con el contrato.

## Criterios de aceptación

Una implementación está lista cuando:

- La operación puede seguirse desde su entrada hasta su resultado.
- Las dependencias respetan las reglas de este documento.
- Los permisos se validan en `domain` o `usecase`, y no solo en el transporte.
- Las escrituras protegen contra conflictos y dejan auditoría.
- Las pruebas pasan.

## Supuestos

- **PostgreSQL:** elegida por su concurrencia y sus restricciones fuertes.
- **Una sola organización:** no hay multi-tenencia. Agregarla después obligaría a tocar casi todas las tablas.
- **Login del usuario humano:** el detalle queda pendiente de definir.
