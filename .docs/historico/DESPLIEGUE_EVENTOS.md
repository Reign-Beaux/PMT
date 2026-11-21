# Despliegue del canal de eventos (migración 00021 y WebSocket `/api/events`)

Pasos para desplegar en la instancia real la publicación de cambios y el
endpoint WebSocket. Los hace el usuario; ningún agente los ejecuta contra el
8080 ni la base real. Los comandos usan marcadores: `<DATABASE_URL>` es la URL
de la base (la misma de `config.yaml` o la variable de entorno que ya usa
goose) y las rutas se eligen al ejecutar. Todos los comandos de Go se lanzan
desde `backend/`.

## 1. Requisitos y qué cambia

- Go (el mismo que compila el servidor), PostgreSQL accesible y `pg_dump`.
- Un cliente WebSocket ya instalado para la comprobación (p. ej. `node` ≥ 22,
  que trae `WebSocket` global, o `python3` con `websockets`). Si no hay
  ninguno, no se instala nada: se comprueba solo con `psql`.

Qué cambia:

- **Migración `00021_notify_changes.sql`**: crea la función
  `notify_pmt_change()` y 12 triggers que publican en el canal PostgreSQL
  `pmt_changes` un aviso JSON con la forma `{"resource" : "<recurso>"}`
  (`json_build_object` deja espacios alrededor de los dos puntos) por cada
  INSERT, UPDATE o DELETE confirmado en las tablas que muestra la UI.
- **Binario nuevo**: sirve `GET /api/events`, un WebSocket que entrega
  `{"type":"resync"}` al conectar y `{"type":"invalidate","resource":"..."}`
  tras cada commit.

## 2. Respaldo

```sh
pg_dump --clean --if-exists "<DATABASE_URL>" > <ruta-respaldo>
```

Alternativa local con la base en Docker:

```sh
docker exec postgresql pg_dump --clean --if-exists -U <usuario> <base> > <ruta-respaldo>
```

Si `pg_dump` falla (credenciales, base inaccesible, archivo no escribible),
**no seguir**: sin respaldo no hay reversión posible.

## 3. Guardar el binario anterior

```sh
cp bin/pmt-server <ruta-bin-anterior>
```

Si `bin/pmt-server` no existe es la primera instalación: anotar que no hay
binario anterior y saltar este paso; en la reversión (sección 9) se recompilará
el código de la versión anterior.

## 4. Compilar el binario nuevo

```sh
go build -o <ruta-bin-nuevo> ./cmd/pmt-server
```

Si la compilación falla, no se continúa: no hay binario que desplegar.

## 5. Aplicar la migración

```sh
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="<DATABASE_URL>"
go tool goose -dir internal/platform/postgres/migrations status
go tool goose -dir internal/platform/postgres/migrations up
go tool goose -dir internal/platform/postgres/migrations status
```

- El `status` final debe mostrar `00021` aplicada.
- Si el `status` inicial ya muestra `00021` aplicada, `up` no hace nada
  ("no migration files to run" o equivalente): es correcto, seguir con la
  sección 6.
- Si `up` falla, **no reiniciar el servidor**: revisar el error y, si la
  migración quedó a medias, bajarla con `go tool goose -dir
  internal/platform/postgres/migrations down` y volver al estado anterior.

Nota: un binario nuevo arrancado con la base aún en `00020` funciona para
todo lo demás, pero **no llegan `invalidate`** al WebSocket (solo `resync` al
conectar), porque los triggers no existen. Por eso la migración se aplica
antes de reiniciar.

## 6. Reiniciar pmt-server

```sh
# parar la instancia actual (según cómo esté lanzada)
kill <pid-de-pmt-server>
# arrancar la nueva
<ruta-bin-nuevo>
```

El binario lee `config.yaml` desde su directorio de trabajo; las variables
`DATABASE_URL` y `SERVER_ADDR` tienen prioridad si se quieren fijar por
entorno. Si no arranca, revisar su log y aplicar la reversión (sección 9).

## 7. Frontend en el mismo puerto

`pmt-server` sirve el frontend compilado junto con `/api`, `/api/events` y
`/mcp`, en el mismo puerto: no hace falta un proxy. Antes de reiniciar el
servidor, compilar el frontend con `pnpm build` en `frontend/`. La ruta del
compilado y el resto de los detalles están en `frontend/README.md`, sección
**Despliegue**.

Si algún día se pone un proxy delante, debe hacer el upgrade de WebSocket en
`/api/events` (`proxy_http_version 1.1` y las cabeceras `Upgrade` y
`Connection`), y su tiempo de lectura debe superar los 30 s del ping del
servidor (`pingInterval` en `internal/platform/events/websocket.go`); si no,
el handshake falla o el proxy corta las conexiones.

## 8. Comprobación

**Canal PostgreSQL.** En una sesión interactiva de `psql` contra la base:

```sql
LISTEN pmt_changes;
```

En otra terminal, hacer un cambio confirmado por la API (p. ej. crear un
proyecto por REST o `create_project` por MCP). La sesión con `LISTEN` debe
mostrar:

```text
Asynchronous notification "pmt_changes" with payload "{"resource" : "project"}" received from server process with PID ...
```

Para una comprobación de una sola vez sin sesión interactiva (el cambio va
dentro de la misma sesión):

```sh
psql "<DATABASE_URL>" <<'SQL'
LISTEN pmt_changes;
SELECT pg_sleep(10);
SQL
```

O, con la base en Docker, `docker exec -i <contenedor> psql -U <usuario> <base>`
con el mismo here-document. El aviso se imprime al terminar la sentencia en
curso. Ojo: con `psql -c "LISTEN ...; SELECT pg_sleep(...)"` (varias
sentencias en un solo `-c`) los avisos **no se imprimen**; usar sesión
interactiva o here-document.

Si no llega nada, verificar que la migración está aplicada (`status` de la
sección 5) y que el cambio se confirmó (una transacción revertida no publica).

**WebSocket.** Con `node` (≥ 22), contra el servidor:

```sh
node -e '
const ws = new WebSocket("ws://127.0.0.1:8080/api/events");
ws.onmessage = (e) => console.log(e.data);
ws.onopen = () => console.log("conectado");
ws.onerror = (e) => console.error("error", e.message);
'
```

Al conectar debe recibirse `{"type":"resync"}`. Con el cliente conectado,
hacer un cambio confirmado por REST o MCP: debe llegar
`{"type":"invalidate","resource":"..."}` con el recurso correcto (`project`,
`stage`, `task`, `bug`, `comment` o `status_change`).

Alternativa con `python3` y `websockets`:

```sh
python3 -c '
import asyncio, websockets
async def main():
    async with websockets.connect("ws://127.0.0.1:8080/api/events") as ws:
        while True:
            print(await ws.recv())
asyncio.run(main())
'
```

Un error de handshake (400/426 o cierre inmediato) indica que el servidor no
es el binario nuevo, o que hay un proxy delante sin el upgrade de la sección 7.

## 9. Reversión

1. Parar el servidor nuevo (`kill <pid>`).
2. Bajar la migración:

   ```sh
   go tool goose -dir internal/platform/postgres/migrations down
   go tool goose -dir internal/platform/postgres/migrations status
   ```

   El `status` debe volver a `00020`. Una sola ejecución de `down` baja
   únicamente `00021`.
3. Restaurar el binario anterior:

   ```sh
   cp <ruta-bin-anterior> bin/pmt-server
   ```

4. Arrancar el binario anterior y comprobar que la API responde.

Restaurar el respaldo de la sección 2 **solo si algo más falló** (datos
corruptos, migración a medias irreparable). Con el servidor parado; el
respaldo se hizo con `--clean`, así que reemplaza los objetos existentes:

```sh
psql "<DATABASE_URL>" -v ON_ERROR_STOP=1 -f <ruta-respaldo>
```

Alternativa local con la base en Docker:

```sh
docker exec -i postgresql psql -U <usuario> <base> -v ON_ERROR_STOP=1 < <ruta-respaldo>
```

Tras la reversión, un binario nuevo contra la base en `00020` arranca pero no
entrega `invalidate` (ver nota de la sección 5): es el síntoma esperado de que
la migración no está aplicada.
