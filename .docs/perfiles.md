# Perfiles de agentes

La definición acordada contempla 8 perfiles de agentes. DEV se divide en DevFrontend y DevBackend para separar responsabilidades y reducir el contexto que necesita cada perfil. Las transiciones por perfil y la asignación ya están en el backend; ver `backend_architecture.md`.

| Perfil | Nombre | Qué hace |
|---|---|---|
| **PP** | Project Planning | Interactúa con el usuario para definir proyectos su alcance y requerimientos. Crea la documentación de los requerimientos y planeación de etapas y tareas. |
| **PC** | Project Coordinator | Interactúa con el usuario para consultar avances, da seguimiento y registra comunicaciones, comentarios y bloqueos/desbloqueos. |
| **TL** | Tech Lead | Dirige técnicamente la etapa: organiza tareas, asigna agentes técnicos y libera tareas para desarrollo. Decide sobre propuestas de cambio, aprueba o rechaza planes QAP y puede descartar hallazgos con justificación. Realiza la verificación final y registra la integración para cerrar tareas como `done`. |
| **DevFrontend** | Developer Frontend | Lee las especificaciones de frontend e implementa la interfaz con React, Vite y TypeScript. Escribe y ejecuta sus pruebas unitarias con Vitest, registra sus resultados y entrega el trabajo. Propone cambios técnicos y corrige los hallazgos de CR o QA correspondientes al frontend. |
| **DevBackend** | Developer Backend | Lee las especificaciones de backend e implementa la lógica con Go. Escribe y ejecuta sus pruebas unitarias con el paquete estándar `testing` mediante `go test`, registra sus resultados y entrega el trabajo. Propone cambios técnicos y corrige los hallazgos de CR o QA correspondientes al backend. |
| **CR** | Code Reviewer | Revisa la lógica, integridad y calidad del código, incluyendo prácticas como SOLID y DRY. Registra hallazgos, solicita correcciones, verifica su resolución y aprueba la revisión de código. |
| **QAP** | Quality Assurance Planning | Diseña y versiona planes de pruebas para casos que combinan cambios de frontend y backend. Vincula tareas, define casos de prueba y envía el plan al TL para su aprobación. |
| **QA** | Quality Assurance | Verifica el funcionamiento mediante pruebas y evidencias. Ejecuta planes aprobados, comprueba persistencia y realiza exploración y regresión cuando corresponde. Registra hallazgos, solicita correcciones y aprueba QA; después queda pendiente el cierre del TL. |

El usuario humano tiene el perfil `User`: actúa sin token como `saul`, puede todo lo de PP y PC, y no aprueba por CR, QA ni TL. PC solo detiene y reanuda etapas y proyectos.

## Flujo por perfil

| Paso | Perfil | Comando | Efecto |
| --- | --- | --- | --- |
| 1 | PP | `create_*`, `import_project`, `assign_stage_lead` | Planea y pone un TL a cada etapa |
| 2 | PP | `release_project` | Proyecto y etapas a `stand_by` |
| 3 | TL | `add_stage_member`, `start_stage` | Etapa a `in_dev`, tareas a `ready`; con la primera etapa, proyecto a `in_dev` |
| 4 | Dev | `change_tasks_status` a `in_dev` | Toma tareas de su equipo, que quedan a su nombre |
| 5 | Dev | a `in_review` | Entrega |
| 6 | CR | a `in_qa`, `changes_requested` o, con motivo, `pending_close` | Aprueba, pide corrección o cierra su parte en un cambio interno |
| 7 | Dev | de `changes_requested` a `in_dev` | Corrige y vuelve a entregar |
| 8 | QA | a `pending_close` o `changes_requested` | Aprueba o pide corrección |
| 9 | TL | a `done`, o con motivo a `in_review` o `in_qa` | Cierra o devuelve |
| 10 | TL | `change_stage_status` a `done` | Cierra la etapa; con la última de la versión 1, el proyecto pasa solo a `working`. Cancelar la última abierta, potestad de PP o `User`, tiene el mismo efecto si otra quedó en `done` |

**Camino que termina en CR:** un cambio interno va de `in_review` a `pending_close` sin pasar por QA, y lo cierra el TL.

**Criterio de QA:** aplica cuando el cambio altera algo observable desde fuera: pantalla, respuesta MCP o API, resultado de un worker o datos guardados. No aplica si es interno y el comportamiento sigue igual: refactor, limpieza o renombre.

**Cómo prueba QA:** con plan, Playwright headless; en MCP, ejecuta el comando; en un worker, registro falso funcional. Siempre verifica la integridad en la base de datos. Una falla bloqueante se devuelve de inmediato; las no bloqueantes, juntas al terminar. El script de Playwright se guarda en el repositorio del proyecto probado.

**Comunicación entre agentes:** se avisan con hooks de herdr, fuera de PMT. Los comandos de PMT solo afectan la base de datos; PMT no envía avisos ni notificaciones.

Los perfiles definen responsabilidades; el modelo de IA se configura por separado para cada agente.

## Reglas actuales de desarrollo

- Frontend: React, Vite y TypeScript; pruebas unitarias con Vitest.
- Backend: Go; pruebas unitarias con `testing` y `go test`.
- Cada perfil trabaja con las especificaciones y el contexto de su lado. TL separa el trabajo de frontend y backend en tareas relacionadas y coordina el contrato entre ambos.
- QAP y QA conservan la planificación y verificación conjunta cuando el caso de uso involucra frontend y backend. Las pruebas unitarias de cada desarrollador forman parte de su entrega.
