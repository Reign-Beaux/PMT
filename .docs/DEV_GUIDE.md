# Developer Guide — PMT

Guía práctica para desarrollar en este proyecto. Cubre cómo agregar nuevas funcionalidades siguiendo la arquitectura hexagonal, desde el dominio hasta el endpoint HTTP.

---

## Índice

1. [Setup del proyecto](#1-setup-del-proyecto)
2. [Estructura de paquetes](#2-estructura-de-paquetes)
3. [Agregar un nuevo aggregate (CRUD completo)](#3-agregar-un-nuevo-aggregate-crud-completo)
   - [Paso 1 — Dominio](#paso-1--dominio)
   - [Paso 2 — Migración SQL](#paso-2--migración-sql)
   - [Paso 3 — Repositorio PostgreSQL](#paso-3--repositorio-postgresql)
   - [Paso 4 — Servicio de aplicación](#paso-4--servicio-de-aplicación)
   - [Paso 5 — Handler HTTP](#paso-5--handler-http)
   - [Paso 6 — Router](#paso-6--router)
   - [Paso 7 — Wiring en main.go](#paso-7--wiring-en-maingo)
4. [Cuándo usar Unit of Work](#4-cuándo-usar-unit-of-work)
5. [Migraciones](#5-migraciones)
6. [Testing](#6-testing)
7. [Comandos de referencia](#7-comandos-de-referencia)

---

## 1. Setup del proyecto

**Requisitos:** Go, Docker

```bash
# 1. Copiar variables de entorno
cp .env.example .env
# Editar .env con tus credenciales de PostgreSQL

# 2. Levantar la base de datos (si usas Docker)
docker compose up -d

# 3. Correr el servidor (crea la DB y corre migraciones automáticamente)
go run ./cmd/api
```

La base de datos `pmt` se crea sola si no existe. Las migraciones corren automáticamente al arrancar.

---

## 2. Estructura de paquetes

```
cmd/api/
  main.go                          ← solo wiring: conecta repositorios, servicios, handlers

internal/
  domain/{aggregate}/              ← reglas de negocio puras (sin imports externos)
    {aggregate}.go                 ← entidad con constructor y comportamiento
    value_objects.go               ← Value Objects validados
    errors.go                      ← errores de dominio (ErrNotFound, ErrInvalidX)
    {aggregate}_test.go            ← tests de comportamiento del dominio

  domain/shared/
    id.go                          ← ID (UUID), NewID(), ParseID()

  application/{aggregate}/
    service.go                     ← use cases (Create, GetByID, List, Update, Delete)
    repository.go                  ← interface del repositorio (driven port)

  application/uow/
    repositories.go                ← tipo Repositories para callback del UnitOfWork
                                     (NO es un aggregate — es infraestructura de la capa application)

  adapter/driven/postgres/
    {aggregate}_repo.go            ← implementación GORM del repositorio
    uow.go                         ← implementación concreta del UnitOfWork
    migrations/
      000NNN_{descripcion}.up.sql  ← migración forward
      000NNN_{descripcion}.down.sql← migración rollback

  adapter/driving/httpserver/
    router.go                      ← registro de todas las rutas
    handler/
      {aggregate}.go               ← handler HTTP + interface del servicio (driving port)
      respond.go                   ← helpers writeJSON / writeError
```

**Regla de dependencias:**
```
handler → application → domain
postgres_repo ← application ← domain
```
- `domain` no importa nada externo
- `application` importa solo `domain`
- `adapter` importa `application` y `domain`, nunca entre sí

---

## 3. Agregar un nuevo aggregate (CRUD completo)

El ejemplo usa un aggregate hipotético `Tag`. Sigue los pasos en orden.

---

### Paso 1 — Dominio

Crea el paquete `internal/domain/tag/`.

**`errors.go`** — errores de dominio:
```go
package tag

import "errors"

var (
    ErrNotFound    = errors.New("tag not found")
    ErrInvalidName = errors.New("tag name cannot be empty")
)
```

**`value_objects.go`** — Value Objects validados en construcción:
```go
package tag

import "strings"

type Name struct {
    value string
}

func NewName(s string) (Name, error) {
    n := Name{value: strings.TrimSpace(s)}
    if !n.isValid() {
        return Name{}, ErrInvalidName
    }
    return n, nil
}

func (n Name) String() string { return n.value }
func (n Name) isValid() bool  { return n.value != "" }
```

**`tag.go`** — entidad con constructor y comportamiento:
```go
package tag

import (
    "time"
    "project-management-tools/internal/domain/shared"
)

type Tag struct {
    id        shared.ID
    name      Name
    createdAt time.Time
    updatedAt time.Time
}

// New crea un Tag nuevo validando las invariantes del dominio.
func New(name Name) (Tag, error) {
    if !name.isValid() {
        return Tag{}, ErrInvalidName
    }
    now := time.Now()
    return Tag{
        id:        shared.NewID(),
        name:      name,
        createdAt: now,
        updatedAt: now,
    }, nil
}

// Reconstitute reconstruye un Tag desde datos persistidos.
// No pasa por el constructor — confía en la integridad de la DB.
func Reconstitute(id shared.ID, name Name, createdAt, updatedAt time.Time) Tag {
    return Tag{id: id, name: name, createdAt: createdAt, updatedAt: updatedAt}
}

func (t Tag) ID() shared.ID        { return t.id }
func (t Tag) Name() Name           { return t.name }
func (t Tag) CreatedAt() time.Time { return t.createdAt }
func (t Tag) UpdatedAt() time.Time { return t.updatedAt }

func (t *Tag) Rename(name Name) error {
    if !name.isValid() {
        return ErrInvalidName
    }
    t.name = name
    t.updatedAt = time.Now()
    return nil
}
```

**`tag_test.go`** — tests de comportamiento (table-driven):
```go
package tag_test

import (
    "errors"
    "testing"
    "project-management-tools/internal/domain/tag"
)

func TestNewName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr error
    }{
        {name: "valid", input: "backend", wantErr: nil},
        {name: "empty", input: "", wantErr: tag.ErrInvalidName},
        {name: "spaces only", input: "   ", wantErr: tag.ErrInvalidName},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := tag.NewName(tt.input)
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("got %v, want %v", err, tt.wantErr)
            }
        })
    }
}
```

---

### Paso 2 — Migración SQL

Crea los archivos en `internal/adapter/driven/postgres/migrations/`.

El número debe ser el siguiente en la secuencia. Revisa los existentes para saber cuál sigue.

**`000005_create_tags.up.sql`:**
```sql
CREATE TABLE tags (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

**`000005_create_tags.down.sql`:**
```sql
DROP TABLE IF EXISTS tags;
```

La migración corre automáticamente la próxima vez que arranque el servidor.

> **Nunca modifiques una migración ya ejecutada.** Si necesitas cambiar el schema, crea una nueva migración.

---

### Paso 3 — Repositorio PostgreSQL

Crea `internal/adapter/driven/postgres/tag_repo.go`.

```go
package postgres

import (
    "context"
    "errors"
    "time"

    "gorm.io/gorm"
    "project-management-tools/internal/domain/tag"
    "project-management-tools/internal/domain/shared"
)

// tagModel es el modelo GORM. Nunca sale de este paquete.
type tagModel struct {
    ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
    Name      string    `gorm:"not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (tagModel) TableName() string { return "tags" }

type TagRepository struct {
    db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
    return &TagRepository{db: db}
}

func (r *TagRepository) Save(ctx context.Context, t tag.Tag) error {
    model := toTagModel(t)
    return r.db.WithContext(ctx).Create(&model).Error
}

func (r *TagRepository) FindByID(ctx context.Context, id shared.ID) (tag.Tag, error) {
    var model tagModel
    err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return tag.Tag{}, tag.ErrNotFound
    }
    if err != nil {
        return tag.Tag{}, err
    }
    return toTagDomain(model)
}

func (r *TagRepository) FindAll(ctx context.Context) ([]tag.Tag, error) {
    var models []tagModel
    if err := r.db.WithContext(ctx).Find(&models).Error; err != nil {
        return nil, err
    }
    tags := make([]tag.Tag, 0, len(models))
    for _, m := range models {
        t, err := toTagDomain(m)
        if err != nil {
            return nil, err
        }
        tags = append(tags, t)
    }
    return tags, nil
}

func (r *TagRepository) Update(ctx context.Context, t tag.Tag) error {
    model := toTagModel(t)
    return r.db.WithContext(ctx).Save(&model).Error
}

func (r *TagRepository) Delete(ctx context.Context, id shared.ID) error {
    return r.db.WithContext(ctx).Delete(&tagModel{}, "id = ?", id.String()).Error
}

// toTagModel convierte dominio → GORM. Solo existe aquí.
func toTagModel(t tag.Tag) tagModel {
    return tagModel{
        ID:        t.ID().String(),
        Name:      t.Name().String(),
        CreatedAt: t.CreatedAt(),
        UpdatedAt: t.UpdatedAt(),
    }
}

// toTagDomain convierte GORM → dominio usando Reconstitute.
func toTagDomain(m tagModel) (tag.Tag, error) {
    id, err := shared.ParseID(m.ID)
    if err != nil {
        return tag.Tag{}, err
    }
    name, err := tag.NewName(m.Name)
    if err != nil {
        return tag.Tag{}, err
    }
    return tag.Reconstitute(id, name, m.CreatedAt, m.UpdatedAt), nil
}
```

---

### Paso 4 — Servicio de aplicación

Crea `internal/application/tag/repository.go` y `service.go`.

**`repository.go`** — driven port (interface definida en el consumidor):
```go
package tag

import (
    "context"
    "project-management-tools/internal/domain/tag"
    "project-management-tools/internal/domain/shared"
)

type Repository interface {
    Save(ctx context.Context, t tag.Tag) error
    FindByID(ctx context.Context, id shared.ID) (tag.Tag, error)
    FindAll(ctx context.Context) ([]tag.Tag, error)
    Update(ctx context.Context, t tag.Tag) error
    Delete(ctx context.Context, id shared.ID) error
}
```

**`service.go`** — use cases:
```go
package tag

import (
    "context"
    "project-management-tools/internal/domain/tag"
    "project-management-tools/internal/domain/shared"
)

type CreateInput struct{ Name string }
type UpdateInput struct{ Name *string }

type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (tag.Tag, error) {
    name, err := tag.NewName(input.Name)
    if err != nil {
        return tag.Tag{}, err
    }
    t, err := tag.New(name)
    if err != nil {
        return tag.Tag{}, err
    }
    if err := s.repo.Save(ctx, t); err != nil {
        return tag.Tag{}, err
    }
    return t, nil
}

func (s *Service) GetByID(ctx context.Context, id shared.ID) (tag.Tag, error) {
    return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]tag.Tag, error) {
    return s.repo.FindAll(ctx)
}

func (s *Service) Update(ctx context.Context, id shared.ID, input UpdateInput) (tag.Tag, error) {
    t, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return tag.Tag{}, err
    }
    if input.Name != nil {
        name, err := tag.NewName(*input.Name)
        if err != nil {
            return tag.Tag{}, err
        }
        if err := t.Rename(name); err != nil {
            return tag.Tag{}, err
        }
    }
    if err := s.repo.Update(ctx, t); err != nil {
        return tag.Tag{}, err
    }
    return t, nil
}

func (s *Service) Delete(ctx context.Context, id shared.ID) error {
    if _, err := s.repo.FindByID(ctx, id); err != nil {
        return err
    }
    return s.repo.Delete(ctx, id)
}
```

---

### Paso 5 — Handler HTTP

Crea `internal/adapter/driving/httpserver/handler/tag.go`.

```go
package handler

import (
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    tagapp "project-management-tools/internal/application/tag"
    "project-management-tools/internal/domain/tag"
    "project-management-tools/internal/domain/shared"
)

// TagService es el driving port — definido aquí, en el consumidor.
type TagService interface {
    Create(ctx context.Context, input tagapp.CreateInput) (tag.Tag, error)
    GetByID(ctx context.Context, id shared.ID) (tag.Tag, error)
    List(ctx context.Context) ([]tag.Tag, error)
    Update(ctx context.Context, id shared.ID, input tagapp.UpdateInput) (tag.Tag, error)
    Delete(ctx context.Context, id shared.ID) error
}

type TagHandler struct{ svc TagService }

func NewTagHandler(svc TagService) *TagHandler { return &TagHandler{svc: svc} }

type tagResponse struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func toTagResponse(t tag.Tag) tagResponse {
    return tagResponse{
        ID:        t.ID().String(),
        Name:      t.Name().String(),
        CreatedAt: t.CreatedAt(),
        UpdatedAt: t.UpdatedAt(),
    }
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Name string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    t, err := h.svc.Create(r.Context(), tagapp.CreateInput{Name: body.Name})
    if err != nil {
        writeTagError(w, err)
        return
    }
    writeJSON(w, http.StatusCreated, toTagResponse(t))
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
    tags, err := h.svc.List(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, "failed to list tags")
        return
    }
    resp := make([]tagResponse, 0, len(tags))
    for _, t := range tags {
        resp = append(resp, toTagResponse(t))
    }
    writeJSON(w, http.StatusOK, resp)
}

func (h *TagHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    id, err := shared.ParseID(chi.URLParam(r, "id"))
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid tag id")
        return
    }
    t, err := h.svc.GetByID(r.Context(), id)
    if err != nil {
        writeTagError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, toTagResponse(t))
}

func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
    id, err := shared.ParseID(chi.URLParam(r, "id"))
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid tag id")
        return
    }
    var body struct {
        Name *string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    t, err := h.svc.Update(r.Context(), id, tagapp.UpdateInput{Name: body.Name})
    if err != nil {
        writeTagError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, toTagResponse(t))
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id, err := shared.ParseID(chi.URLParam(r, "id"))
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid tag id")
        return
    }
    if err := h.svc.Delete(r.Context(), id); err != nil {
        writeTagError(w, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func writeTagError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, tag.ErrNotFound):
        writeError(w, http.StatusNotFound, err.Error())
    case errors.Is(err, tag.ErrInvalidName):
        writeError(w, http.StatusBadRequest, err.Error())
    default:
        writeError(w, http.StatusInternalServerError, "internal server error")
    }
}
```

---

### Paso 6 — Router

En `internal/adapter/driving/httpserver/router.go`, agrega el handler como parámetro y registra las rutas:

```go
func NewRouter(
    projectHandler *handler.ProjectHandler,
    phaseHandler   *handler.PhaseHandler,
    issueHandler   *handler.IssueHandler,
    tagHandler     *handler.TagHandler,   // ← agregar
) http.Handler {
    // ...
    r.Route("/tags", func(r chi.Router) {
        r.Post("/", tagHandler.Create)
        r.Get("/", tagHandler.List)
        r.Get("/{id}", tagHandler.GetByID)
        r.Patch("/{id}", tagHandler.Update)
        r.Delete("/{id}", tagHandler.Delete)
    })
}
```

---

### Paso 7 — Wiring en main.go

```go
// Repositories
tagRepo := pgadapter.NewTagRepository(db)

// Services
tagService := tagapp.NewService(tagRepo)

// Handlers
tagHandler := handler.NewTagHandler(tagService)

router := httpserver.NewRouter(projectHandler, phaseHandler, issueHandler, tagHandler)
```

---

## 4. Cuándo usar Unit of Work

La mayoría de use cases **no** usan el UoW — inyectan su repositorio directamente. El UoW solo entra cuando un use case necesita atomicidad entre múltiples aggregates.

**Regla práctica:**

| El use case toca… | Inyecta |
|---|---|
| Un solo aggregate | Su `Repository` directamente |
| Múltiples aggregates en la misma operación | `unitOfWork` |

**Cómo integrarlo cuando sea necesario:**

1. Declara la interfaz en el use case que la consume (no de forma global):

```go
// internal/application/somefeature/service.go
type unitOfWork interface {
    Execute(ctx context.Context, fn func(uow.Repositories) error) error
}
```

2. Inyéctalo en el servicio junto a los repositorios que necesites:

```go
type Service struct {
    uow  unitOfWork
    repo Repository // si también necesita acceso fuera de transacción
}
```

3. Úsalo solo en el método que requiere la transacción:

```go
func (s *Service) DeleteWithCascade(ctx context.Context, id shared.ID) error {
    return s.uow.Execute(ctx, func(repos uow.Repositories) error {
        if err := repos.Phases.DeleteByProject(ctx, id); err != nil {
            return err
        }
        return repos.Projects.Delete(ctx, id)
    })
}
```

4. En `main.go`, instancia el UoW y pásalo al servicio:

```go
uow := pgadapter.NewUnitOfWork(db)
someService := somefeatureapp.NewService(uow, projectRepo)
```

`*postgres.UnitOfWork` satisface la interfaz implícitamente — no hay registro ni cast explícito.

---

## 5. Migraciones


Las migraciones son archivos SQL versionados en `internal/adapter/driven/postgres/migrations/`.

```
000001_initial.up.sql         ← activa extensión uuid-ossp
000002_create_projects.up.sql
000003_create_phases.up.sql
000004_create_issues.up.sql
000005_...up.sql              ← próxima
```

**Correr migraciones:** se ejecutan automáticamente al arrancar el servidor.

**Crear nueva migración:**
1. Crea `000NNN_descripcion.up.sql` — cambios forward
2. Crea `000NNN_descripcion.down.sql` — rollback
3. Arranca el servidor — se aplica sola

**Reglas:**
- Nunca modifiques una migración ya ejecutada — crea una nueva
- El `.down.sql` debe revertir exactamente lo que hace el `.up.sql`
- Usa `uuid_generate_v4()` como default para columnas UUID
- FKs siempre con `ON DELETE CASCADE` salvo decisión explícita contraria

---

## 6. Testing

Los tests del dominio viven junto al código que prueban: `internal/domain/{aggregate}/{aggregate}_test.go`.

```bash
# Correr todos los tests
go test -race ./...

# Correr tests de un paquete específico
go test -race ./internal/domain/project/...

# Ver cobertura
go test -cover ./...
```

**Reglas:**
- Table-driven tests siempre
- Paquete `_test` externo (`package project_test`) — prueba la API pública
- No testear detalles de implementación (métodos privados, orden de llamadas)
- Usar `errors.Is` para validar errores de dominio

---

## 7. Comandos de referencia

```bash
# Levantar servidor
go run ./cmd/api

# Compilar
go build ./...

# Tests con race detector
go test -race ./...

# Análisis estático
go vet ./...

# Ver tablas en la DB
docker exec postgresql psql -U yubel -d pmt -c "\dt"

# Consultar datos
docker exec postgresql psql -U yubel -d pmt -c "SELECT * FROM projects;"
docker exec postgresql psql -U yubel -d pmt -c "SELECT * FROM phases ORDER BY sort_order;"
docker exec postgresql psql -U yubel -d pmt -c "SELECT * FROM issues;"

# Entrar a psql interactivo
docker exec -it postgresql psql -U yubel -d pmt

# Ver migraciones aplicadas
docker exec postgresql psql -U yubel -d pmt -c "SELECT * FROM schema_migrations;"

# Limpiar dependencias
go mod tidy
```
