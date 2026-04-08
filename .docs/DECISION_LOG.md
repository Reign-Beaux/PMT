# Decision Log — PMT

Registro de decisiones técnicas tomadas o diferidas, con su contexto y razonamiento.

---

## [PENDING] Unit of Work

**Estado:** Diferido — implementar en una fase próxima post-MVP

**Contexto:**
El MVP actual maneja cada aggregate de forma independiente. Cada use case toca un solo aggregate, por lo que no hay necesidad de coordinación transaccional entre repos.

**Por qué se implementará:**
A medida que el dominio crezca, aparecerán operaciones que requieren atomicidad entre aggregates (ej. crear una Phase y actualizar el estado del Project en la misma transacción).

**Diseño acordado:**
- No será el punto central de acceso a repos (diferencia clave con la implementación .NET/EF)
- Será un coordinador quirúrgico — solo lo usan los use cases que necesitan transacciones multi-aggregate
- Los use cases simples siguen inyectando repos directamente
- Implementación: `internal/adapter/driven/postgres/uow.go`
- Interface (driven port): definida en el use case que la consuma

**Forma esperada:**
```go
type UnitOfWork struct { db *gorm.DB }

func (u *UnitOfWork) Execute(ctx context.Context, fn func(repos Repositories) error) error {
    return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        return fn(Repositories{
            Projects: NewProjectRepository(tx),
            Phases:   NewPhaseRepository(tx),
            Issues:   NewIssueRepository(tx),
        })
    })
}
```

**Comportamiento de transacciones:**

GORM maneja commit/rollback automáticamente dentro de `db.Transaction()`:

```
Execute() llamado
    │
    ├── fn() retorna nil   → COMMIT automático  ✓
    │
    └── fn() retorna error → ROLLBACK automático ✓
                             el error sube al use case → handler → HTTP response
```

Para casos donde se necesite control explícito de la transacción (lógica compleja mid-transaction), se puede exponer:

```go
func (u *UnitOfWork) Begin(ctx context.Context) (*gorm.DB, error) { ... }
func (u *UnitOfWork) Commit(tx *gorm.DB) error                    { ... }
func (u *UnitOfWork) Rollback(tx *gorm.DB)                        { ... }
```

Para el 90% de los casos, el rollback automático de `Execute()` es suficiente.

**Diferencia clave con .NET/EF:**

| .NET / EF Core | Go / GORM |
|---|---|
| UoW es punto central de acceso a todos los repos | UoW es coordinador quirúrgico — solo para multi-aggregate |
| Todos los use cases pasan por UoW | Use cases simples siguen inyectando repos directamente |
| Change tracker de EF hace el UoW natural | Cada repo call ejecuta SQL inmediatamente — no hay change tracker |
| `SaveChangesAsync()` es el flush central | `db.Transaction()` agrupa operaciones explícitamente |

**Referencia de implementación .NET:**
Ver `/home/saul/Proyectos/BusinessProjects/FMS(Financial Management System)/Backend/src/Infrastructure/Persistence/UnitOfWork.cs`

---
