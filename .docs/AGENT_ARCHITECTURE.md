# Agent Architecture — PMT

## Architectural Style

Hexagonal Architecture (Ports & Adapters). The domain is the center — it has zero knowledge of HTTP, databases, or any external concern. Everything outside communicates through interfaces called Ports, implemented by Adapters.

```
                ┌─────────────────────────────────────┐
HTTP Request ──►│         A D A P T E R S             │
                │         (Driving / Left)            │
                │  ┌───────────────────────────────┐  │
                │  │       A P P L I C A T I O N   │  │
                │  │  ┌──────────────────────────┐ │  │
                │  │  │       D O M A I N        │ │  │
                │  │  │  Entities, Value Objects │ │  │
                │  │  │  Aggregates              │ │  │
                │  │  └──────────────────────────┘ │  │
                │  │  Use Cases (Port definitions) │  │
                │  └───────────────────────────────┘  │
                │         A D A P T E R S             │
                │         (Driven / Right)            │◄── PostgreSQL
                └─────────────────────────────────────┘
```

## Application vs Domain

**Application is not Domain.** Application knows _what to do_; Domain knows _how things work_.

| Layer           | Responsibility                                                              |
| --------------- | --------------------------------------------------------------------------- |
| **Domain**      | Pure business rules. Entities, Value Objects, Aggregates. No orchestration. |
| **Application** | Orchestration only. Sequences domain operations. No core business rules.    |

Both layers: zero imports from the outside world (no HTTP, no SQL, no GORM).

## Ports

| Type         | Also called        | Direction        | Defined in                     |
| ------------ | ------------------ | ---------------- | ------------------------------ |
| Driving Port | Primary / Input    | Outside → Domain | handler package (consumer)     |
| Driven Port  | Secondary / Output | Domain → Outside | application package (consumer) |

Go rule: interfaces are defined where they are _consumed_, not where they are implemented. This is not style — it is how Go's implicit interfaces work.

## Layers

### `internal/domain/`

Pure business logic. Entities, Value Objects, domain errors. No framework imports — only stdlib.

Each aggregate lives in its own package with:

- Entity (constructor + behavior methods)
- Value Objects (validated on construction via `NewX()`)
- `errors.go` — typed sentinel errors (`ErrNotFound`, `ErrInvalidX`)
- `Reconstitute()` function — rebuilds the entity from persisted data without going through the constructor

### `internal/application/`

Orchestration. Use cases that sequence domain operations. One package per aggregate containing:

- `service.go` — the use case struct with all operations (Create, GetByID, List, Update, Delete)
- `repository.go` — the driven port interface (defined here, in the consumer)

Application imports domain. Application defines the repository interface it needs — not the other way around.

### `internal/adapter/driving/httpserver/`

HTTP adapter (Chi router). Handlers parse requests, call the application service, and write responses. Zero business logic.

Each handler file defines its own service interface (driving port) — defined in the consumer, not the provider.

### `internal/adapter/driven/postgres/`

PostgreSQL adapter (GORM). Contains:

- GORM model structs (struct tags stay here — never leak to domain)
- Repository implementations
- `toXModel()` / `toXDomain()` mapping functions
- SQL migration files (`migrations/*.sql`)

## Dependency Rule

```
adapter/driving → application → domain
adapter/driven  ← application ← domain
```

Domain imports nothing outside stdlib. Application imports only domain. Adapters import application and domain — never each other.

## Key Patterns

### Ports (interfaces)

- Driven port (repository): defined in `application/{aggregate}/repository.go`
- Driving port (service): defined in `adapter/driving/httpserver/handler/{aggregate}.go`

### Value Objects

Constructed via `NewX(input)` — validates on creation. Internal `isValid()` method (unexported). Entities re-validate value objects they receive (deliberate double validation).

### GORM Policy

GORM is confined to `adapter/driven/postgres`. GORM struct tags, `gorm.Model`, and any GORM type must never appear in `domain` or `application`. Domain entities are mapped to/from GORM models inside the adapter.

### Error Mapping

Domain errors are sentinel values (`errors.Is`). HTTP handlers map domain errors to status codes — the domain never knows about HTTP.

## Golden Rules

> **The domain defines the model. The database adapts to it.**

Schema design follows domain design. If a domain entity is being changed because of a database constraint, the direction is backwards.

> **No abstraction without a real, present problem.**

Do not introduce ports, interfaces, or layers speculatively. Add the abstraction when a second implementation — or a test — demands it.

> **The architecture is allowed to be incomplete. It evolves with the domain.**

An early design decision that proves wrong is not a failure — refusing to revise it is.

## Stack

| Concern    | Choice                                     |
| ---------- | ------------------------------------------ |
| Language   | Go                                         |
| Router     | Chi                                        |
| Database   | PostgreSQL                                 |
| ORM        | GORM (confined to adapter/driven/postgres) |
| Migrations | golang-migrate (SQL files, embedded)       |
| Testing    | Standard Go, table-driven                  |
