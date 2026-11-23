# Perfiles y flujo de trabajo

PMT trabaja con nueve perfiles de agentes y el usuario. Cada agente tiene su token, y su perfil decide qué puede hacer en PMT ([dominio.md](dominio.md#permisos)). Las instrucciones que recibe cada agente están en `backend/internal/features/profiles/catalog/perfiles/<perfil>.md` y `comun.md` de esa carpeta ([Dónde viven los perfiles](#dónde-viven-los-perfiles)). Este documento y esos archivos se editan juntos.

## Perfiles

| Perfil | Nombre | Qué hace |
| --- | --- | --- |
| **PP** | Project Planning | Platica con el usuario y planea el proyecto: el qué. Escribe etapas y tareas atómicas, cada una con un entregable verificable, y los tipos del contrato también en las tareas que lo consumen. Para importar escribe el YAML en el proyecto, como respaldo, y pasa su contenido a `import_project` o `import_stage`. Libera el proyecto, registra bugs, sube la versión y agrega lo que un TL le pasa fuera de su etapa. No deja instrucciones en comentarios de PMT. |
| **PC** | Project Coordinator | Responde las dudas del usuario sobre lo que hace el equipo, leyendo las pestañas de herdr. Solo lee: no usa PMT, no avisa a nadie y no afecta el proyecto. Existe para que el usuario consulte sin cargar el contexto de PP. |
| **TL** | Tech Lead | Dirige su etapa, una por vez, y es responsable de ella. La inicia y la cierra, arma el equipo, libera tareas, levanta a DP, Dev, CR, QAP y QA, aprueba o devuelve planes QAP, descarta hallazgos con motivo, cierra tareas y hace commit y push al cerrar. Puede detener y reanudar su etapa, y cancelar o reabrir sus tareas. Solo pasa a PP lo que va más allá de su etapa. |
| **DP** | Development Planning | Define el cómo antes de que Dev empiece: lee el código existente y registra el plan de desarrollo de un grupo de tareas o de un bug. Cada criterio de aceptación dice su tarea, quién lo comprueba y cómo. Es breve y no agrega exigencias que la tarea no pide. No escribe código. |
| **DevBackend** | Developer Backend | Implementa el backend siguiendo el plan, con sus pruebas unitarias y el linter. Entrega con evidencia de los criterios que le tocan y corrige lo que devuelvan CR o QA. |
| **DevFrontend** | Developer Frontend | Igual que DevBackend, en el frontend. No escribe scripts de navegador: lo de pantalla lo verifica QA. |
| **CR** | Code Reviewer | Revisa el código leyéndolo, también contra el plan: lógica, integridad, calidad, SOLID y DRY. No ejecuta pruebas, builds ni servidores. Registra hallazgos con la corrección exacta, aprueba, pide corrección o cierra un cambio interno. |
| **QAP** | Quality Assurance Planning | Diseña el plan de pruebas cuando una tarea crea o cambia una funcionalidad que afecta frontend y backend. Los casos de error van de los absurdos a los comunes e incluyen rechazos del backend. Lo envía al TL. |
| **QA** | Quality Assurance | Prueba los criterios que el plan de desarrollo le asigna y lo que afecta el cambio, revisa los datos en la base, registra hallazgos y aprueba con evidencia. Confirma o rechaza bugs. |

**El usuario** (`User`) es una persona con cuenta: entra al frontend con su correo y contraseña, o usa un token de API suyo, y actúa con su `username`. Tiene todo lo de PP, pero no aprueba como CR, QA ni TL.

## Flujo de una etapa

| Paso | Quién | Comando | Efecto |
| --- | --- | --- | --- |
| 1 | PP | `create_*` o `import_project`, `assign_stage_lead` | Planea y pone un TL a cada etapa |
| 2 | PP | `release_project` | Con el visto bueno del usuario: proyecto y etapas a `stand_by` |
| 3 | TL | `add_stage_member`, `start_stage` | Etapa a `in_dev` y tareas a `ready`; con la primera, proyecto a `in_dev` |
| 4 | DP | `create_dev_plan` | Plan de desarrollo del grupo de tareas |
| 5 | Dev | `change_tasks_status` a `in_dev` | Lee el plan y toma las tareas de su grupo |
| 6 | Dev | a `in_review` | Entrega con evidencia |
| 7 | CR | a `in_qa`, `changes_requested` o, si es interno, `pending_close` | Revisa leyendo el código |
| 8 | QAP | `create_test_plan`, `submit_test_plan` | Solo si la tarea es funcionalidad nueva o cambiada que afecta frontend y backend |
| 9 | QA | a `pending_close` con evidencia, o `changes_requested` | Prueba |
| 10 | TL | a `done`, o devuelve a `in_review` o `in_qa` | Cierra la tarea |
| 11 | TL | `change_stage_status` a `done` | Commit y push de lo que tocó la etapa, y cierra. Con la última etapa de la versión 1 el proyecto pasa solo a `working` |

### Reglas del flujo

- **Una etapa por vez.** El usuario o PP levantan al TL con la etapa que sigue. El TL solo inicia, detiene y cierra la suya.
- **Contrato entre etapas.** Si la etapa depende de otra, el TL fija el contrato en su descripción con `revise_stage` antes de iniciarla.
- **Configuración del proyecto.** La tarea que configura tecnologías, dependencias y herramientas de prueba la toma el dev de ese lado. En backend deja la configuración con un YAML por ambiente en `config/`: `config.dev.yaml` fuera de git con los valores reales, y `config.prod.yaml` versionado con `changeme` en los secretos y `# override: <VARIABLE>` en lo que producción sobrescribe. Si solo deja configuración, termina en CR.
- **Grupos.** El TL agrupa tareas relacionadas, levanta a DP con ellas y luego a cada dev con su grupo. Un dev no recibe otro grupo hasta cerrar el actual.
- **Después de CR.** Si la tarea crea o cambia una funcionalidad que afecta frontend y backend, el TL levanta a QAP; si no, directo a QA. Registra esa decisión con `add_comment`.
- **Cuándo pasa por QA.** Cuando el cambio altera algo observable: pantalla, respuesta de MCP o API, resultado de un worker o datos guardados. Un cambio interno (refactor, limpieza, renombre) va de CR a `pending_close`.
- **Evidencia de QA.** QA escribe en el motivo un resumen de lo verificado y la ruta del script o resultado; PMT rechaza la aprobación sin motivo. Si la evidencia difiere entre tareas, las aprueba una por una.
- **Cierre de etapa.** QA limpia sus datos de prueba y el TL lo comprueba. El TL hace commit y push solo de los archivos de su etapa, por ruta, con el nombre de la etapa en el mensaje; lo ajeno lo anota en `para-usuario.md`. Nadie más hace commit. Si la etapa empezó con el proyecto ya en `working`, avisa a PP para que suba la versión.

### Cómo prueba QA

- Siempre comprueba los criterios que el plan de desarrollo le asigna.
- **Frontend:** solo con Playwright. Ejecuta la regresión y, con plan QAP aprobado, prueba sus casos a mano con `playwright-cli`, el comando global. Si la tarea afecta el frontend, deja un script de regresión en la carpeta de pruebas del proyecto.
- **MCP:** solo los comandos afectados.
- **Worker:** le da lo necesario para ejecutarse y comprueba lo que hizo.
- **Backend:** prueba como en el caso que afecte; si no afecta nada visible, comprueba los criterios contra el servicio levantado.
- **Datos:** siempre revisa en la base que la información quedó bien, con el cliente de base del proyecto, sin instalar nada ni escribir scripts temporales.
- Una falla bloqueante se devuelve de inmediato; las demás, juntas al terminar.

### Planes de desarrollo

- El flujo es TL → DP → Dev. Dev sigue el plan; si lo encuentra mal se lo dice al TL, que relanza a DP si es técnico o pasa a PP si cambia el alcance.
- A Dev nunca le toca comprobar nada en navegador. "Contra la API real" solo si la tarea consume la API.
- El plan no agrega exigencias que la tarea no pide, como apagado por señales o recuperación ante fallos.
- **Effort de DP:** `high` si el grupo toca pagos, permisos, migraciones de datos o contratos entre etapas; `medium` en lo demás. Se cambia con `relanzar-dp.sh` del kit o `pmt relanzar-dp`.

## Flujo de un bug

Solo con el proyecto en `working`, por un fallo hallado en pruebas de rutina o en el uso del sistema.

| Paso | Quién | Qué hace | Luego el TL levanta a |
| --- | --- | --- | --- |
| 1 | Usuario con PP | `create_bug`, queda en `reported` | QA |
| 2 | QA | Lo reproduce: `confirmed`, o `rejected` con motivo | Sigue el TL si es real; si no, avisa a PP |
| 3 | TL | Decide qué dev lo corrige y lo deja en un comentario | DP |
| 4 | DP | `create_dev_plan` del bug | Ese dev |
| 5 | Dev | `in_dev`, corrige, `in_review` con evidencia | CR |
| 6 | CR | `in_qa`, `changes_requested` o, si es interno, `pending_close` | QA o Dev |
| 7 | QA | Prueba y revisa los datos: `pending_close` o `changes_requested` | Dev si falla |
| 8 | TL | Commit y push de los archivos del bug, `done`, y avisa a PP para subir la versión | |

Los bugs con frontend van directo a QA, sin plan QAP, y QA deja el script de regresión.

## Fallos durante el desarrollo

Un fallo hallado mientras se desarrolla no es un bug. Quien lo detecta avisa al TL:

- **Dentro de su etapa:** el TL lo resuelve ahí, por ejemplo devolviendo la tarea con un hallazgo.
- **Fuera de su etapa:** el TL se lo pasa a PP, que lo agrega con `add_task` en la etapa a la que pertenece, o con `revise_task` si esa tarea aún no está en `done`.

## Versiones

`product_version` tiene la forma `X.Y.Z`. PP la fija al planear; en `working` la sube con `change_product_version`, con motivo.

- **Mayor (X):** cambia la app entera o algo deja de ser compatible, como la API o el MCP. Solo con el visto bueno del usuario.
- **Menor (Y):** funcionalidad nueva o ajuste que no rompe nada; normalmente, una etapa nueva al cerrarse.
- **Parche (Z):** un bug cerrado.

Al subir un dígito, los de su derecha vuelven a 0. Las etapas de la versión 1 no suben la versión.

## Coordinación entre agentes

- **Avisos.** Los agentes se comunican con avisos de herdr (`avisar.sh` del kit o `pmt avisar`), fuera de PMT. Lo que deba quedar registrado va en PMT; los avisos no se copian.
- **El TL coordina.** DP, Dev, CR, QAP y QA le avisan al terminar. Él comprueba en PMT que el trabajo quedó registrado, reinicia esa sesión y levanta al siguiente.
- **Encargo.** Indica proyecto, etapa e ID de las tareas, o el bug.
- **Avance.** Se revisa con `get_project_progress`. Sus `signals` apuntan a atascos: `ready_without_work`, `stage_not_closed` y `stage_not_started`.
- **Atasco.** El vigía avisa al TL cuando todos los agentes llevan un rato quietos. Si nadie puede seguir, el TL lo anota en `para-usuario.md`.
- **Detener.** El usuario puede detener una etapa o el proyecto en cualquier momento; el TL, solo su etapa.
- **Limpieza.** Quien levanta un proceso lo baja antes de avisar. El aviso es lo último del turno.
- **Herramientas de prueba.** Nadie crea las suyas: se usan las que se instalaron al configurar el proyecto.
- **Etapa tomada.** Si una escritura responde `conflict: la etapa la está trabajando <usuario>` (o `el bug lo está trabajando`), la etapa o el bug son de otra persona: el agente no reintenta ni busca otro camino. Avisa al TL, que lo anota en `para-usuario.md` para que el usuario hable con esa persona o con el owner del proyecto, el único que la libera (`release_stage`, `release_bug`).
- **IDs.** Un agente carga el esquema de una herramienta de PMT antes de usarla por primera vez y copia los ID de la salida, sin escribirlos de memoria.

## Sesiones

Una sesión dura un solo encargo: PMT guarda el estado y cada agente arranca limpio.

| Perfil | Una sesión por |
| --- | --- |
| DevBackend y DevFrontend | Cada grupo de tareas, hasta que todas estén en `done`. Corrige lo devuelto en la misma sesión |
| DP | Cada grupo de tareas o cada bug |
| CR | Cada revisión |
| QAP | Cada plan |
| QA | Cada ronda de pruebas |
| TL | Cada etapa |
| PP | Cada planeación o cada bug |
| PC | Cada consulta |

## Dónde viven los perfiles

Una sola fuente, en el backend: `backend/internal/features/profiles/catalog/`, incrustada en el binario con `embed`.

| Archivo | Qué es |
| --- | --- |
| `perfiles/<agente>.md` | El rol de cada perfil (`pp`, `pc`, `tl`, `dp`, `devback`, `devfront`, `cr`, `qap`, `qa`) |
| `comun.md` | Lo común, que se agrega después del rol en todos menos PC |
| `pc-comun.md` | Lo que se agrega a PC en lugar de `comun.md`: solo la ruta del proyecto y la tabla de agentes, porque PC solo lee herdr |

**Dos versiones en un texto.** Los datos de la máquina (ruta, base, puertos, comandos del equipo) solo los conoce quien levanta al agente. Donde el texto los usa hay un tramo `[[local||servido]]`:

- el kit (`agentes/iniciar.sh`) toma la parte local y rellena sus `{{MARCADORES}}` con el `.agentes.env`, y genera exactamente los mismos perfiles que antes;
- PMT toma la parte servida, que remite a `.pmt/entorno.md` del proyecto.

Se eligió así, y no un segundo juego de sustituciones en Go, porque un marcador suelto no basta: la versión servida cambia también el texto alrededor (`` `{{KIT}}/avisar.sh <agente> "mensaje"` `` pasa a «el comando de avisar de `.pmt/entorno.md`», sin comillas invertidas ni ruta), y con sustituciones en Go cada frase viviría lejos de su regla. Con los dos lados juntos, quien edita una regla ve las dos versiones en el mismo archivo. `comun.md` y `pc-comun.md` abren con un tramo cuya parte local está vacía: la línea que pide leer `.pmt/entorno.md` solo sale en la versión servida.

### Cómo se sirven

El texto servido de un perfil es el rol más lo común, en el mismo orden que los junta el kit. Se lee de dos formas, que devuelven lo mismo:

- **`get_profile`** (MCP, sin argumentos): el perfil de la sesión. Cualquier perfil puede usarla. Sin perfil declarado responde `invalid` y dice cómo declararlo (`?profile=` en la URL del MCP).
- **`GET /api/profiles/{perfil}`** (REST, `text/markdown`). `GET /api/profiles` lista los perfiles ([arquitectura.md](arquitectura.md#perfiles-servidos)).

Claude Code solo muestra los primeros ~4 000 caracteres de las `instructions` de un servidor MCP, y ocho de los nueve perfiles servidos pasan de eso (entre 4 800 y 9 000). Por eso las instrucciones no llevan el perfil: una sesión MCP con perfil declarado por un usuario (`/mcp?profile=tl`) recibe en `initialize` solo unas líneas (menos de 400 caracteres). Para TL:

```text
Esta sesión de PMT tiene el perfil TL (Tech Lead).

Antes de cualquier otra cosa:
1. Llama a get_profile: devuelve tu perfil completo, con tus reglas y las comunes. Síguelo en toda la sesión.
2. Lee .pmt/entorno.md en la raíz del proyecto: trae los datos de esta máquina (ruta, base, puertos, comandos del equipo) que tu perfil menciona.
```

Las de los demás cambian solo el perfil y su descripción. Sin perfil (`User`) no hay instrucciones. Un **token de agente** (el kit) tampoco las recibe: ya trae su perfil local completo por archivo, y su proyecto todavía no tiene `.pmt/entorno.md`; si llama a `get_profile`, recibe el texto servido de su perfil fijo.

`pmt pp` y `pmt equipo` pasan el perfil completo al lanzar a cada agente, como el kit; las instrucciones y `get_profile` quedan de respaldo ([plan_produccion.md](plan_produccion.md)).

### Formato de `.pmt/entorno.md`

Archivo local del proyecto, en su raíz, fuera de git (lleva la cadena de la base: `.pmt/` va en `.gitignore`). Lo generan `pmt pp` y `pmt equipo` en cada arranque desde [`.pmt/config.yaml`](#formato-de-pmtconfigyaml), así que no se edita a mano. Markdown con estas secciones, en este orden, con los nombres que usan los perfiles servidos:

```markdown
# Entorno del proyecto

Lo generan `pmt pp` y `pmt equipo` en cada arranque desde `.pmt/config.yaml`: no lo edites a mano.

## Proyecto
- Ruta: `/ruta/absoluta/al/proyecto`
- Descripción: Tiene backend/ (Go) y frontend/ (React, Vite, TypeScript).
- Proyecto en PMT: Mi proyecto (`0b9c…`)

## Base de datos
- Cadena: `postgres://usuario:clave@localhost:5432/base?sslmode=disable` (solo DevBackend la escribe, en `config/config.dev.yaml`; los demás usan ese archivo)
- Consultar: `docker exec postgresql psql -U usuario -d base -c "<consulta>"` (cambia `<consulta>` por la tuya)

## Puertos
- Backend: 8090
- Frontend: 5173

## Comandos del equipo
- Avisar: `pmt avisar <agente> "mensaje"`
- Levantar: `pmt levantar <agente> "encargo"`
- Reiniciar: `pmt reiniciar <agente>`
- Relanzar a DP: `pmt relanzar-dp <medium|high>`
- Vigía: `pmt vigia`

## Para el usuario
- Archivo: `/ruta/absoluta/al/proyecto/.pmt/para-usuario.md`

## Agentes
| Agente | PMT | Perfil |
|---|---|---|
| pp | ana | PP |
| tl | ana | TL |
```

Cada sección cubre un marcador del kit: Proyecto (`PROYECTO_RUTA`, `PROYECTO_DESCRIPCION`), Base de datos (`DB_URL`, `DB_CONSULTA`), Puertos (`PUERTO_BACKEND`, `PUERTO_FRONTEND`), Comandos del equipo (`KIT`), Para el usuario (`AGENTES_DIR`) y Agentes (`TABLA_AGENTES`).

- **Proyecto en PMT:** nombre e ID del proyecto, o «todavía no existe; lo crea PP» si se eligió nuevo.
- **Lo que aún no se sabe** (descripción, cadena, consulta, puertos) dice `pendiente: lo completa \`pmt equipo\``. Solo pasa entre el primer `pmt pp` y el primer `pmt equipo`, que lo pregunta.
- **Comandos del equipo:** se listan siempre ([arquitectura.md](arquitectura.md#comandos-del-equipo)).
- **Agentes:** una fila por perfil configurado en `config.yaml`, con su nombre en herdr (con el prefijo, si hay), el `username` de quien lanzó (todos sus agentes actúan con su token) y el perfil. Los nombres en herdr son los del kit: `pp`, `pc`, `tl`, `dp`, `devback`, `devfront`, `cr`, `qap`, `qa`. Los comandos del equipo aceptan el nombre con o sin prefijo (`pmt avisar demo-tl` y `pmt avisar tl` son lo mismo).
- **Para el usuario:** `.pmt/para-usuario.md`, que `pmt pp` crea vacío si falta.

### Formato de `.pmt/config.yaml`

Lo que `pmt` sabe del proyecto en esta máquina. Fuera de git y **sin tokens**: el de PMT sale de las credenciales de `pmt login` y los de MCP extra de archivos en `~/.config/pmt/`. Lo crea `pmt pp` la primera vez (proyecto y PP) y lo completa `pmt equipo` (descripción, base, puertos y los otros ocho perfiles); cada uno lo vuelve a preguntar con `--configurar`. `pmt relanzar-dp` guarda ahí el effort de DP.

```yaml
version: 1
proyecto:
  id: 0b9c…                 # vacío mientras PP no lo crea
  nombre: Mi proyecto
  nuevo_desde: 2026-11-22T18:00:00Z   # solo si se eligió «nuevo, lo crea PP»
  ruta: ""                  # vacío: la carpeta donde está .pmt/ (no se pregunta)
  descripcion: ""           # pmt equipo; propone la descripción del proyecto en PMT
base:
  cadena: ""                # pmt equipo; no se muestra al volver a preguntar
  consulta: ""              # pmt equipo, con <consulta> donde va la consulta
puertos:
  backend: 0                # pmt equipo; propone 8090
  frontend: 0               # pmt equipo; propone 5173
herdr:
  workspace: mi-proyecto    # por omisión, el nombre de la carpeta; se crea si no existe
  prefijo: ""               # se pone solo si otro workspace ya tiene un agente «pp»
codex_home: ""              # CODEX_HOME de Codex; vacío: ~/.codex. Acepta ~/
mcp:                        # MCP extra (HTTP con Bearer), opcional
  dbp:
    url: https://dbp.ejemplo.com/mcp
    token: tokens/dbp-mcp   # archivo con el token, relativo a ~/.config/pmt/
perfiles:                   # por nombre de sesión: pp, pc, tl, dp, devbackend, devfrontend, cr, qap, qa
  pp:
    herramienta: claude     # claude o codex
    modelo: opus            # por omisión opus en Claude, gpt-6.1-terra en Codex
    effort: medium          # medium o high
    mcp: [dbp]              # MCP extra que carga este perfil, opcional
  tl:                       # pmt equipo propone claude, opus, medium para tl, dp y cr,
    herramienta: claude     # y codex, gpt-6.1-terra, high para pc, devbackend, devfrontend, qap y qa
    modelo: opus
    effort: medium
```

- **Raíz del proyecto:** la carpeta actual, o la más cercana hacia arriba que ya tenga `.pmt/`.
- **Proyecto nuevo:** con `nuevo_desde`, cada `pmt pp` busca proyectos de los que el usuario es owner creados desde esa hora; si hay uno pregunta si es ese, si hay varios pide elegir, y guarda `id` y `nombre`. Nunca impide el arranque.
- **MCP extra:** el nombre va en minúsculas y dígitos, distinto de `pmt`. El token se pasa al agente en `MCP_TOKEN_<NOMBRE>`, como en el kit.
- Además, `.pmt/` lleva `entorno.md`, `para-usuario.md`, `perfiles/<perfil>.md` (el perfil descargado en cada arranque), con Claude `mcp-<perfil>.json`, y del vigía `vigia.log` y `vigia.pid`.

## Herramientas y modelos en uso

Con el kit, cada agente se configura en el `.agentes.env` del proyecto; con `pmt`, en `.pmt/config.yaml`. Así trabajaron DBP y está configurado BrisaFMS:

| Perfil | Herramienta | Modelo | Effort |
| --- | --- | --- | --- |
| PP, DP, TL y CR | `claude` | Opus | `medium` |
| DevBackend y DevFrontend | `codex` | gpt-6.1-terra | `high` |
| PC, QAP y QA | `codex` | gpt-6.1-terra | `medium` |

El kit también admite `mcode` (MiniMax) y `qwen`; `pmt`, solo `claude` y `codex`, y propone `high` también para PC, QAP y QA. El frontend de PMT se hizo con Qwen en PC, Devs, QAP y QA. Cómo se levanta el equipo: con `pmt` en [operacion.md](operacion.md#equipo-de-agentes); con el kit, en `agentes/README.md`.
