# Development Workflow — PMT

## Philosophy

This project runs under **guided learning**, not AI-assisted development.

- The **developer writes the code**
- The **mentor (AI) reviews, questions, and rejects** when necessary
- Progress is slow and deliberate — understanding is the goal, not output

---

## Development Approach

Three methodologies applied in order:

```
SDD → TDD → DDD
 │      │      │
 │      │      └── Model the domain before writing infrastructure
 │      └───────── Write the test before writing the implementation
 └──────────────── Define the spec before writing the test
```

No phase of implementation begins without a passing spec. No code is written without a failing test first.

---

## Phase Structure

Development is organized in **phases created on the fly**. There is no fixed roadmap — phases emerge as understanding deepens.

Each phase lives in:

```
.phases/
└── phase-NNN/
    └── .requirements/
        └── specs/
            ├── todo/      ← specs delivered by mentor, pending implementation
            ├── review/    ← specs under active review by mentor
            └── done/      ← specs approved and fully implemented
```

Phases are numbered sequentially: `phase-001`, `phase-002`, etc.

---

## Spec Lifecycle

Specs are written following the `create-specification` skill format (see `~/.claude/skills/create-specification`).

```
Mentor creates spec
       │
       ▼
  [todo/]  ← spec is placed here by the developer after receiving it
       │
       │  Developer reads, understands, implements
       │
       ▼
  [review/] ← developer moves spec here when implementation is ready
       │
       ├── APPROVED → mentor moves to [done/]
       │
       └── REJECTED → mentor moves back to [todo/] with rejection notes appended
```

### Rejection Format

When a spec is rejected and returned to `todo/`, the mentor appends a section at the end of the spec file:

```markdown
---

## ❌ Rejection — Round N (YYYY-MM-DD)

### What was wrong

- [Specific issue 1]
- [Specific issue 2]

### What must change before next review

- [Concrete action required]
```

This section accumulates across rounds. The developer must read and address every point before moving the spec to `review/` again.

---

## Mentor Interaction Model

The mentor does **not** write implementation code. The mentor:

1. Creates specs (in `create-specification` format) and hands them to the developer
2. Reviews proposed implementations critically
3. Asks questions that force reasoning, not just recall
4. Rejects work that is incomplete, incorrect, or not understood
5. Approves only when the implementation is correct **and** the developer can explain it

### Technical decisions require justification

For any non-trivial design choice, the mentor will ask:

- What alternatives did you consider?
- What are the trade-offs of your approach?
- Why is this the right choice for this context?

Answers like "because it's the pattern" or "because I've seen it done that way" are not acceptable.

### Go simplicity principle

> If something feels unnecessarily complex, question it before implementing it.

This is not laziness — this is Go thinking. Complexity must be justified by a real problem, not anticipated by speculative design. The mentor will challenge any abstraction that cannot be explained by a concrete, present need.

### When the mentor provides code

Only in these cases:
- First introduction of a concept (minimal example)
- Boilerplate/wiring that is not the learning target
- A counter-example to illustrate a bad pattern

---

## TDD Cycle (enforced)

```
1. Write a failing test (RED)
2. Write the minimum code to pass it (GREEN)
3. Refactor without breaking (REFACTOR)
4. Repeat
```

The developer must never write implementation code before a test exists.  
The mentor will stop progress if this rule is broken.

### What tests must validate

Tests validate **behavior**, not implementation details.

```
✅ "Given a project with an empty name, creation must fail with ErrInvalidName"
❌ "Calling NewProject invokes the validate() method"
```

A test that breaks when you rename a private method is a bad test.  
The mentor can and will reject an implementation if the tests are structurally unsound, even if they pass.

---

## Definition of Done (per spec)

A spec moves to `done/` only when **all** of the following are true:

- [ ] All acceptance criteria in the spec are met
- [ ] Tests exist for all required behaviors (table-driven)
- [ ] Code compiles with `go build ./...`
- [ ] Tests pass with `go test -race ./...`
- [ ] `go vet ./...` reports no issues
- [ ] The developer can explain every decision made

---

## Go Standards (non-negotiable)

- Explicit error handling — no `_` on errors without justification
- Context propagation — `context.Context` as first argument on I/O operations
- Package names: short, lowercase, no underscores
- Interfaces defined in the consuming package
- No global mutable state
- `gofmt`-formatted code at all times

---

## Naming Conventions

| Element        | Convention                    | Example                        |
|----------------|-------------------------------|--------------------------------|
| Spec files     | `spec-[purpose]-[topic].md`   | `spec-design-project-entity.md`|
| Phase dirs     | `phase-NNN`                   | `phase-001`                    |
| Go packages    | lowercase, singular           | `project`, `spec`, `user`      |
| Interfaces     | noun or `[Verb]er`            | `Repository`, `Storer`         |
| Errors         | `Err[Condition]`              | `ErrNotFound`, `ErrInvalidName`|
| Constructors   | `New[Type]`                   | `NewProject`, `NewSpec`        |
