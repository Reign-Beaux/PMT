# Operación

Cómo se corre PMT en esta máquina. Los comandos de Go se ejecutan desde `backend/`.

## Piezas

| Pieza | Dónde |
| --- | --- |
| PostgreSQL | Contenedor Docker `postgresql` (postgres 18), base `pmt`. Arranca solo con Docker |
| `pmt-server` | `backend/bin/pmt-server`, en `127.0.0.1:8080`, con su log en `backend/bin/pmt-server.log` |
| Configuración | `backend/config/config.dev.yaml`, fuera de git ([arquitectura.md](arquitectura.md#configuración)) |
| Frontend | `frontend/dist`, servido por `pmt-server` |
| Binarios del CLI `pmt` | `dist/cli/` en la raíz del repo, fuera de git, servidos por `pmt-server` en `/install.sh` y `/descargas/` |
| Tokens | De agente: `~/.config/pmt/tokens/<agente>`, permiso 600. De usuario: se crean en la pantalla Cuenta |
| Respaldos viejos | `~/Respaldos/pmt/` (`pg_dump` de octubre y noviembre de 2026) |
| Base de pruebas del frontend | `pmt_frontend_test`, en el mismo contenedor |

## Arrancar

`pmt-server` **no arranca solo** al reiniciar la máquina. Se arranca desde `backend/`, porque lee `config/config.dev.yaml` relativo al directorio de arranque, y aplica las migraciones pendientes antes de abrir el puerto.

```bash
cd backend
pgrep -af bin/pmt-server        # ¿ya corre?
setsid nohup ./bin/pmt-server >> bin/pmt-server.log 2>&1 < /dev/null &
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/readyz   # 200
```

Una sesión de Claude Code que se abrió con el servidor apagado no tiene el MCP `pmt`: se reconecta con `/mcp`. El MCP exige token: una sesión del usuario necesita `PMT_TOKEN` con un token suyo (ver Usuarios y acceso).

## Compilar y reemplazar el servidor

```bash
cd backend
go vet ./... && go test -race ./...
go build -o bin/pmt-server.nuevo ./cmd/pmt-server
kill $(pgrep -f bin/pmt-server$)   # cierra limpio con SIGTERM
mv bin/pmt-server.nuevo bin/pmt-server
setsid nohup ./bin/pmt-server >> bin/pmt-server.log 2>&1 < /dev/null &
```

Si hubo cambios en el frontend, primero `cd frontend && pnpm build`. Si hubo cambios en el CLI, recompila sus binarios (ver [CLI pmt](#cli-pmt)); eso no pide reiniciar. Si el binario nuevo trae migraciones, respalda antes la base (ver Migraciones): las aplica al arrancar.

La imagen de contenedor se construye desde la raíz con `docker build -t pmt --build-arg PMT_VERSION=$(git describe --tags --always) .` y se corre con `DATABASE_CONNECTION_STRING` en el entorno. Ya trae los binarios del CLI en `/app/cli`. Si `docker build` falla con `DeadlineExceeded` al leer los metadatos de las imágenes, es el gestor de credenciales de Docker (`gpg`): se puede construir con `DOCKER_CONFIG` apuntando a una carpeta con `{"currentContext":"default"}`, que baja las imágenes públicas sin credenciales.

## Migraciones

Van embebidas en el binario y `pmt-server` aplica las que falten al arrancar. No hay paso manual: para migrar se reemplaza el binario y se arranca. Para ver en qué versión está una base:

```bash
docker exec postgresql psql -U <usuario> pmt -qtAc "select max(version_id) from goose_db_version"
```

Antes de arrancar un binario con migraciones nuevas contra la base real, respáldala:

```bash
docker exec postgresql pg_dump --clean --if-exists -U <usuario> pmt > ~/Respaldos/pmt/respaldo-<motivo>-<fecha>.sql
```

Para restaurar: `docker exec -i postgresql psql -U <usuario> pmt -v ON_ERROR_STOP=1 < <respaldo>`.

## Usuarios y acceso

PMT no tiene acceso anónimo: el navegador entra con login y los clientes MCP con un token.

**Dueño de la instancia.** Al arrancar, si la base no tiene usuarios, `pmt-server` crea al dueño con `owner.email`, `owner.password` y `owner.username` de la configuración (en esta máquina, `saul`, para que el historial existente siga siendo suyo). El log dice `dueño de la instancia creado`. Si ya hay usuarios no hace nada, así que cambiar `owner.*` después no cambia la contraseña: eso se hace en Cuenta. Sin `owner.email` u `owner.password` no crea a nadie y nadie puede entrar.

**Invitar.** El dueño entra a `/cuenta`, escribe el correo en Invitaciones y copia el enlace (`app.public_url` + `/invitacion/<token>`); PMT no envía correos. El invitado abre el enlace, elige nombre, `username` y contraseña y queda con sesión. La invitación vence en 7 días y sirve una vez.

**Tokens de usuario.** Cada usuario los crea en `/cuenta` con un nombre (por ejemplo «laptop»). Empiezan con `pmt_` y se muestran una sola vez. Para el MCP:

```bash
install -m 600 /dev/null ~/.config/pmt/tokens/<usuario>   # pega ahí el token
export PMT_TOKEN=$(cat ~/.config/pmt/tokens/<usuario>)     # antes de abrir Claude Code
```

`.mcp.json` manda `Authorization: Bearer ${PMT_TOKEN:-}`; sin la variable, el MCP responde 401.

**Agente con token de usuario y perfil.** Un usuario lanza agentes de cualquier perfil con su propio token: cada sesión declara su perfil en la URL del MCP con `?profile=` (`pp`, `pc`, `tl`, `dp`, `devfrontend`, `devbackend`, `cr`, `qap` o `qa`; sin distinguir mayúsculas). Sin `profile`, o con `profile=user`, actúa como `User`. El historial queda con su `username` y el perfil declarado (por ejemplo `saul` + `TL`). Para Claude Code, un archivo por perfil:

```json
{
  "mcpServers": {
    "pmt": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp?profile=tl",
      "headers": { "Authorization": "Bearer ${PMT_TOKEN:-}" }
    }
  }
}
```

y se lanza con `PMT_TOKEN=$(cat ~/.config/pmt/tokens/<usuario>) claude --strict-mcp-config --mcp-config <archivo>`. Codex y Qwen usan la misma URL y la misma cabecera en su configuración de MCP. Un perfil desconocido responde 400 con la lista de válidos. Por REST y `/api/events` el perfil va en la cabecera `X-PMT-Profile`; con la cookie del navegador se ignora. `GET /api/auth/me` devuelve el usuario y `profile`, el perfil efectivo de la petición, para comprobar la configuración.

Para que un usuario trabaje como TL o dev de una etapa, PP lo asigna por su `username` con `assign_stage_lead` o el TL con `add_stage_member`, igual que a un agente. Las operaciones de cuenta (tokens, invitaciones, usuarios) solo se usan sin perfil declarado.

**Usuarios.** El dueño los ve en `/cuenta` y los desactiva o reactiva. Desactivar cierra sus sesiones y revoca sus tokens; al reactivarlo, crea tokens nuevos. Un usuario desactivado conserva sus membresías de proyecto.

## Compartir un proyecto

Cada proyecto tiene miembros ([dominio.md](dominio.md#miembros-y-roles)). Quien lo crea es su owner; un proyecto que no le compartieron no existe para nadie más: no sale en `list_projects` y cualquier ID suyo responde 404 o `not_found`. Los proyectos que existían antes de los miembros quedaron del dueño de la instancia.

Lo comparte el owner, sin perfil declarado: con la cookie, con su token sin `X-PMT-Profile`, o por MCP sin `?profile=` (o con `profile=user`). Todavía no hay pantalla (fase 1e del [plan](plan_produccion.md)):

| Operación | MCP | REST |
| --- | --- | --- |
| Ver miembros (cualquier miembro) | `list_project_members` | `GET /api/projects/{id}/members` |
| Compartir como `editor` o `viewer` | `add_project_member` (`project_id`, `username`, `role`) | `POST /api/projects/{id}/members` `{"username","role"}` |
| Cambiar entre `editor` y `viewer` | `update_project_member` | `PUT /api/projects/{id}/members/{username}` `{"role"}` |
| Dejar de compartir | `remove_project_member` | `DELETE /api/projects/{id}/members/{username}` |

```bash
curl -X POST http://127.0.0.1:8080/api/projects/<id>/members \
  -H "Authorization: Bearer $PMT_TOKEN" -H 'Content-Type: application/json' \
  -d '{"username":"ana","role":"editor"}'
```

El usuario debe existir y estar activo (primero se le invita). El owner no se quita ni cambia de rol, y no se transfiere.

Solo el owner y los editores trabajan en un proyecto con agentes. Un `viewer` solo lo consulta desde el sitio web: con su token (MCP o REST) el proyecto no sale en `list_projects` y cualquier operación sobre él responde `forbidden` («eres viewer de este proyecto: solo puedes consultarlo en el sitio web»). Para que alguien sea TL, dev o dueño de una tarea, compártelo antes como `editor`: con un viewer o un no miembro, `assign_stage_lead`, `add_stage_member` y `assign_task` responden `conflict`.

Los tokens de agente (`pmt-server agent`) actúan con la membresía del dueño de la instancia: ven y tocan sus proyectos y los que le compartieron, y lo que crean queda suyo.

### Etapas y bugs tomados

En un proyecto compartido, la etapa que alguien inicia (o el bug que pasa a `in_dev`) queda tomada por esa persona hasta cerrarla o cancelarla ([dominio.md](dominio.md#bloqueos-por-persona)). Los agentes de otra persona la leen, pero al escribir reciben `conflict`: «la etapa la está trabajando ana». `list_stages`, `list_bugs` y `get_project_progress` muestran quién la tiene en `taken_by` y desde cuándo en `taken_at`.

Si quien la tiene no puede seguir, el owner del proyecto la libera, sin perfil declarado o con perfil PP, con un motivo que queda en el historial de status:

| Libera | MCP | REST |
| --- | --- | --- |
| Una etapa | `release_stage` (`stage_id`, `reason`) | `POST /api/stages/{id}/release` `{"reason"}` |
| Un bug | `release_bug` (`bug_id`, `reason`) | `POST /api/bugs/{id}/release` `{"reason"}` |

```bash
curl -X POST http://127.0.0.1:8080/api/stages/<id>/release \
  -H "Authorization: Bearer $PMT_TOKEN" -H 'Content-Type: application/json' \
  -d '{"reason":"ana no puede seguir"}'
```

El status no cambia y queda sin tomar: escribir en ella no la toma. La toma de nuevo quien la inicie con `start_stage`, la pase otra vez a `in_dev` (reanudar o reabrir) o, en un bug, quien lo pase a `in_dev`.

## CLI pmt

`pmt` inicia sesión en PMT desde la terminal y guarda el token del usuario en `~/.config/pmt/credenciales.yaml` ([arquitectura.md](arquitectura.md#cli-pmt)). Tiene `login`, `logout`, `whoami`, `version` y `pp`; `equipo` y los comandos del equipo vienen en la fase 4c del [plan](plan_produccion.md).

**Compilar los binarios** (Linux y macOS, amd64 y arm64, con `SHA256SUMS`), desde `backend/`:

```bash
scripts/build-cli.sh                 # en ../dist/cli, con la versión de git describe
scripts/build-cli.sh /otra/carpeta v1.2.0
```

`config.dev.yaml` tiene `cli.dist: ../../dist/cli`, así que el 8080 los sirve en cuanto existen, sin reiniciar. En la imagen ya vienen compilados.

**Cómo instala un compañero** (Linux o macOS, sin `sudo`):

```bash
curl -fsSL https://pmt.saulantonio.com/install.sh | sh   # deja pmt en ~/.local/bin
pmt login                                                # abre el navegador; entra y pulsa «Autorizar»
pmt whoami
```

- `install.sh` verifica la suma SHA-256 y avisa si `~/.local/bin` no está en el `PATH`. Para actualizar, se corre otra vez.
- `pmt login` muestra un código y abre `https://pmt.saulantonio.com/cli?codigo=…`. Sin navegador (por SSH, o con `--sin-navegador`), se abre esa URL en cualquier equipo con sesión en PMT. El código vence en 10 minutos.
- Al autorizar se crea un token llamado `pmt <máquina>`, que aparece en `/cuenta` y se puede revocar ahí. `pmt logout` lo revoca y borra el archivo.
- Contra otro servidor: `pmt login --servidor http://127.0.0.1:8080`. Queda guardado.

**Cómo un compañero levanta a PP** (necesita herdr y `claude` o `codex` instalados, y haber hecho `pmt login`):

```bash
herdr                      # si no está dentro de herdr
cd ~/ruta/al/proyecto
pmt pp                     # la primera vez pregunta proyecto de PMT, herramienta, modelo y effort
pmt pp --configurar        # vuelve a preguntar, con lo actual por omisión
```

- La primera vez lista sus proyectos de PMT donde es owner o editor, más «Nuevo: lo crea PP». Propone el que se llame como la carpeta, o el nuevo. Herramienta `claude` (modelo `opus`) o `codex` (`gpt-6.1-terra`), effort `medium` o `high`. Lo guarda en `.pmt/config.yaml` y agrega `.pmt/` al `.gitignore` si el proyecto es un repo git.
- Con «nuevo», en cada `pmt pp` siguiente busca proyectos de los que sea owner creados después; si encuentra, pregunta cuál es y lo guarda. No detiene el arranque.
- Cada arranque regenera `.pmt/entorno.md`, descarga el perfil a `.pmt/perfiles/pp.md` (si el servidor no responde, usa la copia y avisa), y lanza a PP en la pestaña `pp` del workspace de herdr (por omisión, el nombre de la carpeta; lo crea si falta). Si PP ya corre ahí, no lo relanza: enfoca su pestaña.
- La primera vez en una carpeta, Claude pregunta si se confía en ella; se responde una vez. Codex no pregunta: `pmt` la marca confiable en esa sesión.
- Codex usa `~/.codex` salvo que `codex_home` de `.pmt/config.yaml` diga otra carpeta. En esta máquina se pone `codex_home: ~/.codex-flow`, porque `~/.codex` tiene un `pmt` de la v3.
- MCP extra para PP (por ejemplo DBP): se guarda su token en un archivo dentro de `~/.config/pmt/` (permiso 600) y se declara en `.pmt/config.yaml` ([formato](perfiles.md#formato-de-pmtconfigyaml)).

## Agentes y tokens

Los tokens de agente solo se administran por línea de comandos, para que ningún agente pueda emitirlos. Actúan con la membresía del dueño de la instancia ([Compartir un proyecto](#compartir-un-proyecto)). Por lo demás siguen funcionando igual que antes de los usuarios: su perfil es fijo, así que `?profile=` o `X-PMT-Profile` solo se aceptan si coinciden con él (si no, 400). `create` y `rotate` imprimen únicamente el token. `create` rechaza un nombre que ya sea `username` de un usuario.

```bash
cd backend
./bin/pmt-server agent create --name <nombre> --profile <PP|PC|TL|DP|DevFrontend|DevBackend|CR|QAP|QA> > ~/.config/pmt/tokens/<nombre>
chmod 600 ~/.config/pmt/tokens/<nombre>
./bin/pmt-server agent rotate --name <nombre>
./bin/pmt-server agent revoke --name <nombre>
./bin/pmt-server agent list
```

El nombre del agente en PMT debe ser el mismo del archivo de token, porque así lo busca el kit. Los equipos actuales usan nombres con prefijo de proyecto (`dbp-tl`, `fms-qa`). `dev` y `dummy-*` son los agentes con los que se prueban los cambios al MCP.

## Equipo de agentes

El kit en `agentes/` levanta a los nueve agentes de un proyecto en herdr. Su manual es `agentes/README.md`. En resumen:

1. Copiar `agentes/agentes.env.example` al proyecto como `.agentes.env`, ajustarlo y crear los tokens.
2. `agentes/iniciar.sh /ruta/al/proyecto/.agentes.env` crea las pestañas, genera perfiles y MCP en `.agentes/` y lanza a los agentes.
3. El usuario platica con PP. El vigía arranca con el TL.

Lo que los agentes necesitan decirle al usuario queda en `.agentes/para-usuario.md` del proyecto.

El texto de los perfiles vive en `backend/internal/features/profiles/catalog/` y va incrustado en el binario: el kit lo lee de ahí para generar `.agentes/perfiles/`, y PMT lo sirve con la herramienta `get_profile` y en `GET /api/profiles/{perfil}`. Una sesión MCP con token de usuario y `?profile=` recibe en sus instrucciones unas líneas que le piden llamar a `get_profile` y leer `.pmt/entorno.md`; los tokens de agente del kit no reciben instrucciones ([perfiles.md](perfiles.md#cómo-se-sirven)). Un cambio a un perfil llega por `get_profile` solo al reemplazar el servidor; a los agentes del kit, al regenerar con `iniciar.sh`.

## Cambios al código de PMT

- Se hacen en sesión aparte (pestaña Dev del workspace `pmt-v4` en herdr). Si tocan el MCP, un agente `dummy-*` los prueba por MCP.
- No se prueba nunca contra el 8080: las pruebas usan el backend en 8090 con `pmt_frontend_test`.
- Commit y push solo cuando el usuario lo pide.

## Pendiente de operación

- Servicio de usuario de systemd para que `pmt-server` arranque solo.
