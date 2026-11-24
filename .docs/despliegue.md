# Despliegue en la Raspberry (Coolify y Cloudflare Tunnel)

PMT se despliega igual que DBP, que ya corre en la misma Raspberry. Este documento es la guía para hacerlo; [operacion.md](operacion.md) describe cómo corre en la máquina de desarrollo.

## Requisitos

- Raspberry Pi 5 (arm64) con Coolify, con acceso de Coolify a este repositorio.
- Un túnel de Cloudflare (`cloudflared`) ya conectado, corriendo en el mismo host que el proxy de Coolify, para poder apuntar a `localhost:80`.
- El dominio `saulantonio.com` en Cloudflare.

## Crear el recurso

En Coolify, dentro del proyecto, **New Resource** y el repositorio. Build Pack **Docker Compose**, rama `main`, archivo `/docker-compose.prod.yml`.

El compose construye el servicio `pmt` con el `Dockerfile` de la raíz (frontend, `pmt-server` y los binarios del CLI para los cuatro sistemas) y crea el servicio `db` con `postgres:17.12` y el volumen `pmt-pgdata`. No publica puertos: el proxy de Coolify llega al puerto 8080 de `pmt` por la red interna. La imagen se construye en la Raspberry; la primera vez tarda varios minutos.

`pmt version` del CLI descargado muestra el commit: el compose pasa `SOURCE_COMMIT`, que Coolify da al construir, como `PMT_VERSION`. Si falta, dice `dev`.

## Variables

La imagen trae `config/config.prod.yaml` y arranca con `ENV=prod`. No se usa `.env`. En **Environment Variables** del recurso, como secretos:

| Variable | Obligatoria | Valor |
| --- | --- | --- |
| `POSTGRES_PASSWORD` | sí | Contraseña de PostgreSQL de este despliegue |
| `DATABASE_CONNECTION_STRING` | sí | `postgres://pmt:<POSTGRES_PASSWORD>@db:5432/pmt?sslmode=disable`, con la misma contraseña |
| `OWNER_EMAIL` | sí | Correo del dueño de la instancia |
| `OWNER_PASSWORD` | sí | Su contraseña, al menos 8 caracteres |
| `OWNER_USERNAME` | sí | Su `username` (`saul`), el mismo que tiene en la base que se traspasa |
| `APP_PUBLIC_URL` | no | Por omisión `https://pmt.saulantonio.com`. Base de los enlaces de invitación, de la pantalla `/cli` del login del CLI y de `/install.sh` |

El compose falla con `falta VAR` si falta una obligatoria. Si la contraseña de PostgreSQL lleva `@`, `:`, `/` o `#`, hay que codificarlos con percent-encoding en la cadena de conexión; es más simple elegir una alfanumérica. Sin los signos `<` y `>` del ejemplo.

Cada llave de la configuración se sobrescribe con su nombre en mayúsculas y `_` en lugar de `.` ([arquitectura.md](arquitectura.md#configuración)); las previstas están marcadas con `# override:` en `backend/config/config.prod.yaml`. El compose solo pasa las de la tabla: para sobrescribir otra (por ejemplo `AUTH_COOKIE_SECURE`) hay que agregarla también al `environment` del servicio. En producción se deja `auth.cookie_secure: true`, porque el navegador siempre llega por HTTPS.

## Primer arranque y dueño

Al arrancar, `pmt-server` aplica las migraciones y, **si la base no tiene usuarios**, crea al dueño de la instancia con `OWNER_EMAIL`, `OWNER_PASSWORD` y `OWNER_USERNAME`. El log dice `dueño de la instancia creado`. Si ya hay usuarios no hace nada: cambiar estas variables después no cambia la contraseña (eso se hace en `/cuenta`).

Con el [traspaso de datos](#traspaso-de-datos), la restauración reemplaza la tabla `users` con la del respaldo, así que el dueño que se creó al primer arranque desaparece y quedan los usuarios y contraseñas de la máquina de origen. Las variables `OWNER_*` siguen siendo obligatorias para que una base vacía nunca arranque sin nadie que pueda entrar.

## Persistencia y salud

El volumen `pmt-pgdata` guarda los datos. Un redeploy no lo borra; al eliminar el recurso, consérvalo si quieres los datos. `db` usa `pg_isready`; `pmt` espera a que esté healthy y su healthcheck es `/readyz` (503 si la base no responde). `/healthz` solo comprueba el proceso.

## Dominio y túnel

1. En Coolify, el dominio del servicio `pmt` es `http://pmt.saulantonio.com:8080`: Cloudflare termina TLS y Coolify enruta al puerto 8080 del contenedor.
2. En Cloudflare Zero Trust, **Networks > Tunnels**, el túnel de la Raspberry, **Public Hostname**: `pmt.saulantonio.com` hacia `HTTP` `localhost:80` (el proxy de Coolify). El túnel crea el CNAME.
3. En el panel del dominio, **Network > WebSockets** activado. Los avisos en vivo de la web van por WebSocket en `/api/events`.

La web, `/api`, `/api/events`, `/mcp`, `/install.sh` y `/descargas/` comparten el origen; no hacen falta reglas por ruta. `/api/events` no valida el encabezado `Origin`: se autentica con la cookie de sesión (o con token) y solo emite nombres de recursos.

## Cloudflare Access (opcional)

PMT ya exige login en la web y token en `/mcp` y `/api`, así que Access no es necesario. Si se usa, hay que dejar pasar todo lo que usan clientes sin navegador, que no pueden hacer el login de Access:

| Ruta | Quién la usa sin navegador |
| --- | --- |
| `pmt.saulantonio.com/mcp*` | Claude Code, Codex y los agentes, con `Authorization: Bearer` |
| `pmt.saulantonio.com/api*` | El CLI `pmt` (`/api/projects`, `/api/profiles`, `/api/auth/me`, `/api/cli/login` y `/api/cli/login/poll`) y los agentes por REST, con token |
| `pmt.saulantonio.com/install.sh` | `curl -fsSL …/install.sh \| sh` |
| `pmt.saulantonio.com/descargas/*` | `install.sh` al bajar el binario y `SHA256SUMS` |

Access no distingue una petición con token de una sin él, así que el bypass de `/api` es completo: con esas reglas, Access solo protege las páginas de la web, cuyos datos igual llegan por `/api` con la sesión de PMT. Para cada ruta, una aplicación **Self-hosted** con política **Bypass**, **Include: Everyone**; la aplicación general de `pmt.saulantonio.com` con tu política. Access aplica la ruta más específica.

Con Access, `curl -i https://pmt.saulantonio.com/` sin sesión de Access debe dar 302 al login de Access, y `curl -i -X POST https://pmt.saulantonio.com/mcp` debe dar 401 de PMT, no la página de Access.

## Actualizar

Tras subir los cambios a `main`, **Redeploy** en Coolify (o su webhook). Las migraciones nuevas se aplican solas al arrancar. Antes de un redeploy con migraciones, un respaldo a mano con `scripts/backup.sh`.

## Verificar

1. `curl -s -o /dev/null -w '%{http_code}\n' https://pmt.saulantonio.com/readyz` → `200`.
2. Abrir `https://pmt.saulantonio.com` y entrar con el usuario dueño. Abrir el mismo proyecto en dos pestañas, cambiar algo en una y ver el aviso en la otra (WebSocket).
3. MCP: en `/conectar`, crear un token y probar
   ```sh
   claude mcp add --transport http pmt https://pmt.saulantonio.com/mcp --header "Authorization: Bearer <token>"
   ```
   y listar proyectos. `curl -i -X POST https://pmt.saulantonio.com/mcp` sin token → 401.
4. CLI, en cualquier equipo:
   ```sh
   curl -fsSL https://pmt.saulantonio.com/install.sh | sh
   pmt version     # el commit desplegado
   pmt login       # abre el navegador en https://pmt.saulantonio.com/cli?codigo=…
   pmt whoami
   ```
   `curl -fsSL https://pmt.saulantonio.com/install.sh | head` debe mostrar `https://pmt.saulantonio.com` como servidor.

## Respaldos

Los scripts corren en el host con Bash y acceso a Docker. Las tareas programadas de Coolify corren dentro de un contenedor sin Docker: se usa cron del host. Primero, el nombre real del contenedor de la base, que genera Coolify:

```sh
docker ps --filter name=db --format '{{.Names}}'
```

Se elige el del recurso de PMT (no el de DBP) y se usa como `BACKUP_CONTAINER`.

| Variable | Por omisión |
| --- | --- |
| `BACKUP_CONTAINER` | obligatoria |
| `BACKUP_DB_USER` | `pmt` |
| `BACKUP_DB` | `pmt` |
| `BACKUP_DIR` | `./backups` |
| `BACKUP_KEEP` | `14` |

`scripts/backup.sh` hace `pg_dump -Fc` dentro del contenedor, guarda `pmt-AAAAMMDD-HHMMSS.dump` en `BACKUP_DIR` y conserva los últimos `BACKUP_KEEP`. Imprime la ruta del archivo. Ejemplo de `crontab -e`, cada día a las 03:30 (DBP respalda a las 03:00):

```cron
30 3 * * * BACKUP_CONTAINER=<contenedor> BACKUP_DIR=/srv/pmt-backups /ruta/al/repositorio/scripts/backup.sh >> /var/log/pmt-backup.log 2>&1
```

El usuario de cron debe poder usar Docker y escribir en `BACKUP_DIR` y en el log. Los scripts se toman de un clon del repositorio en la Raspberry (Coolify no deja el código en una ruta fija). Copia `BACKUP_DIR` también fuera de la Pi, por ejemplo con `rsync` desde otra máquina:

```sh
rsync -a <usuario>@<raspberry>:/srv/pmt-backups/ ~/Respaldos/pmt-pi/
```

**Restaurar.** Detener el servicio `pmt` en Coolify y dejar `db` en marcha. La base de destino debe existir (`pmt` existe siempre). Pide escribir el nombre de la base para confirmar, porque reemplaza los objetos del respaldo:

```sh
BACKUP_CONTAINER=<contenedor> /ruta/al/repositorio/scripts/restore.sh /srv/pmt-backups/<archivo>.dump pmt
```

Para automatizar se agrega `--yes`. Al terminar, arrancar `pmt` y comprobar `/readyz`.

## Traspaso de datos

La base de la máquina de desarrollo (contenedor `postgresql`, base `pmt`, PostgreSQL 18) pasa a la Raspberry (PostgreSQL 17). `pg_restore` 17 restaura el formato custom de `pg_dump` 18 con el esquema de PMT (probado con `pmt_frontend_test`).

1. **Desplegar primero** en Coolify con las variables, para que existan el volumen y la base y se apliquen las migraciones. La imagen desplegada debe ser del mismo commit (o posterior) que el servidor de desarrollo, para que el respaldo no traiga migraciones que la imagen no conoce.
2. **Respaldo en la máquina de desarrollo**, con el 8080 detenido para que no cambie nada a medio camino:
   ```sh
   kill $(pgrep -f bin/pmt-server$)
   docker exec postgresql pg_dump -Fc -U <usuario> -d pmt > ~/Respaldos/pmt/traspaso-$(date +%Y%m%d).dump
   ```
3. **Copiar** a la Raspberry: `scp ~/Respaldos/pmt/traspaso-<fecha>.dump <usuario>@<raspberry>:/srv/pmt-backups/`.
4. En la Raspberry, **detener `pmt`** en Coolify (dejar `db`) y **restaurar**:
   ```sh
   BACKUP_CONTAINER=<contenedor> /ruta/al/repositorio/scripts/restore.sh /srv/pmt-backups/traspaso-<fecha>.dump pmt
   ```
5. **Limpiar lo que dependía de la máquina de origen** (ver abajo):
   ```sh
   docker exec -i <contenedor> psql -U pmt -d pmt -v ON_ERROR_STOP=1 <<'SQL'
   DELETE FROM sessions;
   DELETE FROM cli_logins;
   DELETE FROM invitations WHERE accepted_at IS NULL;
   SQL
   ```
6. **Dejar al usuario como owner de todos los proyectos**, incluidos DBP y PMT. El owner anterior de cada proyecto queda como editor y los demás miembros no cambian:
   ```sh
   BACKUP_CONTAINER=<contenedor> /ruta/al/repositorio/scripts/owner.sh saul
   ```
   Imprime de cuántos proyectos es owner. Falla, sin cambiar nada, si el `username` no existe o está desactivado.
7. **Arrancar `pmt`** en Coolify, comprobar `/readyz` y entrar a la web con el usuario y la contraseña de la máquina de origen.
8. En la máquina de desarrollo, que pasa a ser un cliente: `pmt login --servidor https://pmt.saulantonio.com`, y cambiar `PMT_TOKEN` y los `.mcp.json` a la URL nueva con un token creado en la Raspberry.

### Qué depende de la máquina de origen

| Tabla | Qué hacer |
| --- | --- |
| `sessions` | Borrar. Son las sesiones del navegador en `127.0.0.1:8080`; la cookie no sirve en otro dominio |
| `cli_logins` | Borrar. Son inicios de sesión del CLI ya consumidos o vencidos |
| `invitations` sin aceptar | Borrar. Sus enlaces llevan la `public_url` de desarrollo; se invita de nuevo desde `/cuenta` |
| `api_tokens` | Se pueden conservar: solo se guarda el hash, y los tokens en texto viven en `~/.config/pmt/` de la máquina de origen. Siguen sirviendo contra la Raspberry. Los de prueba o de la `pmt <máquina>` de desarrollo se revocan en `/conectar` cuando ya no se usen |
| `agents` | Igual que los tokens: los del kit siguen sirviendo si el kit apunta a la Raspberry. `dev` y `dummy-*` son de pruebas del MCP; conviene revocarlos si no se van a usar allá: `docker exec <contenedor de pmt> /app/pmt-server agent revoke --name <nombre>` ([operacion.md](operacion.md#agentes-y-tokens)) |

## Problemas comunes

- **502**: comprobar que ambos contenedores estén healthy y revisar sus logs en Coolify.
- **El compose no arranca con `falta VAR`**: falta una variable obligatoria en Coolify.
- **`pmt` reinicia en bucle tras un traspaso**: el respaldo trae una migración más nueva que la imagen; desplegar el commit correcto.
- **No llegan los avisos en vivo**: WebSockets desactivado en Cloudflare, o Access sin bypass de `/api*` bloqueando `/api/events`.
- **`install.sh` o `pmt login` apuntan a otra URL**: `APP_PUBLIC_URL` en Coolify tiene un valor viejo; corregirlo o borrarlo y redeploy.
- **`pmt version` dice `dev`**: la construcción no recibió `SOURCE_COMMIT`; revisar que el recurso sea de tipo Docker Compose desde git.
- **El MCP o el CLI reciben HTML o 302**: falta el bypass de Access para `/mcp*` o `/api*`.
- **Bucle de redirección**: el dominio en Coolify debe ser `http://`, porque Cloudflare termina TLS.
- **No se puede entrar tras el traspaso**: la contraseña es la de la máquina de origen, no `OWNER_PASSWORD`.
