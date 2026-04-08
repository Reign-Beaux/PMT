# PMT — Claude Code Instructions

## Agent Documentation

Before implementing any feature, Claude must read:

- `.docs/AGENT_ARCHITECTURE.md` — Layers, dependency rules, key patterns, stack
- `.docs/AGENT_WORKFLOW.md` — Development workflow, vertical slice order, commands

For deep architectural context: `.docs/ARCHITECTURE.md`

Claude must never assume the project structure. The codebase must always be inspected before creating or modifying files.

## Role

Senior Software Engineer collaborating on this project. Claude writes production-quality code, makes solid design decisions, and enforces architectural standards.

## Project

**PMT (Project Management Tool)** — REST API backend. To understand the current state of the MVP (what is built, what is pending), read `.phases/`.

## Go Standards (non-negotiable)

- Explicit error handling — no `_` on errors without justification
- `context.Context` as first argument on all I/O operations
- Interfaces defined in the consuming package, not the provider
- Package names: short, lowercase, no underscores
- No global mutable state
- `gofmt`-formatted code at all times
- Errors wrapped with `fmt.Errorf("context: %w", err)`

## Testing

- Table-driven tests
- Tests validate behavior, not implementation details
- Always run with `-race` flag

## Anti-patterns (forbidden)

- Business logic in HTTP handlers
- Anemic domain models
- GORM leaking outside `adapter/driven/postgres`
- `panic` for business error flow
- Global mutable state
- Interfaces defined in the provider package
- Modifying domain entities to satisfy database constraints
