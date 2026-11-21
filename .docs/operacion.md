# Operación

Cómo se corre PMT en esta máquina. Los comandos de Go se ejecutan desde `backend/`.

## Piezas

| Pieza | Dónde |
| --- | --- |
| PostgreSQL | Contenedor Docker `postgresql` (postgres 18), base `pmt`. Arranca solo con Docker |
| `pmt-server` | `backend/bin/pmt-server`, en `127.0.0.1:8080`, con su log en `backend/bin/pmt-server.log` |
| Configuración | `backend/config/config.dev.yaml`, fuera de git ([arquitectura.md](arquitectura.md#configuración)) |
| Frontend | `frontend/dist`, servido por `pmt-server` |
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

Si hubo cambios en el frontend, primero `cd frontend && pnpm build`. Si el binario nuevo trae migraciones, respalda antes la base (ver Migraciones): las aplica al arrancar.

La imagen de contenedor se construye desde la raíz con `docker build -t pmt .` y se corre con `DATABASE_CONNECTION_STRING` en el entorno.

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

**Usuarios.** El dueño los ve en `/cuenta` y los desactiva o reactiva. Desactivar cierra sus sesiones y revoca sus tokens; al reactivarlo, crea tokens nuevos.

## Agentes y tokens

Los tokens de agente solo se administran por línea de comandos, para que ningún agente pueda emitirlos. Siguen funcionando igual que antes de los usuarios. `create` y `rotate` imprimen únicamente el token. `create` rechaza un nombre que ya sea `username` de un usuario.

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

## Cambios al código de PMT

- Se hacen en sesión aparte (pestaña Dev del workspace `pmt-v4` en herdr). Si tocan el MCP, un agente `dummy-*` los prueba por MCP.
- No se prueba nunca contra el 8080: las pruebas usan el backend en 8090 con `pmt_frontend_test`.
- Commit y push solo cuando el usuario lo pide.

## Pendiente de operación

- Servicio de usuario de systemd para que `pmt-server` arranque solo.
