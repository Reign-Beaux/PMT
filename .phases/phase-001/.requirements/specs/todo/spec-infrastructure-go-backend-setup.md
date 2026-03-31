---
title: Go Backend — Project Base Setup
version: 1.0
date_created: 2026-03-30
owner: Developer
tags: infrastructure, setup, http, adapter
---

# Introduction

This spec defines the minimum necessary to transform the existing Go module into a functional HTTP backend. The result must be a server that starts, responds to requests, and is structurally aligned with the Hexagonal Architecture defined in `.docs/ARCHITECTURE.md`.

No domain logic. No database. No patterns beyond what is strictly necessary.

## 1. Purpose & Scope

**Purpose**: Establish the foundational layer of the PMT backend — a running HTTP server with a single health endpoint, correctly wired under Hexagonal Architecture constraints.

**Scope**: This spec covers only:
- Go module dependency setup
- HTTP adapter configuration (router + server)
- Application entry point (`cmd/api/main.go`)
- Folder structure for what is built in this phase

**Out of scope**: database, ORM, domain entities, use cases, authentication, CQRS, any application logic.

**Audience**: The developer implementing this spec. Assumes familiarity with Go modules and basic HTTP concepts.

## 2. Definitions

| Term | Definition |
|------|-----------|
| **Driving Adapter** | External actor that initiates interaction with the application. HTTP is a driving adapter. |
| **Port** | Interface contract defined by the consuming layer. Not relevant yet in this phase — introduced when use cases exist. |
| **Router** | Component responsible for mapping HTTP routes to handlers. Lives inside the HTTP driving adapter. |
| **Handler** | Function that receives an HTTP request and writes a response. Contains no business logic. |
| **Wiring** | The act of connecting adapters to ports in `main.go`. `main.go` is exclusively a composition root. |
| **Health endpoint** | A route that returns a fixed response confirming the server is running. No logic involved. |
| **Chi** | Lightweight HTTP router for Go that composes with `net/http` stdlib without introducing its own context type. |

## 3. Requirements, Constraints & Guidelines

- **REQ-001**: The project must add `chi` as the HTTP router dependency via `go get`.
- **REQ-002**: The server must start on a configurable port. The port must be read from an environment variable (`PORT`). If not set, default to `8080`.
- **REQ-003**: A `GET /health` route must return HTTP 200 and a JSON body: `{"status": "ok"}`.
- **REQ-004**: `main.go` must contain only wiring — instantiation and connection of components. Zero logic.
- **REQ-005**: The router setup must live in `internal/adapter/driving/http/router.go`, not in `main.go`.
- **REQ-006**: The health handler must be a separate function or type in `internal/adapter/driving/http/handler/`.
- **REQ-007**: The server must use `http.Server` from stdlib with explicit `ReadTimeout` and `WriteTimeout` configured.

- **CON-001**: No Gin. No gorilla/mux. No other router. Chi only.
- **CON-002**: No business logic anywhere in this phase.
- **CON-003**: No global variables. No `init()` functions.
- **CON-004**: `main.go` must not import `chi` directly. Router wiring belongs in the adapter layer.
- **CON-005**: The health handler must not hardcode the JSON string. Use `encoding/json` or a struct.

- **GUD-001**: Package names must be short, lowercase, singular, no underscores — Go convention.
- **GUD-002**: Errors from `http.ListenAndServe` must be handled explicitly — not ignored.
- **GUD-003**: `context.Context` is not needed yet in handlers, but do not block it — use `r.Context()` if you need to pass it.

## 4. Interfaces & Data Contracts

### Health endpoint

```
GET /health
Response: 200 OK
Content-Type: application/json

{
  "status": "ok"
}
```

No request body. No query parameters. No headers required.

### Folder structure produced by this spec

```
cmd/
  api/
    main.go
internal/
  adapter/
    driving/
      http/
        handler/
          health.go
          health_test.go
        router.go
go.mod
go.sum
```

No other directories are created. Do not create `domain/`, `application/`, or `driven/` yet — they do not exist in this phase.

## 5. Acceptance Criteria

- **AC-001**: Given the project directory, when `go build ./...` is run, then it must compile with zero errors and zero warnings.
- **AC-002**: Given the compiled binary, when the server is started and `GET /health` is called, then the response must be HTTP 200 with body `{"status":"ok"}` and `Content-Type: application/json`.
- **AC-003**: Given no `PORT` environment variable, when the server starts, then it must listen on port `8080`.
- **AC-004**: Given `PORT=9090` in the environment, when the server starts, then it must listen on port `9090`.
- **AC-005**: Given the health handler, when `go test ./...` is run, then all tests must pass — including a table-driven test that covers at least: correct status code, correct response body, correct Content-Type header.
- **AC-006**: Given `go vet ./...`, when run, then it must report zero issues.
- **AC-007**: Given `main.go`, when reviewed, then it must contain no logic beyond: creating the router, creating the server, calling `ListenAndServe`. No conditionals. No parsing. No computation.
- **AC-008**: Given the router configuration, when reviewed, then `main.go` must not import `chi` — that import belongs exclusively in the adapter layer.

## 6. Test Automation Strategy

- **Test Level**: Unit only in this phase. No integration tests yet (no external dependencies exist).
- **Framework**: Standard Go testing (`testing` package). No third-party assertion libraries.
- **Pattern**: Table-driven tests (`[]struct{ ... }`) for the health handler.
- **What to test**: Handler behavior — status code, response body, Content-Type. Test the handler function directly using `net/http/httptest`.
- **What NOT to test**: That `main.go` starts a server. That the router registers routes. Those are integration-level concerns and belong to a later phase.
- **TDD enforcement**: The test file must exist and fail before the handler is written. Do not write `health.go` before `health_test.go`.
- **Coverage**: All acceptance criteria in AC-005 must have a corresponding test case.

```
// Minimal shape of what a table-driven handler test looks like in Go.
// This is the pattern — do not copy it mechanically. Understand it.
func TestHealthHandler(t *testing.T) {
    tests := []struct {
        name           string
        expectedStatus int
        expectedBody   string
    }{
        // your cases here
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // use httptest.NewRecorder() and httptest.NewRequest()
        })
    }
}
```

## 7. Rationale & Context

**Why Chi over Gin?**

Chi composes with `net/http` directly. Handlers are `http.HandlerFunc` — no framework type wrapping. This matters because:
- Handlers in this project will eventually call use cases. Use cases receive `context.Context` from stdlib. With Chi, `r.Context()` is the real context — no extraction from a framework wrapper.
- The domain and application layers will never import Chi. With Gin's `*gin.Context`, that boundary is easier to violate accidentally.
- Chi is a router, not a framework. It does not prescribe how you structure the application.

**Why is `main.go` a composition root only?**

`main.go` is the only place in the entire application that knows about every layer. That makes it the highest-coupling file by design. If it contains logic, that logic is untestable (can't import `main`). Its only job is to wire things together and start the process.

**Why explicit `http.Server` with timeouts?**

`http.ListenAndServe` with no timeout configuration is a known security and reliability issue (Slowloris attacks, resource exhaustion). Setting timeouts explicitly is a production baseline, not premature optimization.

**Why not create `domain/` and `application/` directories now?**

Creating empty directories signals intent but adds noise and false structure. Directories appear when the code that belongs in them is written. Not before.

## 8. Dependencies & External Integrations

### Technology Platform Dependencies
- **PLT-001**: Go 1.26+ — already initialized in `go.mod`.
- **PLT-002**: Chi router — lightweight HTTP router, stdlib-compatible. Added via `go get github.com/go-chi/chi/v5`.

### Infrastructure Dependencies
- **INF-001**: Environment variable `PORT` — integer string, used to configure the HTTP server listen address. No external system required.

No database. No external services. No authentication. No message queues.

## 9. Examples & Edge Cases

### Edge case: What happens when PORT is not a valid integer?

The spec does not require complex validation here. If `PORT` is invalid, the server will fail to start. That failure must be explicit — not silent. A log message or a returned error from `main` is acceptable.

### Edge case: What happens if the port is already in use?

`ListenAndServe` will return an error. That error must be handled — not ignored with `_`.

### What a minimal `main.go` looks like structurally

```go
// This illustrates structure, not implementation.
// main.go should read like a table of contents for the application.
func main() {
    // 1. Read configuration
    // 2. Build the router (via the adapter layer)
    // 3. Configure the server
    // 4. Start the server
    // 5. Handle the error
}
```

Notice: no imports from `chi`, no handler registration, no business logic. If your `main.go` is longer than ~20 lines, question whether it is doing too much.

## 10. Validation Criteria

The implementation is considered complete when:

- [ ] `go build ./...` succeeds with no errors
- [ ] `go test -race ./...` passes with no failures
- [ ] `go vet ./...` reports no issues
- [ ] The server starts and `GET /health` returns 200 with `{"status":"ok"}`
- [ ] Port is configurable via `PORT` environment variable
- [ ] `main.go` contains no Chi imports and no handler logic
- [ ] Tests were written before the handler (TDD — you must be able to demonstrate this in the git history)
- [ ] The developer can explain: why Chi, why the folder structure chosen, what `httptest` does, and why timeouts matter

## 11. Related Specifications / Further Reading

- `.docs/ARCHITECTURE.md` — Hexagonal Architecture structure and dependency rules
- `.docs/WORKFLOW.md` — TDD cycle and Definition of Done
- [Chi documentation](https://github.com/go-chi/chi) — router API reference
- [net/http/httptest](https://pkg.go.dev/net/http/httptest) — testing HTTP handlers without a real server
