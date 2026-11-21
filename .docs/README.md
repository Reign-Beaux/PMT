# PMT

PMT es la herramienta con la que un equipo de agentes de IA desarrolla proyectos de software. Guarda el estado de cada proyecto (etapas, tareas, bugs, planes, hallazgos y comentarios) y aplica las reglas de quién puede hacer qué. Los agentes trabajan en pestañas de herdr, consultan y registran ese estado por MCP, y se avisan entre sí con herdr. El usuario sigue el avance en la interfaz web.

Esta carpeta describe lo que PMT **es hoy**, al 21 de noviembre de 2026. La documentación anterior, escrita mientras se diseñaba, está en `historico/`.

## Estado actual

- **En uso.** `pmt-server` corre en `http://127.0.0.1:8080` y sirve en un solo proceso la API REST, el MCP y la interfaz web.
- **Probado con proyectos reales.** El equipo de agentes desarrolló con PMT:
  - **DBP** (`/home/saul/Proyectos/tools/dbp`), editor de modelos de base de datos, hoy en producción. En PMT está en `working`, versión 1.17.0.
  - **El frontend de PMT**, que PMT se gestionó a sí mismo. En PMT está en `working`, versión 1.3.0, con 7 etapas en `done` y la etapa "Pruebas E2E del frontend nuevo" en `stand_by`.
- **BrisaFMS** tiene el kit preparado en su repositorio y sus agentes `fms-*` creados, pero aún no tiene proyecto en PMT.

## Partes del repositorio

| Carpeta | Qué es | Documento |
| --- | --- | --- |
| `backend/` | Servidor en Go: API REST, MCP, eventos en tiempo real y base PostgreSQL | [arquitectura.md](arquitectura.md) |
| `frontend/` | Interfaz web de consulta en React | [frontend.md](frontend.md) |
| `agentes/` | Kit que levanta y coordina al equipo de agentes en herdr | `agentes/README.md` |
| `.docs/` | Esta documentación y la deuda técnica | |

## Documentos

| Documento | Contenido |
| --- | --- |
| [dominio.md](dominio.md) | Conceptos del negocio, status, transiciones y reglas que PMT aplica |
| [perfiles.md](perfiles.md) | Los nueve perfiles de agentes, el flujo de trabajo de una etapa y de un bug, y las versiones |
| [arquitectura.md](arquitectura.md) | Cómo está construido el backend, sus features y la lista de herramientas MCP |
| [frontend.md](frontend.md) | Pantallas, tiempo real y pruebas del frontend |
| [importar.md](importar.md) | Formato YAML de `import_project` e `import_stage` |
| [operacion.md](operacion.md) | Arrancar, compilar, migrar, administrar tokens, desplegar y respaldar |
| [deuda_tecnica.md](deuda_tecnica.md) | Lo pendiente conocido |

## Cómo se trabaja

```text
Usuario ── PP ── planea el proyecto en PMT (qué se hace)
            │
            └── TL de la etapa ── DP (cómo) ── Dev ── CR ── QAP/QA ── TL cierra
                                                          │
Interfaz web ◄── WebSocket ── pmt-server ◄── MCP ─────────┘
```

1. El usuario platica el proyecto con **PP**, que lo planea en etapas y tareas atómicas, o lo importa de un YAML.
2. Con el visto bueno del usuario, PP libera el proyecto y se levanta al **TL** de la primera etapa. Se trabaja una etapa por vez.
3. El TL coordina su etapa: **DP** escribe el plan de desarrollo, **DevBackend** o **DevFrontend** lo implementan, **CR** revisa el código leyéndolo, **QAP** diseña el plan de pruebas cuando aplica y **QA** prueba. Al final el TL cierra cada tarea y la etapa, y hace commit y push.
4. Cuando la versión 1 termina, el proyecto pasa a `working`. Desde ahí crece con etapas nuevas y bugs, y cada cambio sube la versión de producto.

El detalle está en [perfiles.md](perfiles.md).
