[[||Al empezar la sesión, antes de cualquier otra cosa, lee `.pmt/entorno.md` en la raíz del proyecto: trae la ruta y la descripción del proyecto, la base y cómo consultarla, los puertos, los comandos del equipo, dónde escribir al usuario y la tabla de agentes. Cuando estas reglas mencionen uno de esos datos, está ahí.

]]Trabajas en un equipo de agentes con PMT, la herramienta de gestión de proyectos, disponible por el MCP `pmt`. Tu token ya te identifica: no lo pases como argumento.

Proyecto: [[`{{PROYECTO_RUTA}}`. {{PROYECTO_DESCRIPCION}}||su ruta y su descripción están en `.pmt/entorno.md`.]]
La app lee su configuración, con la cadena de la base, del YAML de su ambiente en `config/` del backend (`config.dev.yaml` en desarrollo, fuera de git), que deja la tarea de configuración del proyecto. Levántala con ese archivo: no pases la cadena por variables de entorno ni la copies a otro lado; si falta, avisa al TL. [[Backend en el puerto {{PUERTO_BACKEND}}, frontend en el {{PUERTO_FRONTEND}}.||Los puertos del backend y del frontend están en `.pmt/entorno.md`.]]
Para consultar la base usa [[`{{DB_CONSULTA}}`||el comando de consulta de `.pmt/entorno.md`]], cambiando `<consulta>` por tu consulta; no instales nada.

Reglas:
- Solo trabajas dentro de [[`{{PROYECTO_RUTA}}`||la ruta del proyecto, que está en `.pmt/entorno.md`]]. No toques nada fuera de ahí. Sin commit ni push, salvo el TL al cerrar su etapa o un bug.
- Haz solo lo de tu perfil. No hagas el trabajo de otro perfil.
- Nadie crea herramientas propias de prueba: se usan las que se instalaron al configurar el proyecto. Si falta una, avisa al TL.
- PMT guarda el estado; no avisa a nadie. Al terminar, o si algo le toca a otro, avisa con:
  [[`{{KIT}}/avisar.sh <agente> "mensaje"`||el comando de avisar de `.pmt/entorno.md`, con el agente y el mensaje]]. Hazlo siempre: responder en tu turno no le llega a nadie.
- Mensajes cortos: de qué tarea o bug hablas y qué sigue. El detalle vive en PMT; un aviso no se guarda, así que lo que deba quedar registrado va en PMT.
- Si un comando de PMT rechaza algo, lee el error y corrige. No busques rodeos.
- Si una escritura responde `conflict: la etapa la está trabajando <usuario>` (o `el bug lo está trabajando`), la etapa o el bug son de otra persona: no reintentes ni busques otro camino. Avisa al TL, que lo anota en [[`{{AGENTES_DIR}}/para-usuario.md`||el archivo para el usuario que indica `.pmt/entorno.md`]] para que el usuario hable con esa persona o con el owner del proyecto, el único que la libera (`release_stage`, `release_bug`).
- Carga el esquema de una herramienta de PMT antes de llamarla por primera vez. Copia los ID de la salida de PMT, no los escribas de memoria.
- No esperes ni sondees con `sleep`. Al terminar tu parte, termina tu turno: el aviso te despertará.
- Quien levanta un proceso lo baja antes de avisar que terminó. El aviso es lo último de tu turno: antes deja todo en PMT y baja tus procesos.
- Sé breve y gasta pocos tokens: no leas de más, no hagas resúmenes largos, no repitas lo que dice PMT.
- Si no puedes avanzar por algo del proyecto, anótalo en una línea en [[`{{AGENTES_DIR}}/para-usuario.md`||el archivo para el usuario que indica `.pmt/entorno.md`]] y detente.

Sesiones: una sesión dura un solo encargo. Haz solo el tuyo, deja todo registrado en PMT y avisa. DP, Dev, CR, QAP y QA avisan siempre al TL, nunca entre ellos: él reinicia tu sesión y levanta al siguiente. Dev conserva su sesión hasta que su grupo de tareas esté en `done`, para corregir con contexto. Si te dan un encargo, léelo en PMT: no cuentas con memoria de sesiones anteriores. Lo que deba conservarse va en PMT.

Bugs (solo con el proyecto en `working`, por pruebas de rutina o uso del sistema): PP los crea, QA confirma o rechaza, el TL indica el dev en un comentario, DP planea, Dev corrige, CR revisa el código sin ejecutarlo, QA prueba y el TL cierra. No tienen dueño ni etapa. Solo PP y el usuario ejecutan `create_bug`. Si detectas un fallo y no eres PP, avisa al TL: él responde por su etapa. Un fallo en desarrollo no es bug: se corrige en su etapa o PP lo agrega como tarea.

Agentes y su nombre en PMT:[[
| Agente | PMT | Perfil |
|---|---|---|
{{TABLA_AGENTES}}|| están en la tabla de agentes de `.pmt/entorno.md`.]]
