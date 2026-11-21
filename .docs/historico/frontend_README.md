# Frontend de PMT

Interfaz web de PMT: React 19 + TypeScript + Vite, React Compiler y estilos con Sass (SCSS Modules). UI de consulta en español. Implementa la propuesta aprobada de `codex/propuesta` con datos reales de la API.

## Interfaz

- Navegación principal: Dashboard y Projects. `/` conserva el acceso al listado redirigiendo a `/projects`.
- `/dashboard`: indicadores calculados a partir de proyectos, etapas, tareas y bugs existentes.
- `/projects`: catálogo de proyectos.
- `/projects/:projectId`: descripción visible, indicadores, etapas y responsable de la etapa activa.
- `/projects/:projectId/bugs`: comparte exactamente la cabecera, descripción, indicadores y panel lateral del proyecto; cambia la lista.
- `/projects/:projectId/stages/:stageId`: descripción visible y tabla de tareas; el título es el único acceso al detalle.
- `?task=<id>` y `?bug=<id>` abren el detalle y se conservan al recargar. Los demás parámetros de consulta no se borran al cerrar.
- Detalles con pestañas de Comentarios e Historial, transiciones completas, autor y perfil, fecha/hora y motivo. Todo el texto se muestra como texto plano, conservando saltos de línea.
- Diálogo nativo con foco contenido, Escape, cierre desde el fondo, scroll independiente y devolución del foco sin mover la lista.
- No se añaden entidades ni campos al backend. El porcentaje de avance excluye las tareas canceladas.

Los clientes de API, validadores, selección por URL, conexión WebSocket y reglas de aislamiento proceden de `frontend_og`, que se conserva sin modificaciones. Se mantienen los plugins de React, React Compiler mediante Babel/Rolldown, Oxlint, Vitest y Playwright. Las versiones ya presentes en el scaffold nuevo se conservan; se agregan las dependencias faltantes del OG.

## Requisitos

- Node 24 y pnpm.
- Go (el harness e2e compila su propio backend de pruebas desde `../backend`).
- Docker con el contenedor `postgresql` en marcha y la base `pmt_frontend_test`
  creada (el harness la migra con goose si está vacía).

## Instalación y comandos

```sh
pnpm install        # dependencias
pnpm e2e:install    # Chromium de Playwright dentro de node_modules (PLAYWRIGHT_BROWSERS_PATH=0)

pnpm dev            # servidor de desarrollo en http://localhost:5173 (strictPort)
pnpm build          # tsc -b + vite build → dist/
pnpm lint           # oxlint
pnpm test:run       # Vitest, una ejecución (unitarias: src/ y config/)
pnpm test           # Vitest en modo watch
pnpm test:ui        # Playwright: recorridos UI con HTTP/WebSocket controlados, sin base
pnpm test:e2e       # Playwright contra 5173 + backend de pruebas en 8090
```

## Variables de entorno

Ninguna variable se versiona ni contiene credenciales reales; se pasan por entorno
en cada ejecución.

- `PMT_TEST_DATABASE_URL` — **obligatoria para `pnpm test:e2e`**. URL de la base
  de pruebas; solo se acepta la base `pmt_frontend_test` con protocolo
  postgres/postgresql. Ejemplo con credenciales de marcador:

  ```sh
  PMT_TEST_DATABASE_URL='postgres://<usuario>:<clave>@localhost:5432/pmt_frontend_test?sslmode=disable'
  ```

- `PMT_API_TARGET` — opcional, defecto `http://127.0.0.1:8090`. Destino del proxy
  `/api` de Vite. En esta etapa el frontend solo corre para pruebas: tanto
  `vite.config.ts` como `playwright.config.ts` rechazan al cargar cualquier
  destino que no sea `127.0.0.1:8090` o `localhost:8090` (la instancia real del
  8080 queda protegida). Solo URLs http(s) sin ruta.

Sin `PMT_TEST_DATABASE_URL`, o con una base distinta, `pnpm test:e2e` aborta con
un mensaje antes de levantar servidores o crear datos.

## Proxy de desarrollo

`pnpm dev` levanta Vite en 5173 (strictPort) con proxy al backend de pruebas:

- `/api` → `PMT_API_TARGET` (HTTP, `changeOrigin`).
- `/api/events` → `PMT_API_TARGET` con `ws: true` (WebSocket de eventos, el
  proxy hace el upgrade). Ver la sección "Tiempo real".

## Tiempo real

La UI mantiene un solo WebSocket con el canal de eventos y refresca las vistas
al recibir cambios; no hay sondeo HTTP periódico:

- El backend debe exponer `/api/events` (WebSocket), lo que exige tener aplicada
  la migración `00021_notify_changes`.
- En desarrollo el proxy de Vite hace el upgrade a WebSocket (`ws: true`); en
  producción no hace falta proxy (ver "Despliegue").
- Un solo WebSocket por pestaña: navegar entre vistas no crea conexiones nuevas.
- Reconexión ante caída: las esperas crecen 1, 2, 4… segundos hasta un tope de
  30 s. Si pasan 30 s sin recibir `resync`, los datos se consideran viejos
  (stale) y se sigue reintentando.
- Indicador de conexión en el encabezado (`role="status"`): "En vivo"
  (conectado), "Reconectando…" (reintentando) y "Sin conexión, datos de las
  HH:MM" (stale, con la hora del último dato).
- Las vistas se refrescan por evento: un `invalidate` vuelve a consultar el
  recurso afectado y un `resync` (al reconectar) refresca todo lo visible, así
  que los cambios hechos con la conexión caída aparecen al volver sin recargar.

## Pruebas de la interfaz

`pnpm test:ui` inicia su propio Vite y valida navegación, cabecera compartida, descripciones, apertura por título, pestañas, historial, foco, scroll, errores, recarga por URL, eventos y cinco anchos de pantalla (360–1512 px). Las respuestas controladas solo existen en `e2e/ui`; la aplicación no contiene fixtures ni un modo de demostración.

Los navegadores se instalan con `pnpm e2e:install`. Si ya existe un Chromium de sistema, puede indicarse mediante `PMT_BROWSER_PATH=/ruta/al/chromium`. Las pruebas unitarias y estas pruebas de UI no requieren `PMT_TEST_DATABASE_URL`.

## Harness de pruebas e2e

`pnpm test:e2e` usa el harness de `e2e/harness/` (ver `playwright.config.ts`):

1. **Valida** destino y base de pruebas al cargar la configuración
   (`config/apiTarget.ts`); aborta con mensaje si no coinciden.
2. **Levanta** su propio backend: compila `../backend/cmd/pmt-server` a
   `frontend/.harness/pmt-server-pruebas` (ignorado por git) y lo ejecuta sin
   `config.yaml`, con `DATABASE_URL`, `SERVER_ADDR=127.0.0.1:8090` y
   `FRONTEND_DIST=frontend/dist` (el binario corre desde `.harness`, así que el
   default relativo no aplica). Nunca toca `backend/bin` ni la instancia del
   8080. Después levanta Vite en 5173. Si algo ya está escuchando, lo reutiliza
   (`reuseExistingServer`).
3. **Migra** la base con `go tool goose` solo si no tiene esquema.
4. **Centinela**: crea un proyecto `e2e-<RUN_ID>-centinela` y lo lee por la API
   real para confirmar que el backend en 8090 usa `pmt_frontend_test`; si no,
   falla antes de crear más datos.
5. **Fixtures** con identificador único por ejecución (`e2e-<RUN_ID>-<n>`, ids
   UUID) y **limpieza en `finally`**: borra solo lo creado por la ejecución,
   incluso si la prueba falla, y verifica por SQL que no queda nada
   (`restantes 0` en consola y en las anotaciones del test, sin credenciales).
6. Playwright baja solo los procesos que él levantó.

El recorrido real de `e2e/recorrido.real.spec.ts` crea sus datos mediante el harness y verifica proyectos, etapas, tareas, comentarios, transiciones y foco. El harness exige `PMT_TEST_DATABASE_URL`; sin esta variable no se ejecutan pruebas contra una base real. La configuración mantiene la separación del proyecto de reconexión del OG para incorporar sus pruebas sin interferir con el resto.

Comprobación de limpieza tras una ejecución:

```sh
docker exec postgresql psql -U games_test -d pmt_frontend_test \
  -c "select count(*) from projects where name like 'e2e-%'"
```

## Despliegue

El despliegue real lo hace el usuario. No hacen falta nginx ni un proxy aparte:
`pmt-server` sirve el frontend compilado en su mismo puerto.

1. `pnpm build` en `frontend/` genera `dist/`.
2. Arrancar `pmt-server` desde `backend/`: sirve `frontend/dist` junto con
   `/api`, `/api/events` (WebSocket) y `/mcp` en un solo proceso, con fallback
   SPA (toda ruta que no es API sirve `index.html`, así las rutas profundas
   cargan al recargar).
3. La ruta del compilado se configura con `frontend.dist` (config.yaml) o la
   variable `FRONTEND_DIST`; por defecto es `../frontend/dist` (relativa al
   directorio de arranque). Si `dist` falta, la API arranca igual y las rutas
   de UI responden "frontend no compilado".

Para comprobarlo con la instancia de pruebas (**nunca el 8080**, que es la
instancia real):

```sh
pnpm build                       # en frontend/
cd ../backend
DATABASE_URL='postgres://<usuario>:<clave>@localhost:5432/pmt_frontend_test?sslmode=disable' \
SERVER_ADDR=127.0.0.1:8090 go run ./cmd/pmt-server
```

Con eso, http://127.0.0.1:8090/ abre la UI, el indicador del encabezado muestra
"En vivo" y los cambios hechos por la API se reflejan sin recargar.

Nota: la vitrina de componentes `/dev/componentes` solo existe en desarrollo
(`import.meta.env.DEV`); en el build de producción responde "Página no
encontrada".
