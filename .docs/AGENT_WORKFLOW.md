# Agent Workflow — PMT

## Development Approach

```
SDD → TDD → DDD
```

- Model the domain before writing infrastructure
- Write the test before writing the implementation
- Define the spec before writing the test

## Before Starting Any Task

1. Read the current state of the codebase — never assume structure
2. Identify all files that will be affected
3. For changes spanning more than 3 files, explain the approach and wait for confirmation

## Adding a New Feature (Vertical Slice)

Follow this order strictly:

1. **Migration** — `migrations/000NNN_create_{table}.up/down.sql`
2. **Domain** — entity, value objects, errors (if the aggregate is new)
3. **Application** — `repository.go` interface + `service.go` use cases
4. **Postgres adapter** — GORM model + repository implementation
5. **HTTP handler** — handler struct + driving port interface + response mapping
6. **Router** — register new routes
7. **main.go** — wire repository → service → handler

## Testing Requirements

- Table-driven tests for all domain behavior
- Tests validate behavior, not implementation details
- Run with `-race` flag: `go test -race ./...`
- A task is not complete until tests pass

## Commands

```bash
# Build
go build ./...

# Test
go test -race ./...

# Vet
go vet ./...

# Run server
go run ./cmd/api
```

## Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Go packages | lowercase, singular | `project`, `phase`, `issue` |
| Interfaces | noun or `[Verb]er` | `Repository`, `Storer` |
| Errors | `Err[Condition]` | `ErrNotFound`, `ErrInvalidName` |
| Constructors | `New[Type]` | `NewProject`, `NewPhase` |
| Reconstitute | `Reconstitute(...)` | rebuilds from persisted data |

## What Tests Must Validate

Tests validate **behavior**, not implementation details.

```
✅ "Given a project with an empty name, creation must fail with ErrInvalidName"
❌ "Calling NewProject invokes the validate() method"
```

A test that breaks when a private method is renamed is a bad test.

## Definition of Done

- `go build ./...` succeeds
- `go test -race ./...` passes
- `go vet ./...` reports no issues
- Server starts and endpoints respond correctly

## Phase Structure

Development is organized in phases under `.phases/phase-NNN/`. To understand what has been built and what is pending, read the specs in `.phases/`.
