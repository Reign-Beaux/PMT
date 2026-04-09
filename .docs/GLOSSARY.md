# Glosario — PMT

Referencia de los conceptos del dominio usados en este proyecto.

---

## Project (Proyecto)

El contenedor raíz de todo el trabajo. Un proyecto agrupa fases y tiene un backlog propio. Representa una iniciativa, producto o área de trabajo.

**Ejemplo:** "Rediseño del sitio web", "Backend v2", "App móvil".

---

## Phase (Fase)

Una subdivisión ordenada dentro de un proyecto. Puede representar un sprint, una etapa de desarrollo, un milestone o cualquier bloque de trabajo con un inicio y un fin. Las phases tienen un orden explícito dentro del proyecto.

**Ejemplo:** "Sprint 1", "Alpha", "Q2 Entregas", "Bug Bash".

---

## Issue

La unidad de trabajo. Un issue representa una tarea concreta que alguien tiene que hacer. Siempre pertenece a un proyecto y opcionalmente está asignado a una phase.

Campos principales:
- **Title** — Descripción corta de la tarea.
- **Spec** — Descripción larga, contexto, criterios de aceptación.
- **Priority** — `low`, `medium`, `high`.
- **Status** — Estado actual del issue (ver Status más abajo).

**Ejemplo:** "Corregir error de login en móvil", "Implementar endpoint de autenticación".

---

## Backlog

El conjunto de issues de un proyecto que aún no han sido asignados a ninguna phase. Es la lista de trabajo pendiente sin planificar. Un issue en backlog tiene `phase_id = null`.

En Jira se llama igual. En Monday equivale a un grupo sin sprint.

**Flujo típico:**  
`Backlog → se asigna a una Phase → se trabaja → done/closed`

> En PMT, el backlog es implícito: no es una phase especial, sino la ausencia de phase.

---

## Status

El estado en el que se encuentra un issue. Los cambios de estado siguen un flujo de transiciones válidas — no se puede saltar de cualquier estado a cualquier otro.

| Status        | Significado                                           |
|---------------|-------------------------------------------------------|
| `open`        | Creado, pendiente de iniciar                          |
| `in_progress` | Alguien está trabajando en él activamente             |
| `done`        | El trabajo está terminado, pendiente de revisión/cierre |
| `closed`      | Terminado y cerrado definitivamente                   |

**Transiciones válidas:**

```
open        → in_progress, closed
in_progress → done, open
done        → closed
closed      → (terminal, no hay salida)
```

---

## Transition (Transición)

El acto de mover un issue de un status a otro. Solo se permiten transiciones válidas según el flujo definido. Intentar una transición inválida devuelve error.

**Endpoint:** `PATCH /projects/{projectId}/phases/{phaseId}/issues/{id}/transition`

---

## Priority (Prioridad)

Indica la urgencia relativa de un issue. Tres niveles:

| Valor    | Cuándo usarlo                                |
|----------|----------------------------------------------|
| `low`    | Puede esperar, no hay presión de tiempo      |
| `medium` | Default. Importante pero no urgente          |
| `high`   | Bloquea algo o tiene fecha límite próxima    |

---

## Spec (Especificación)

Campo de texto libre dentro de un issue. Sirve para escribir el contexto del problema, los pasos para reproducirlo, los criterios de aceptación, o cualquier detalle relevante. No tiene estructura impuesta — es el "cuerpo" del issue.

En Jira equivale al campo **Description**. En Monday al **Updates** o **Details**.

---

## Jerarquía completa

```
Project
├── Backlog
│   ├── Issue (sin phase)
│   └── Issue (sin phase)
└── Phase "Sprint 1"
    ├── Issue
    └── Issue
        └── (status, priority, spec, transitions...)
```
