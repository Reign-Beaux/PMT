# Architecture — PMT (Project Management Tool)

## Architectural Style: Hexagonal Architecture (Ports & Adapters)

Hexagonal Architecture — coined by Alistair Cockburn — places the **application domain at the center**. Everything outside (HTTP, database, message queues, CLI) is an external actor that communicates with the domain through well-defined contracts called **Ports**, implemented by **Adapters**.

```
                    ┌─────────────────────────────────────┐
   HTTP Request ───►│                                     │
   CLI Command ───►│           A D A P T E R S            │
                   │           (Driving / Left)            │
                   │  ┌───────────────────────────────┐   │
                   │  │         A P P L I C A T I O N │   │
                   │  │  ┌─────────────────────────┐  │   │
                   │  │  │       D O M A I N       │  │   │
                   │  │  │  Entities, Value Objects │  │   │
                   │  │  │  Aggregates, Domain Svc  │  │   │
                   │  │  └─────────────────────────┘  │   │
                   │  │  Use Cases (Port definitions)  │   │
                   │  └───────────────────────────────┘   │
                   │           A D A P T E R S            │
                   │           (Driven / Right)            │◄─── PostgreSQL
                   └─────────────────────────────────────┘
```

---

## Core Concepts

### The Hexagon (Domain + Application)

The center of the system. Isolated from all infrastructure concerns. Composed of two distinct layers that must not be confused:

| Layer | Responsibility |
|-------|---------------|
| **Domain** | Pure business rules. Entities, Value Objects, Aggregates, Domain Services. No orchestration. |
| **Application** | Orchestration and flow coordination. Use cases sequence domain operations and apply flow rules (authorization checks, input validation at the boundary). Does not contain core business rules — those belong in Domain. |

**Application is not Domain.** Application knows what to do; Domain knows how things work.

Both layers share one rule:
- Zero imports from the outside world (no HTTP, no SQL, no GORM)
- Defines what it needs from the outside via **Ports** (Go interfaces)
- Never knows _how_ data is persisted or _how_ requests arrive

### Ports

Ports are **interfaces** — contracts that define how the outside world can interact with the application (or how the application interacts with the outside world).

Two kinds:

| Type | Also called | Direction | Example |
|------|-------------|-----------|---------|
| Driving Port | Primary / Input | Outside → Domain | `CreateProjectUseCase` interface |
| Driven Port | Secondary / Output | Domain → Outside | `ProjectRepository` interface |

**Go-idiomatic rule**: interfaces are defined where they are *consumed*, not where they are implemented.

- A use case that needs a repository → defines `Repository` interface in the use case's own package
- An HTTP handler that calls a use case → defines the use case interface in the handler's package

This is not optional style — it is how Go's implicit interfaces are designed to be used. Defining interfaces in the provider package (the .NET way) creates unnecessary coupling.

**Note on the `application/port/` directory**: this is an *organizational* convention, not a centralized contract registry. It groups related interfaces for navigability. It must not become a shared "interface hub" that everything imports — that would replicate the .NET service-interface pattern and defeat the purpose. Each interface should still conceptually belong to its consumer.

### Adapters

Adapters implement Ports. They translate between the external world and the domain.

| Adapter Type | Examples |
|-------------|----------|
| Driving (Left) | HTTP handler (Gin/Chi), CLI, gRPC server |
| Driven (Right) | PostgreSQL repository (GORM), in-memory repo (tests), mock |

**Rule**: Adapters know about the domain. The domain never knows about adapters.

---

## Package Structure

```
project-management-tools/
├── cmd/
│   └── api/
│       └── main.go                        # Wiring: connect adapters to ports
│
├── internal/
│   ├── domain/
│   │   ├── project/
│   │   │   ├── project.go                 # Entity / Aggregate Root
│   │   │   ├── project_test.go
│   │   │   ├── value_objects.go           # ProjectName, Status, etc.
│   │   │   └── errors.go                  # ErrNotFound, ErrInvalidName, etc.
│   │   ├── spec/
│   │   ├── requirement/
│   │   └── shared/                        # Shared Value Objects (ID, etc.)
│   │
│   ├── application/
│   │   ├── port/
│   │   │   ├── input/                     # Driving Ports (use case interfaces)
│   │   │   │   └── create_project.go      # type CreateProject interface { ... }
│   │   │   └── output/                    # Driven Ports (repository interfaces)
│   │   │       └── project_repository.go  # type ProjectRepository interface { ... }
│   │   └── usecase/
│   │       └── create_project.go          # Implements driving port
│   │
│   └── adapter/
│       ├── driving/
│       │   └── http/
│       │       ├── handler/
│       │       │   └── project_handler.go # HTTP adapter (calls input port)
│       │       ├── middleware/
│       │       └── router.go
│       └── driven/
│           └── postgres/
│               └── project_repo.go        # DB adapter (implements output port)
│
├── .docs/
├── .phases/
├── go.mod
└── go.sum
```

---

## Dependency Rule

```
adapter/driving  →  application/port/input  →  domain
adapter/driven   ←  application/port/output ←  domain
                                   ↑
                         Interfaces defined here
                         Implementations outside
```

The `domain` package imports nothing outside `stdlib`.  
The `application` package imports only `domain`.  
`adapter` packages import `application` and `domain` — never each other.

---

## Domain Model (initial)

| Concept      | Type            | Notes                                      |
|--------------|-----------------|--------------------------------------------|
| Project      | Aggregate Root  | Container for all project artifacts        |
| Spec         | Entity          | Belongs to a Project                       |
| Stage        | Entity          | Ordered phases within a Spec               |
| Requirement  | Entity          | Belongs to a Stage                         |
| UserStory    | Entity          | Derived from a Requirement                 |
| Status       | Value Object    | Shared across multiple aggregates          |
| ProjectName  | Value Object    | Validated, non-empty string                |

> The domain model will evolve. New concepts emerge as development progresses.

---

## Key Principles

- **Domain isolation**: no framework, no ORM, no HTTP in domain or application packages
- **Ports as contracts**: use case interfaces are driving ports; repository interfaces are driven ports
- **Rich domain**: entities enforce their own invariants — they are not data bags
- **Adapters are thin**: HTTP handlers parse and delegate; DB repos translate and persist
- **Testability by design**: driven adapters can be swapped for in-memory fakes in tests
- **Explicit errors**: domain errors are typed values, not strings, not panics

### Value Object Construction Pattern

Value Objects are constructed exclusively through a `NewX` constructor that validates on creation. Internal validation logic is **unexported** (`isValid()`, not `IsValid()`), accessible only within the domain package.

Entities that receive Value Objects apply **deliberate double validation** — the entity re-checks the Value Object before accepting it. Rationale: Go's zero value makes it impossible to guarantee a struct was constructed through its constructor. The cost (a simple field check) is trivial; the benefit is that domain invariants hold regardless of how the Value Object was instantiated.

```go
// unexported — validation is an internal concern of the domain package
func (n ProjectName) isValid() bool {
    return strings.TrimSpace(n.value) != ""
}

// NewProject re-validates — deliberate, not accidental
func NewProject(name ProjectName) (Project, error) {
    if !name.isValid() {
        return Project{}, ErrInvalidName
    }
    // ...
}
```

This policy is suspended when the cost of validation is non-trivial (e.g., involves I/O or complex rules). In those cases, trust the constructor and document the assumption explicitly.

---

## Tech Stack

| Concern        | Choice                         |
|----------------|--------------------------------|
| Language       | Go 1.26+                       |
| HTTP Router    | Gin or Chi (TBD per phase)     |
| Database       | PostgreSQL                     |
| ORM            | GORM — constrained (see below) |
| Testing        | Standard Go + table-driven     |
| Linting        | golangci-lint                  |

---

## GORM Policy

GORM is permitted **only** inside `adapter/driven/postgres`. These rules are non-negotiable:

- GORM model structs (`gorm.Model`, struct tags) must **never** appear in `domain` or `application`
- Domain entities are mapped to/from GORM models inside the adapter — never passed directly
- The domain defines the model. The database adapts to it, not the other way around
- If GORM's requirements (e.g., `ID uint`, auto-migrations) conflict with domain design, GORM loses
- GORM may be replaced by `sqlx` or raw `database/sql` if it proves to be more hindrance than help

---

## Golden Rules

> **The domain defines the model. The database adapts to it.**

Schema design follows domain design. If you are changing a domain entity because of a database constraint, you have the direction backwards.

> **No abstraction without a real, present problem.**

Do not introduce ports, interfaces, or layers speculatively. If there is currently one repository implementation and no test double is needed yet, the interface can wait. Add the abstraction when the second implementation — or the test — demands it.

> **The architecture is allowed to be incomplete. It evolves with the domain.**

The initial structure is a starting point, not a commitment. As new domain concepts emerge, the architecture adapts. An early design decision that proves wrong is not a failure — refusing to revise it is.

---

## Hexagonal vs Clean Architecture (reference)

These two are often confused. Key differences in terminology:

| Concept | Hexagonal | Clean Architecture |
|---------|-----------|--------------------|
| Center | Domain + Application | Entities + Use Cases |
| Contracts | Ports (interfaces) | Interface Adapters |
| Outside code | Adapters | Frameworks & Drivers / Gateways |
| Direction metaphor | Left (driving) / Right (driven) | Inner / Outer rings |

Both enforce the same core rule: **dependencies point inward**. Hexagonal makes the two adapter directions (driving vs driven) more explicit.

---

## Anti-patterns Explicitly Forbidden

- Business logic in HTTP handlers (driving adapters)
- **Fat use cases**: all logic living in the application layer instead of the domain — use cases orchestrate, they do not decide
- Anemic domain models (entities that are just data bags with no behavior)
- Domain packages importing `adapter`, `gorm`, `gin`, or `net/http`
- Using `panic` for business error flow
- Global mutable state
- Importing driven adapters directly from use cases (must go through the port interface)
- Defining interfaces in the provider package instead of the consuming package
- Letting GORM struct tags or constraints leak into domain entities
