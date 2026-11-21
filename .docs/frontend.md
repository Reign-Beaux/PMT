# Frontend

Interfaz web para que el usuario siga los proyectos. **Sobre los proyectos solo consulta:** usa únicamente `GET` y no cambia nada; los cambios los hacen los agentes por MCP. Las únicas escrituras son las de la cuenta: login, invitaciones, tokens, contraseña y usuarios. En producción la sirve `pmt-server` en `http://127.0.0.1:8080`, y se actualiza sola en tiempo real.

Sin sesión, toda ruta lleva a `/login` y vuelve a la pedida al entrar. La sesión es la cookie `pmt_session` del backend; el frontend no guarda credenciales. Si la API responde 401 a media sesión, vuelve a `/login`.

| Necesidad | Elección |
| --- | --- |
| UI | React 19 con React Compiler, React Router |
| Lenguaje y build | TypeScript, Vite |
| Estilos | Sass con CSS Modules, tokens propios, solo tema oscuro, sin bibliotecas de componentes |
| Lint | Oxlint |
| Pruebas | Vitest con Testing Library (unitarias), Playwright (interfaz y recorrido real) |
| Paquetes | pnpm, Node 24 |

## Pantallas

| Ruta | Muestra |
| --- | --- |
| `/login` | Correo y contraseña. Avisa si el usuario está desactivado |
| `/invitacion/:token` | Aceptar una invitación: nombre, `username` (no cambia) y contraseña. Abre la sesión al crear la cuenta |
| `/cuenta` | Mis tokens de API (crear, ver el token una sola vez, revocar), cambiar la contraseña y cerrar sesión. El dueño ve además Invitaciones (crear y copiar el enlace, lista con estado) y Usuarios (desactivar y reactivar) |
| `/` | Redirige a `/projects` |
| `/dashboard` | Indicadores calculados de proyectos, etapas, tareas y bugs |
| `/projects` | Catálogo de proyectos |
| `/projects/:projectId` | Descripción, indicadores, etapas y TL de la etapa activa |
| `/projects/:projectId/bugs` | Misma cabecera y panel del proyecto, con la lista de bugs |
| `/projects/:projectId/stages/:stageId` | Descripción y tabla de tareas |
| `?task=<id>` o `?bug=<id>` | Abre el detalle en un diálogo, con pestañas de comentarios e historial de status. Se conserva al recargar |
| `/dev/componentes` | Vitrina de componentes, solo con `pnpm dev` |

El avance de una etapa excluye las tareas canceladas. Los textos se muestran como texto plano, con sus saltos de línea.

## Estructura

```text
frontend/
├── src/
│   ├── api/          clientes por recurso sobre GET, POST y DELETE con timeout y validación; auth.ts para la cuenta
│   ├── realtime/     conexión WebSocket única y refresco por evento
│   ├── routes/       rutas y selección del detalle por URL
│   ├── shell/        marco de la app, migas, layout del proyecto, RequireAuth y la tarjeta de login
│   ├── views/        una vista por pantalla, más los detalles de tarea y bug
│   ├── components/   piezas reutilizables: tarjetas, avance, avatar, detalle, diálogo, estados
│   ├── domain/       status, tonos, fechas y resúmenes
│   ├── hooks/        useAsync
│   └── styles/       tokens, reset, breakpoints y estilos globales
├── config/apiTarget.ts   valida el destino del proxy y la base de pruebas
├── e2e/
│   ├── ui/           recorridos de interfaz con HTTP y WebSocket simulados, sin base
│   ├── harness/      backend y base de pruebas para el recorrido real
│   └── recorrido.real.spec.ts
└── dist/             compilado que sirve pmt-server, fuera de git
```

## Tiempo real

- Una sola conexión a `/api/events` por pestaña ([arquitectura.md](arquitectura.md#tiempo-real)), abierta por `RequireAuth` solo con sesión; el WebSocket se autentica con la cookie. No hay sondeo periódico.
- `invalidate` vuelve a consultar el recurso afectado; `resync` refresca todo lo visible.
- Si se cae, reintenta con esperas de 1, 2, 4… hasta 30 s. Después de 30 s sin `resync` los datos se marcan viejos.
- El encabezado lo indica: "En vivo", "Reconectando…" o "Sin conexión, datos de las HH:MM".

## Desarrollo y pruebas

El frontend en desarrollo **nunca apunta al 8080**: `vite.config.ts` y `playwright.config.ts` rechazan cualquier destino que no sea `127.0.0.1:8090` o `localhost:8090`, el backend de pruebas sobre la base `pmt_frontend_test`.

```bash
pnpm install
pnpm dev          # Vite en 5173, con proxy de /api y /api/events a PMT_API_TARGET (por omisión 127.0.0.1:8090)
pnpm build        # tsc -b y vite build → dist/
pnpm lint         # oxlint
pnpm test:run     # Vitest una vez; pnpm test en modo watch
pnpm test:ui      # Playwright con respuestas simuladas, sin base
pnpm e2e:install  # Chromium de Playwright dentro de node_modules
pnpm test:e2e     # Playwright contra 5173 y el backend de pruebas en 8090
```

**`pnpm test:e2e`** exige `PMT_TEST_DATABASE_URL` con la base `pmt_frontend_test`; si no, aborta antes de levantar nada. El harness:

1. Compila su propio `pmt-server` en `frontend/.harness/` y lo arranca con `CONFIG_FILE=e2e/harness/config.e2e.yaml` (127.0.0.1:8090, `frontend/dist`), la base en `DATABASE_CONNECTION_STRING`, tomada de `PMT_TEST_DATABASE_URL`, y el dueño de pruebas de `e2e/harness/owner.ts` en `OWNER_*`. No lee `backend/config/config.dev.yaml` ni toca `backend/bin` o el 8080. Espera a que `/readyz` responda.
2. El propio servidor aplica las migraciones que falten y crea el dueño de pruebas (`e2e-dueno`) si la base no tiene usuarios.
3. Entra con el dueño de pruebas, crea un proyecto centinela y lo lee por la API para confirmar que el 8090 usa la base de pruebas.
4. Abre sesión por el proxy de Vite y cada página del navegador arranca con esa cookie (`storageState`).
5. Crea sus datos con prefijo `e2e-<RUN_ID>-` y los borra al terminar, aunque falle, y comprueba por SQL que no queda nada.

`PMT_BROWSER_PATH` permite usar un Chromium del sistema. Los recorridos de `e2e/ui/` simulan también `GET /api/auth/me`.

**Pendiente:** `playwright.config.ts` todavía tiene un proyecto `reconexion` para `etapa5-tiempo-real.spec.ts`, una prueba del frontend anterior que ya no existe. Las pruebas E2E del frontend nuevo son la etapa 9 del proyecto PMT, en `stand_by`.

## Desplegar

```bash
cd frontend && pnpm build
```

`pmt-server` sirve `dist/` sin proxy aparte. Abre la carpeta al arrancar, así que lo seguro es reiniciarlo después de compilar. Ver [operacion.md](operacion.md).
