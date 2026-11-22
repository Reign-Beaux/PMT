# Frontend

Interfaz web para que el usuario siga los proyectos. **Sobre el trabajo de los proyectos solo consulta:** etapas, tareas y bugs los cambian los agentes por MCP. Las únicas escrituras son las de la cuenta (login, invitaciones, tokens, contraseña y usuarios) y, solo para el owner de un proyecto, administrar sus miembros y liberar etapas o bugs tomados. Van con la cookie y sin perfil declarado (perfil `User`); los errores del backend se muestran con su mensaje. En producción la sirve `pmt-server` en `http://127.0.0.1:8080`, y se actualiza sola en tiempo real.

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
| `/projects` | Catálogo de proyectos, con mi rol en cada uno. Si perdí el acceso a uno mientras lo veía, llego aquí con un aviso |
| `/projects/:projectId` | Descripción, mi rol, indicadores, etapas (con quién tiene tomada cada una y desde cuándo) y TL de la etapa activa |
| `/projects/:projectId/bugs` | Misma cabecera y panel del proyecto, con la lista de bugs y quién tiene tomado cada uno |
| `/projects/:projectId/members` | Miembros: `username`, nombre y rol, para todos. El owner agrega por `username` como `editor` o `viewer`, cambia el rol y quita (con confirmación) |
| `/projects/:projectId/stages/:stageId` | Descripción, quién la tiene tomada y desde cuándo, y tabla de tareas. El owner ve «Liberar» en una etapa tomada |
| `?task=<id>` o `?bug=<id>` | Abre el detalle en un diálogo, con pestañas de comentarios e historial de status. Se conserva al recargar. El de un bug dice quién lo tiene tomado y, al owner, le ofrece «Liberar» |
| `/dev/componentes` | Vitrina de componentes, solo con `pnpm dev` |

El avance de una etapa excluye las tareas canceladas. Los textos se muestran como texto plano, con sus saltos de línea.

**Rol y permisos en pantalla.** El rol llega en `role` de `GET /api/projects` y `GET /api/projects/{id}`. Solo el owner ve el formulario y las acciones de miembros y el botón «Liberar»; un editor o un viewer ven las mismas pantallas sin ellos. «Liberar» pide el motivo, obligatorio, que queda en el historial de status, y no cambia el status. Las reglas las aplica el backend: la interfaz solo oculta lo que no se puede hacer.

## Estructura

```text
frontend/
├── src/
│   ├── api/          clientes por recurso sobre GET, POST, PUT y DELETE con timeout y validación; auth.ts para la cuenta, members.ts para los miembros
│   ├── realtime/     conexión WebSocket única y refresco por evento
│   ├── routes/       rutas y selección del detalle por URL
│   ├── shell/        marco de la app, migas, layout del proyecto, RequireAuth y la tarjeta de login
│   ├── views/        una vista por pantalla, más los detalles de tarea y bug
│   ├── components/   piezas reutilizables: tarjetas, avance, avatar, detalle, diálogo, estados; members/ con el rol, «tomado por» y «Liberar»
│   ├── domain/       status, tonos, roles, fechas y resúmenes
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
- `invalidate` vuelve a consultar el recurso afectado; `resync` refresca todo lo visible. Un cambio de miembros llega como `project` y refresca el proyecto (con mi rol) y la lista de miembros; tomar y liberar llegan como `stage` o `bug`.
- **Pérdida de acceso.** Si el proyecto que estoy viendo responde 404 en un refresco (me quitaron o se eliminó), la pantalla vuelve a `/projects` con el aviso «Ya no tienes acceso al proyecto «…»». Un proyecto que nunca cargó sigue mostrando «Proyecto no encontrado».
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
3. Entra con el dueño de pruebas, crea un proyecto centinela y lo lee por la API para confirmar que el 8090 usa la base de pruebas. Los proyectos que el harness crea por SQL quedan con el dueño de pruebas como owner (`project_members`), porque sin membresía la API no los muestra.
4. Abre sesión por el proxy de Vite y cada página del navegador arranca con esa cookie (`storageState`).
5. Crea sus datos con prefijo `e2e-<RUN_ID>-` y los borra al terminar, aunque falle, y comprueba por SQL que no queda nada. `createStage` y `createBug` aceptan `takenBy` (un `username`) para dejarlos tomados, y `createUser` crea un segundo usuario como en la app: el dueño lo invita, él acepta por la API y la prueba recibe su sesión (`storageState`) para abrirla en otro contexto del navegador. Al terminar borra esos usuarios, sus invitaciones y los registros de status de sus etapas y bugs (liberar escribe uno).

`e2e/miembros.real.spec.ts` cubre la fase 1e: el owner comparte con un segundo usuario como viewer y como editor, éste ve el proyecto o deja de verlo en vivo, y el owner libera una etapa y un bug tomados con motivo.

`PMT_BROWSER_PATH` permite usar un Chromium del sistema. Los recorridos de `e2e/ui/` simulan también `GET /api/auth/me`.

**Pendiente:** `playwright.config.ts` todavía tiene un proyecto `reconexion` para `etapa5-tiempo-real.spec.ts`, una prueba del frontend anterior que ya no existe. Las pruebas E2E del frontend nuevo son la etapa 9 del proyecto PMT, en `stand_by`.

## Desplegar

```bash
cd frontend && pnpm build
```

`pmt-server` sirve `dist/` sin proxy aparte. Abre la carpeta al arrancar, así que lo seguro es reiniciarlo después de compilar. Ver [operacion.md](operacion.md).
