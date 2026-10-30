# Perfiles de agentes

La definición acordada contempla 8 perfiles de agentes. DEV se divide en DevFrontend y DevBackend para separar responsabilidades y reducir el contexto que necesita cada perfil. Esta definición queda documentada aquí; su implementación en el backend está pendiente.

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

Los perfiles definen responsabilidades; el modelo de IA se configura por separado para cada agente.

## Reglas actuales de desarrollo

- Frontend: React, Vite y TypeScript; pruebas unitarias con Vitest.
- Backend: Go; pruebas unitarias con `testing` y `go test`.
- Cada perfil trabaja con las especificaciones y el contexto de su lado. TL separa el trabajo de frontend y backend en tareas relacionadas y coordina el contrato entre ambos.
- QAP y QA conservan la planificación y verificación conjunta cuando el caso de uso involucra frontend y backend. Las pruebas unitarias de cada desarrollador forman parte de su entrega.
