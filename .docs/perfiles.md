# Perfiles de agentes

La definición acordada contempla 9 perfiles de agentes. DEV se divide en DevFrontend y DevBackend para separar responsabilidades y reducir el contexto que necesita cada perfil. Las transiciones por perfil y la asignación ya están en el backend; ver `backend_architecture.md`.

| Perfil | Nombre | Qué hace |
|---|---|---|
| **PP** | Project Planning | Interactúa con el usuario para definir proyectos su alcance y requerimientos. Crea la documentación de los requerimientos y planeación de etapas y tareas: el qué. Planea todas las tareas de forma atómica desde el inicio, en todas las etapas. No prepara herramientas de prueba y no deja instrucciones en comentarios de PMT: lo que haya que decir se dice al momento con avisos de herdr. |
| **PC** | Project Coordinator | Resuelve las dudas del usuario sobre lo que el equipo hace en este momento, leyendo las pestañas de los agentes en herdr. Solo lee: no usa PMT, no avisa a nadie y no afecta el proyecto. Existe para que el usuario consulte sin cargar el contexto de PP. |
| **TL** | Tech Lead | Dirige técnicamente la etapa de su encargo, una por vez, y solo cambia el status de esa etapa: organiza tareas, asigna agentes técnicos y libera tareas para desarrollo. Decide sobre propuestas de cambio, aprueba o rechaza planes QAP y puede descartar hallazgos con justificación. Puede detener y reanudar su etapa, y cancelar o reabrir sus tareas, con motivo. No rechaza bugs: si un bug llegó al TL es real y se corrige. Realiza la verificación final y registra la integración para cerrar tareas como `done`. |
| **DevFrontend** | Developer Frontend | Lee las especificaciones de frontend e implementa la interfaz con las tecnologías definidas en la planeación. Escribe y ejecuta sus pruebas unitarias con la herramienta de pruebas definida, ejecuta el linter al terminar, si la tecnología tiene uno, y corrige lo que marque, registra sus resultados y entrega el trabajo. No escribe scripts propios de navegador: lo que se comprueba en pantalla lo verifica QA. Propone cambios técnicos y corrige los hallazgos de CR o QA correspondientes al frontend. |
| **DevBackend** | Developer Backend | Lee las especificaciones de backend e implementa la lógica con las tecnologías definidas en la planeación. Escribe y ejecuta sus pruebas unitarias con la herramienta de pruebas definida, ejecuta el linter al terminar, si la tecnología tiene uno, y corrige lo que marque, registra sus resultados y entrega el trabajo. Propone cambios técnicos y corrige los hallazgos de CR o QA correspondientes al backend. |
| **CR** | Code Reviewer | Revisa la lógica, integridad y calidad del código, incluyendo prácticas como SOLID y DRY. Registra hallazgos, solicita correcciones, verifica su resolución y aprueba la revisión de código. No ejecuta pruebas: revisa leyendo el código, y probar le toca a QA. Solo revisa el código del producto, no los scripts de evidencia de Dev, y en cada hallazgo anota la corrección exacta para que Dev la aplique. |
| **QAP** | Quality Assurance Planning | Diseña y versiona planes de pruebas cuando una tarea crea una funcionalidad nueva o cambia una existente, y afecta frontend y backend. Los casos de error van desde los más absurdos, como un número en un nombre, hasta los comunes, como guardar sin llenar campos obligatorios, y cubren también los rechazos y fallas del backend. Vincula tareas, define casos de prueba, incluye los criterios que el plan de desarrollo le asigna a QA y envía el plan al TL para su aprobación. |
| **DP** | Development Planning | Define el cómo de un grupo de tareas o de un bug antes de que Dev empiece: PP define el qué y DP lo convierte en un plan de desarrollo que Dev sigue. Lee el código existente antes de planear. Cada criterio de aceptación dice a qué tarea pertenece y quién lo comprueba: Dev con pruebas unitarias, build y lint, y QA lo de pantalla, la API real si la tarea la consume y los datos. No agrega exigencias que la tarea no pide y es breve. No escribe código, no cambia status y no aprueba nada. |
| **QA** | Quality Assurance | Verifica el funcionamiento mediante pruebas y evidencias. Ejecuta planes aprobados, comprueba persistencia y realiza exploración y regresión cuando corresponde. Registra hallazgos, solicita correcciones y aprueba QA; después queda pendiente el cierre del TL. |

El usuario humano tiene el perfil `User`: actúa sin token como `saul`, puede todo lo de PP, y no aprueba por CR, QA ni TL.

## Flujo por perfil

| Paso | Perfil | Comando | Efecto |
| --- | --- | --- | --- |
| 1 | PP | `create_*`, `import_project`, `assign_stage_lead` | Planea y pone un TL a cada etapa |
| 2 | PP | `release_project` | Proyecto y etapas a `stand_by` |
| 3 | TL | `add_stage_member`, `start_stage` | Etapa a `in_dev`, tareas a `ready`; con la primera etapa, proyecto a `in_dev` |
| 4 | DP | `create_dev_plan` | Registra el plan de desarrollo del grupo de tareas |
| 5 | Dev | `list_dev_plans`, `change_tasks_status` a `in_dev` | Lee el plan y toma tareas de su equipo, que quedan a su nombre |
| 6 | Dev | a `in_review` | Entrega, con evidencia de los criterios de aceptación que le tocan |
| 7 | CR | a `in_qa`, `changes_requested` o, con motivo, `pending_close` | Revisa el código sin ejecutarlo, también contra el plan. Aprueba, pide corrección o cierra su parte en un cambio interno |
| 8 | Dev | de `changes_requested` a `in_dev` | Corrige y vuelve a entregar |
| 9 | QA | a `pending_close`, con evidencia en el motivo, o `changes_requested` | Aprueba o pide corrección |
| 10 | TL | a `done`, o con motivo a `in_review` o `in_qa` | Cierra o devuelve |
| 11 | TL | `change_stage_status` a `done` | Cierra la etapa; con la última de la versión 1, el proyecto pasa solo a `working`. Cancelar la última abierta, potestad de PP o `User`, o retirarla, tiene el mismo efecto si otra quedó en `done` |

**Hallazgos:** CR y QA los registran con `create_finding`, indicando si son bloqueantes, y los cierran como `fixed` con `close_finding`. El TL de la etapa, o cualquier TL si es un bug o si la tarea se retiró o se eliminó, los descarta como `dismissed`. Ambos cierres exigen motivo. Todos los perfiles los consultan con `list_findings`, por tarea, por bug o por etapa. No frenan transiciones: la decisión de devolver es de CR o QA.

**Planes QAP:** QAP crea el plan con `create_test_plan`, lo versiona con `revise_test_plan` y lo envía con `submit_test_plan`. El TL de la etapa lo aprueba con `approve_test_plan` o lo devuelve con correcciones con `return_test_plan`; un plan devuelto exige una versión nueva antes de reenviarse. QA y los demás perfiles los consultan con `list_test_plans` y `list_test_plan_versions`. No frenan transiciones, y el script de Playwright y los resultados no se guardan en PMT.

**Planes de desarrollo:** el flujo es TL → DP → Dev. DP registra el plan con `create_dev_plan`, de una etapa con sus tareas o de un bug, y lo versiona con `revise_dev_plan`, con motivo. Cada versión define `scope` (archivos a crear o tocar y lo que queda fuera), `technical_decisions` (estructura, contrato y bibliotecas), `screen_states` (carga, vacío, error y móvil), `edge_cases` (valores extremos, caracteres especiales y datos corruptos), `error_handling` (qué falla con mensaje, nunca en silencio) y `acceptance_criteria` (cada criterio con la tarea a la que pertenece, quién lo comprueba y cómo: Dev con pruebas unitarias, build y lint; QA en pantalla, contra la API real solo si la tarea la consume, y en los datos guardados). Todos son obligatorios: si uno no aplica, se escribe el motivo. No tiene status ni aprobación: el TL solo comprueba que quedó registrado, y vale la versión más reciente. No frena transiciones. Todos los perfiles los consultan con `list_dev_plans`, por etapa, por tarea o por bug, y `list_dev_plan_versions`.

- **Dev** sigue el plan y entrega evidencia de los criterios que le tocan. Si encuentra que el plan está mal, se lo dice al TL y no lo cambia por su cuenta.
- **El TL** levanta a DP para una versión nueva si el problema es técnico. PP resuelve lo que cambie alcance o requisitos.
- **CR** revisa el código también contra el plan, leyéndolo: no ejecuta pruebas, builds ni servidores.
- **Reglas del plan:** a Dev nunca le toca comprobar nada en navegador; "contra la API real" solo si la tarea consume la API; el plan no agrega exigencias que la tarea no pide, ni garantías de infraestructura como apagado por señales, recuperación ante fallos o verificación del destino, salvo que la tarea las pida; y es breve, con lo necesario para cada tarea.
- **Effort:** si el grupo toca pagos, permisos, migraciones de datos o contratos entre etapas, el TL relanza a DP en `high`; en lo demás usa `medium`.

**Camino que termina en CR:** un cambio interno va de `in_review` a `pending_close` sin pasar por QA, y lo cierra el TL.

**Criterio de QA:** aplica cuando el cambio altera algo observable desde fuera: pantalla, respuesta MCP o API, resultado de un worker o datos guardados. No aplica si es interno y el comportamiento sigue igual: refactor, limpieza o renombre.

**Cómo prueba QA:** siempre lee en el plan de desarrollo (`list_dev_plans`) los criterios que le tocan y los comprueba. Además prueba según lo que afecta el cambio. En frontend, solo con Playwright, sin herramientas propias de navegador: ejecuta las pruebas de regresión y, con plan de QAP aprobado, prueba sus casos a mano con Playwright (CLI o MCP, headless); si Playwright no está instalado, avisa al TL. En MCP, solo el o los comandos afectados. En un worker, agrega la información necesaria para que se ejecute y comprueba lo que hizo. En backend, depende de lo que afecta: frontend, MCP o worker, y prueba como en ese caso. Siempre revisa en la base, con el cliente de la base del proyecto y con todos los accesos, que la información se guardó correctamente y sin incongruencias, sin instalar nada ni escribir scripts temporales. Una falla bloqueante se devuelve de inmediato; las no bloqueantes, juntas al terminar. El script de Playwright se guarda en la carpeta de pruebas de regresión que dejó la configuración del proyecto.

**Comunicación entre agentes:** se comunican con avisos de herdr, fuera de PMT. Los comandos de PMT solo afectan la base de datos; PMT no envía avisos ni notificaciones. Un aviso no se guarda: lo que deba quedar registrado va en PMT.

**Sesiones:** una sesión de agente dura un solo encargo. PMT guarda el estado, así que cada agente arranca limpio y lee en PMT lo que necesita.

| Perfil | Una sesión por |
| --- | --- |
| DevFrontend y DevBackend | Cada grupo de tareas, hasta que todas estén en `done` |
| DP | Cada grupo de tareas o cada bug: termina al registrar el plan y el TL la reinicia |
| CR | Cada revisión |
| QAP | Cada plan de pruebas |
| QA | Cada ronda de pruebas |
| TL | Cada etapa |
| PP | Cada planeación o cada bug que registra |
| PC | Cada consulta |

- **Correcciones:** si un trabajo regresa, Dev lo corrige en su misma sesión, con el contexto cargado. DP, CR, QAP y QA siempre usan sesión nueva.
- **Coordinación:** el TL coordina su etapa. DP, Dev, CR, QAP y QA le avisan al terminar. Él revisa en PMT que el trabajo quedó registrado, reinicia esa sesión y levanta al siguiente. A Dev solo lo reinicia al cerrar su grupo. El usuario levanta a PP y a PC; el usuario o PP levantan al TL, con una sesión nueva por etapa.
- **Encargo:** indica proyecto, etapa e ID de las tareas.
- **Sesión limpia:** en Claude y mcode se reinicia con `/new`, sin cerrar el programa; en Codex se relanza el proceso, porque tras `/new` descarta el primer mensaje. Después se vuelve a nombrar al agente en herdr, que lo pierde.
- **Levantar Claude:** se usa `claude-400`, comando propio del usuario definido en su `~/.bashrc`, que ejecuta `claude --dangerously-skip-permissions --autocompact 400k` y acepta los argumentos de `claude`. Por ser una función de shell, solo existe en un shell interactivo.

**Herramientas de PMT:** un agente carga el esquema de una herramienta antes de llamarla por primera vez, y copia los ID de la salida de PMT, sin escribirlos de memoria.

## Coordinación paso a paso

Completa la tabla del flujo con quién actúa y cuándo.

1. **Planear:** PP escribe los tipos del contrato también en las tareas que lo consumen.
2. **Liberar:** PP ejecuta `release_project` con el visto bueno del usuario. Se trabaja una etapa por vez: el usuario o PP levantan al TL con la etapa que sigue.
3. **Iniciar la etapa:** el TL inicia solo la etapa de su encargo y solo cambia el status de esa etapa. Si depende de otra, antes fija el contrato en su descripción con `revise_stage`.
   - **Configuración del proyecto:** la tarea que configura las tecnologías e instala dependencias y herramientas de prueba la toma el dev de ese lado. Si solo dejó configuración y el cascarón funcionando, sin lógica, termina en CR, que valida los archivos.
4. **Planear el desarrollo:** el TL agrupa tareas relacionadas y levanta a DP con proyecto, etapa e ID de las tareas, o con el bug. DP lee el código, registra el plan y avisa al TL, que comprueba en PMT que quedó registrado y reinicia a DP.
5. **Tomar tareas:** el TL levanta a cada dev con su grupo de tareas. Dev sigue el plan, entrega evidencia de los criterios que le tocan y avisa al TL, que levanta a CR para revisar el código; el encargo nunca le pide ejecutar pruebas. Un dev no recibe otro grupo hasta cerrar el actual.
6. **Decidir el camino tras CR:** con la tarea en `in_qa`, si la tarea crea una funcionalidad nueva o cambia una existente, y afecta frontend y backend, el TL levanta a QAP; si no, a QA.
7. **Aprobar en QA:** QA escribe en el motivo un resumen de lo verificado y la ruta del script o resultado. No hay adjuntos. PMT exige ese motivo, en tareas y bugs: sin él rechaza la aprobación. Si la evidencia difiere entre tareas, QA las aprueba una por una.
8. **Cerrar la tarea:** el TL revisa si quedan tareas en `ready`. Si cerró el grupo, reinicia al dev y lo levanta con el siguiente.
9. **Cerrar la etapa:** QA limpia sus datos de prueba al terminar cada ronda, y el TL lo comprueba antes de cerrar. Al cerrarla lo anota en `para-usuario.md` y termina su sesión; el usuario o PP levantan una nueva para la siguiente etapa.

**En todo momento:**

- **Registro:** el TL registra con `add_comment` la decisión del paso 6 y las demás que no sean una transición. Todos los perfiles los consultan con `list_comments`. No se copian los avisos.
- **Avance:** se revisa siempre con `get_project_progress`, nunca solo proyecto y etapas. Sus `signals` son hechos que apuntan a un atasco: `ready_without_work`, `stage_not_closed` y `stage_not_started`.
- **Atasco:** el vigía da aviso al TL cuando todos los agentes llevan un rato quietos. El TL revisa el avance y levanta a quien le toque, o lo anota en `para-usuario.md` si nadie puede seguir.
- **Detener:** el usuario puede detener una etapa o el proyecto en cualquier paso; el TL, solo su etapa.
- **Limpieza:** quien levanta un proceso lo baja.
- **Herramientas de prueba:** nadie crea las suyas; se usan las que se instalaron al configurar el proyecto.

## Flujo de bugs

Un bug solo se registra con el proyecto en `working`. Nadie espera otra orden: PP y cada perfil avisan al TL, y él levanta al siguiente con sesión nueva.

| Paso | Quién | Qué hace | El TL levanta a |
| --- | --- | --- | --- |
| 1 | Usuario con PP | Registra el bug con `create_bug`, en `reported` | QA |
| 2 | QA | Lo reproduce. Real: `confirmed`. Si no: `rejected` con motivo | Hace el paso 3 si es real; si se rechaza, avisa a PP |
| 3 | TL | Decide a qué dev le toca y lo deja en un comentario con `add_comment` | DP |
| 4 | DP | Registra el plan del bug con `create_dev_plan` y avisa al TL | Ese dev |
| 5 | Dev | Sigue el plan: `in_dev`, corrige, `in_review`, con evidencia de los criterios que le tocan | CR |
| 6 | CR | Revisa el código sin ejecutarlo, también contra el plan. Falla: `changes_requested`. Interno: `pending_close`. Si no: `in_qa` | Dev o QA; si es interno, hace el paso 8 |
| 7 | QA | Prueba y verifica los datos. Bien: `pending_close`. Si no: `changes_requested` | Dev; si está bien, hace el paso 8 |
| 8 | TL | Cierra en `done`, o regresa a CR o QA | CR o QA si lo regresa |

- **Crear y corregir:** solo PP y `User` ejecutan `create_bug` y `update_bug`. Otro perfil que detecte un fallo avisa al TL, que es enteramente responsable de su etapa: lo resuelve dentro de ella y solo si por fuerza mayor va más allá de la etapa se lo pasa a PP. Un bug en `done` o `rejected` ya no se corrige.
- **Sin dueño:** un bug no tiene dev ni TL asignado. El TL indica al dev en el comentario, y PMT no restringe qué dev o TL mueve el bug.
- **Bugs con frontend:** van directo a QA, sin plan de QAP. QA los prueba con Playwright y deja el script de regresión.
- **Avance:** `get_project_progress` trae los bugs abiertos por status en `open_bugs`.

Los perfiles definen responsabilidades; el modelo de IA se configura por separado para cada agente.

## Configuración recomendada

| Perfil | Herramienta | Modelo | Effort |
| --- | --- | --- | --- |
| PP, DP, TL y CR | `claude` | Opus | `medium` |
| PC, DevFrontend, DevBackend, QAP y QA | `mcode` | MiniMax M3.1 Flash | `high` |

MiniMax va en `high`: en una comparación con el mismo plan redujo los bloqueantes y el tiempo con unos 17 % más de tokens. El effort de DP puede subir a `high` por grupo, como indica Planes de desarrollo.

## Kit de agentes

Los perfiles y scripts que coordinan a los agentes están en `agentes/`, con la configuración de cada proyecto en su `.agentes.env`. Ver `agentes/README.md`.

## Reglas actuales de desarrollo

- Frontend: las tecnologías y la herramienta de pruebas unitarias se definen en la planeación.
- Backend: las tecnologías y la herramienta de pruebas unitarias se definen en la planeación.
- Cada perfil trabaja con las especificaciones y el contexto de su lado. TL separa el trabajo de frontend y backend en tareas relacionadas y coordina el contrato entre ambos.
- QAP y QA conservan la planificación y verificación conjunta cuando el caso de uso involucra frontend y backend. Las pruebas unitarias de cada desarrollador forman parte de su entrega.
