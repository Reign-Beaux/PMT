---
title: Project Domain Entity Design
version: 1.0
date_created: 2026-04-01
owner: Saúl Antonio Morquecho Cela
tags: design, domain, entity, hexagonal, go
---

# Introduction

This specification defines the `Project` domain entity for the PMT (Project Management Tool) backend. It covers the entity structure, pure domain validation rules, the repository port interface, and the TDD strategy for verifying behavior.

## 1. Purpose & Scope

Define the `Project` entity as the root aggregate of the PMT domain. This entity must encapsulate its own invariants and expose a constructor that enforces them. Uniqueness validation is explicitly excluded from the entity and delegated to the application layer via a port interface.

**Intended audience**: Developer implementing the Go backend under Hexagonal Architecture guidance.

**Assumptions**:
- The entity lives in `internal/domain/project/` and has zero external dependencies.
- The port interface lives in `internal/application/` or alongside the use case that consumes it.
- No database, no HTTP, no framework is imported by the domain package.

## 2. Definitions

| Term | Definition |
|---|---|
| **Entity** | A domain object with identity, lifecycle, and business rules. |
| **Aggregate Root** | An entity that is the single entry point to a cluster of domain objects. `Project` is the aggregate root of its cluster. |
| **Port** | An interface defined inside the application boundary that an external adapter must implement. |
| **Driven Adapter** | An infrastructure component (e.g., PostgreSQL repository) that implements a port. |
| **Pure domain validation** | A validation rule the entity can enforce using only its own data — no I/O, no external calls. |
| **Application-level validation** | A rule that requires external data (e.g., DB lookup). Lives in the use case, not the entity. |
| **Value Object** | An immutable, self-validating object defined by its attributes, not identity. |
| **TDD** | Test-Driven Development: write a failing test first, then write the minimum code to pass it. |
| **Table-driven test** | Go testing pattern where test cases are defined as a slice of structs and iterated in a single test function. |

## 3. Requirements, Constraints & Guidelines

- **REQ-001**: The `Project` entity must have an `id` field of type `uuid.UUID`.
- **REQ-002**: The `Project` entity must have a `name` field of type `string`.
- **REQ-003**: `name` must not be empty or consist solely of whitespace.
- **REQ-004**: `name` must not exceed 255 characters after trimming.
- **REQ-005**: `name` must be stored trimmed — leading and trailing whitespace must be removed on creation and update.
- **REQ-006**: The entity must expose a constructor `NewProject(name string) (Project, error)` that enforces all domain rules.
- **REQ-007**: The entity must expose an `UpdateName(name string) error` method that enforces the same rules as the constructor.
- **REQ-008**: The `ProjectRepository` port must define `ExistsByName` to support uniqueness validation in the application layer.
- **REQ-009**: The `ProjectRepository` port must define `ExistsByNameExcludingID` to support uniqueness validation on update.

- **CON-001**: The `internal/domain` package must not import any package outside the Go standard library and `github.com/google/uuid`.
- **CON-002**: The entity must not receive a repository, context, or any interface as a constructor parameter.
- **CON-003**: Uniqueness validation must not be implemented inside the entity or its constructor.
- **CON-004**: Errors returned by domain validation must be sentinel errors (package-level `var Err... = errors.New(...)`) — not strings, not fmt.Errorf with dynamic content at the entity level.

- **GUD-001**: Prefer returning `(Project, error)` over panicking on invalid input.
- **GUD-002**: Sentinel errors should follow the convention `ErrProjectNameEmpty`, `ErrProjectNameTooLong`.
- **GUD-003**: The `Project` struct fields should be unexported; expose read access via methods if needed.

- **PAT-001**: Follow the factory constructor pattern: `NewProject` is the only valid way to create a `Project` in a valid state.

## 4. Interfaces & Data Contracts

### Entity

```go
// internal/domain/project/project.go

type Project struct {
    id   uuid.UUID
    name string
}

func NewProject(name string) (Project, error)
func (p *Project) UpdateName(name string) error
func (p Project) ID() uuid.UUID
func (p Project) Name() string
```

### Sentinel Errors

```go
// internal/domain/project/errors.go

var (
    ErrProjectNameEmpty   = errors.New("project name cannot be empty")
    ErrProjectNameTooLong = errors.New("project name cannot exceed 255 characters")
)
```

### Port Interface

```go
// internal/application/project/port.go  (or alongside the use case)

type ProjectRepository interface {
    Save(ctx context.Context, project project.Project) error
    GetByID(ctx context.Context, id uuid.UUID) (project.Project, error)
    ExistsByName(ctx context.Context, name string) (bool, error)
    ExistsByNameExcludingID(ctx context.Context, name string, excludeID uuid.UUID) (bool, error)
}
```

## 5. Acceptance Criteria

- **AC-001**: Given a valid name `"My Project"`, when `NewProject` is called, then it returns a `Project` with `Name() == "My Project"` and a non-zero UUID, and `error == nil`.
- **AC-002**: Given an empty string `""`, when `NewProject` is called, then it returns `ErrProjectNameEmpty`.
- **AC-003**: Given a string of only whitespace `"   "`, when `NewProject` is called, then it returns `ErrProjectNameEmpty`.
- **AC-004**: Given a name with 256 characters, when `NewProject` is called, then it returns `ErrProjectNameTooLong`.
- **AC-005**: Given a name with exactly 255 characters, when `NewProject` is called, then it returns no error.
- **AC-006**: Given a name `"  trimmed  "`, when `NewProject` is called, then `Name()` returns `"trimmed"`.
- **AC-007**: Given an existing `Project`, when `UpdateName` is called with a valid name, then `Name()` reflects the new value.
- **AC-008**: Given an existing `Project`, when `UpdateName` is called with an empty name, then it returns `ErrProjectNameEmpty` and `Name()` is unchanged.
- **AC-009**: Given an existing `Project`, when `UpdateName` is called with a name exceeding 255 characters, then it returns `ErrProjectNameTooLong` and `Name()` is unchanged.

## 6. Test Automation Strategy

- **Test Level**: Unit only. No database, no HTTP, no mocks required for entity tests.
- **Framework**: Go standard `testing` package.
- **Pattern**: Table-driven tests. Each acceptance criterion maps to one or more table entries.
- **File location**: `internal/domain/project/project_test.go`
- **Test naming**: `TestNewProject`, `TestProject_UpdateName`
- **Coverage requirement**: All sentinel errors must be reachable by at least one test case.
- **Race detection**: Tests must pass with `go test -race ./...`

### Table-driven test structure reference

```go
func TestNewProject(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr error
    }{
        // cases derived from acceptance criteria
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // assert
        })
    }
}
```

## 7. Rationale & Context

**Why unexported fields?**
Go does not have access modifiers per method. Unexported fields enforce that the only way to create or modify a `Project` is through its defined methods — ensuring invariants are always checked.

**Why sentinel errors instead of fmt.Errorf?**
Sentinel errors are comparable with `errors.Is()`. This allows callers (use cases, HTTP handlers) to check the specific error type without string matching. Dynamic messages (with field values) belong in the adapter layer, not the domain.

**Why is uniqueness a port concern?**
The entity only knows about itself. Checking uniqueness requires querying persistent storage — an external dependency. Injecting a repository into the entity would break the Hexagonal constraint that the domain has no outward dependencies. The use case owns this rule and enforces it before calling `Save`.

**Why `(Project, error)` instead of `(*Project, error)`?**
`Project` is a value with identity provided by UUID. Returning a value type makes the zero-value behavior explicit and avoids nil pointer handling at the call site. Pointer receivers on methods are still valid for mutation (`UpdateName`).

## 8. Dependencies & External Integrations

### Technology Platform Dependencies
- **PLT-001**: Go standard library (`errors`, `strings`) — for validation and sentinel error definition.
- **PLT-002**: `github.com/google/uuid` — for UUID generation at entity creation time. This is the only non-stdlib import permitted in the domain package.

## 9. Examples & Edge Cases

```go
// Valid creation
p, err := project.NewProject("PMT Backend")
// p.Name() == "PMT Backend", err == nil

// Empty name
_, err := project.NewProject("")
// errors.Is(err, project.ErrProjectNameEmpty) == true

// Whitespace-only name
_, err := project.NewProject("   ")
// errors.Is(err, project.ErrProjectNameEmpty) == true

// Name trimmed on creation
p, _ := project.NewProject("  Backend  ")
// p.Name() == "Backend"

// Name exactly at limit (255 chars)
name := strings.Repeat("a", 255)
p, err := project.NewProject(name)
// err == nil

// Name one over limit (256 chars)
name := strings.Repeat("a", 256)
_, err := project.NewProject(name)
// errors.Is(err, project.ErrProjectNameTooLong) == true
```

## 10. Validation Criteria

- [ ] `go build ./...` passes with no errors.
- [ ] `go test -race ./internal/domain/project/...` passes with no failures.
- [ ] `go vet ./...` reports no issues.
- [ ] All acceptance criteria AC-001 through AC-009 have a corresponding test case.
- [ ] No import of `database/sql`, `gorm`, `net/http`, or any adapter package exists in `internal/domain/`.
- [ ] Developer can explain why `ErrProjectNameEmpty` is defined as a sentinel and not returned via `fmt.Errorf`.
- [ ] Developer can explain why `UpdateName` does not return the updated entity but mutates via pointer receiver.

## 11. Related Specifications / Further Reading

- `.phases/phase-001/.requirements/specs/done/spec-infrastructure-go-backend-setup.md`
- `.docs/ARCHITECTURE.md`
- `.docs/WORKFLOW.md`
