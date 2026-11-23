# Plan: PMT en producción y PP con un solo comando

Acordado con el usuario el 21 de noviembre de 2026. Resuelve la deuda técnica "Levantar un agente es complejo".

## Objetivo

- PMT corre en la Raspberry Pi 5 del usuario y se expone en `https://pmt.saulantonio.com` con Cloudflare Tunnel, como DBP.
- Lo usan varios compañeros. No tienen acceso a la configuración, variables ni archivos de la máquina del usuario.
- En cada proyecto, un compañero levanta a PP con un solo comando, ya conectado al MCP de PMT y con su perfil cargado.

## Lo que hoy lo impide

1. Sin token, cualquiera actúa como `saul` con perfil `User`. En internet, cualquiera podría planear, borrar o cancelar.
2. La identidad es por agente y por proyecto (`dbp-tl`, `fms-qa`…). Los tokens se crean con `pmt-server agent` en la máquina del servidor y viven en `~/.config/pmt/tokens/` del usuario.
3. Los perfiles viven en `agentes/perfiles/` de este repo y los arma `iniciar.sh` con `.agentes.env`.
4. `import_project` e `import_stage` leen una ruta en la máquina del servidor.
5. El kit son scripts de bash de este repo con rutas de esta máquina.

## Decisiones

### Identidad y acceso
- Cada compañero es un **usuario** con correo y contraseña, como en DBP. El dueño se crea al arrancar desde la configuración; los demás entran por invitación.
- El frontend entra con login y cookie. Ya no hay acceso sin token: todo responde 401 salvo `/healthz`, `/readyz`, login e invitación.
- Los agentes usan un **token de API del usuario**. Cada sesión declara su **perfil** (PP, PC, TL, DP, DevFrontend, DevBackend, CR, QAP o QA). Cualquier usuario puede usar cualquier perfil. El actor pasa a ser usuario más perfil, y las reglas de permisos actuales siguen igual.
- El historial actual se conserva: los nombres viejos de agentes quedan como texto en los registros.

### Proyectos compartidos (como DBP)
- Quien crea un proyecto es su **owner**. Lo comparte como **editor** (trabaja en él) o **viewer** (solo lo ve).
- Solo el owner y los editores trabajan en un proyecto con agentes. El **viewer solo consulta desde el sitio web**: con su token, por MCP o REST, el proyecto no está disponible.
- El TL, los devs y el dueño de una tarea, si son usuarios, deben ser **owner o editor** del proyecto.
- Un proyecto no compartido solo lo ve su owner, ni en la interfaz ni por MCP, incluido `list_projects`.

### Bloqueo por persona
- **Etapa:** al iniciarla con `start_stage`, queda tomada por esa persona. Sus agentes, de cualquier perfil, trabajan en ella y en sus tareas. Los de otra persona pueden leerla, pero cualquier escritura sobre esa etapa o sus tareas, incluidos `revise_*` y `retire_*`, responde `conflict` nombrando a quien la tiene.
- **Liberación:** al cerrarla (`done`) o cancelarla. Detenida (`stopped`) sigue tomada. El owner del proyecto puede liberarla a mano.
- **Bugs:** quien lo pasa a `in_dev` lo toma, con las mismas reglas.
- **Etapas nuevas:** se pueden agregar aunque otra persona trabaje otra etapa.
- Registro del proyecto: nombre y descripción se editan en cualquier momento, como antes (decisión del 22 de noviembre de 2026).

### Datos
- Al montar la Raspberry se copian los dos proyectos actuales, DBP y PMT, con todo su historial. El usuario queda como owner de ambos.

### Equipo
- Por regla se usa herdr: `pmt pp` y `pmt equipo` corren ahí.

## Fases

### Fase 1. Identidad, proyectos compartidos y bloqueos (backend)
- Usuarios, login con cookie, invitaciones y tokens de API, tomando el patrón de la feature `auth` de DBP.
- Perfil por sesión en el MCP (por ejemplo `/mcp?profile=pp`) y en REST.
- Quitar el acceso sin token y el comando `pmt-server agent`. Las cuentas se administran desde la interfaz.
- Miembros de proyecto con roles owner, editor y viewer, y filtrado de todas las lecturas por membresía.
- Bloqueo de etapas y bugs por persona.
- Frontend: login, invitación, administración de miembros y tokens.

### Fase 2. Perfiles servidos por PMT
- `agentes/perfiles/*.md` y `agentes/comun.md` se incrustan en el binario.
- Al conectarse con un perfil, el MCP da unas `instructions` cortas que piden leer el perfil completo con la herramienta `get_profile` y después `.pmt/entorno.md` (Claude Code solo muestra ~4 000 caracteres de las instrucciones). Además, `GET /api/profiles/{perfil}` devuelve el mismo texto. Los tokens de agente del kit no reciben instrucciones.
- Lo que hoy rellena `iniciar.sh` en los perfiles (puertos, base, descripción del proyecto) pasa a la configuración local del proyecto que lee el CLI.

### Fase 3. Importar por contenido
- `import_project` e `import_stage` reciben el contenido del YAML en vez de una ruta. El formato y las validaciones no cambian.

### Fase 4. CLI `pmt`
- Instalación con `curl -fsSL https://pmt.saulantonio.com/install.sh | sh`, para Linux y macOS, x86 y arm.
- `pmt login`: abre el navegador, el compañero entra y el token queda en su `~/.config/pmt/`.
- `pmt pp`, en la carpeta del proyecto: la primera vez pregunta lo mínimo (proyecto de PMT, herramienta y modelo) y lo guarda en el proyecto, fuera de git. Luego lanza Claude o Codex en herdr con el MCP de `pmt.saulantonio.com`, su token y el perfil PP.
- `pmt equipo` levanta los demás perfiles en herdr. `pmt avisar`, `pmt levantar`, `pmt reiniciar` y `pmt vigia` sustituyen a los scripts del kit.
- `pmt pp` y `pmt equipo` pasan el perfil completo (de `GET /api/profiles/{perfil}`) al lanzar a Claude o Codex, como hoy hace el kit; las instrucciones del MCP y `get_profile` son el respaldo.
- La configuración local del proyecto (cadena de su base, puertos) la genera `pmt` y queda en el proyecto del compañero, fuera de git.

**Decisiones acordadas con el usuario** (22 de noviembre de 2026):

1. **Lenguaje y lugar:** Go, en este repo, `backend/cmd/pmt` (mismo módulo). Un binario sin dependencias para Linux y macOS, amd64 y arm64.
2. **Instalación:** `pmt-server` sirve `/install.sh` y los binarios. `curl -fsSL https://pmt.saulantonio.com/install.sh | sh` instala en `~/.local/bin/pmt`.
3. **`pmt login`:** flujo con código por navegador (como `gh auth login`); crea un token de API del usuario con el nombre de la máquina.
4. **Configuración del proyecto:** `.pmt/` en la raíz del proyecto, fuera de git: `config.yaml` (proyecto de PMT, herramienta y modelo por perfil, puertos, comando de consulta de la base, workspace de herdr) y `entorno.md` generado.
5. **`pmt pp`** la primera vez pregunta solo proyecto de PMT (owner o editor, o «nuevo, lo crea PP»), herramienta y modelo. Ruta, puertos y base los pregunta `pmt equipo`.
6. **Herramientas:** solo **Claude y Codex**. Qwen y mcode (MiniMax) quedan fuera.
7. **`pmt equipo`:** levanta los otros ocho perfiles en herdr con el token del usuario y `?profile=`. Propone Opus para DP, TL y CR, y Codex `gpt-6.1-terra` para PC, devs, QAP y QA.
8. **Comandos del equipo:** `pmt avisar`, `pmt levantar`, `pmt reiniciar`, `pmt vigia`, que `entorno.md` lista.
9. **MCP extra** por perfil (por ejemplo DBP para PP y TL), con su token en `~/.config/pmt/`.
10. **Tokens de agente y `pmt-server agent`:** se retiran en la fase 6.
11. **Partes:** 4a (binario, instalación, login), 4b (`pp`, `config.yaml`, `entorno.md`), 4c (`equipo` y comandos del equipo).

**4a, hecha:** `pmt login`, `logout`, `whoami` y `version`; login por navegador con `/api/cli/login` y la pantalla `/cli`; `pmt-server` sirve `/install.sh` y `/descargas/` desde `cli.dist`; la imagen lleva los cuatro binarios ([arquitectura.md](arquitectura.md#cli-pmt), [operacion.md](operacion.md#cli-pmt)).

**4b, hecha:** `pmt pp` y `pmt pp --configurar`, `.pmt/config.yaml` y `.pmt/entorno.md` ([arquitectura.md](arquitectura.md#pmt-pp), [perfiles.md](perfiles.md#formato-de-pmtconfigyaml), [operacion.md](operacion.md#cli-pmt)). Sin cambios a las decisiones.

### Fase 5. Despliegue en la Raspberry Pi 5
- Imagen arm64 con el `Dockerfile` actual.
- `docker-compose.prod.yml` como el de DBP: PMT y PostgreSQL con su volumen, sin puertos expuestos, y Cloudflare Tunnel hacia `pmt.saulantonio.com`.
- Copia de la base local con `pg_dump`, asignando al usuario como owner de DBP y PMT.
- Respaldo periódico de la base en la Raspberry.
- Esta máquina pasa a ser un cliente más, con `pmt login`.

### Fase 6. Documentación y retiro del kit
- Guía de una página para compañeros: instalar, login y `pmt pp`.
- Retirar `agentes/` cuando `pmt` lo cubra.
- Actualizar `.docs/`, la skill `equipo-pmt` y quitar la deuda técnica.

## Orden

Las fases van en orden. Con las fases 1, 3 y 5, PMT queda en internet de forma segura, aunque sin `pmt pp`: cada compañero tendría que configurar el MCP a mano. Las fases 2 y 4 dan el comando único.
