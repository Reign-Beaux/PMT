# Frontend

Interfaz web para que el usuario siga los proyectos. **Solo consulta:** usa únicamente `GET` de la API y no cambia nada; los cambios los hacen los agentes por MCP. En producción la sirve `pmt-server` en `http://127.0.0.1:8080`, y se actualiza sola en tiempo real.

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
│   ├── api/          clientes por recurso sobre un GET con timeout y validación de respuestas
│   ├── realtime/     conexión WebSocket única y refresco por evento
│   ├── routes/       rutas y selección del detalle por URL
│   ├── shell/        marco de la app, migas y layout del proyecto
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

- Una sola conexión a `/api/events` por pestaña ([arquitectura.md](arquitectura.md#tiempo-real)). No hay sondeo periódico.
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

1. Compila su propio `pmt-server` en `frontend/.harness/` y lo arranca con `CONFIG_FILE=e2e/harness/config.e2e.yaml` (127.0.0.1:8090, `frontend/dist`) y la base en `DATABASE_CONNECTION_STRING`, tomada de `PMT_TEST_DATABASE_URL`. No lee `backend/config/config.dev.yaml` ni toca `backend/bin` o el 8080. Espera a que `/readyz` responda.
2. El propio servidor aplica las migraciones que falten al arrancar.
3. Crea un proyecto centinela y lo lee por la API para confirmar que el 8090 usa la base de pruebas.
4. Crea sus datos con prefijo `e2e-<RUN_ID>-` y los borra al terminar, aunque falle, y comprueba por SQL que no queda nada.

`PMT_BROWSER_PATH` permite usar un Chromium del sistema.

**Pendiente:** `playwright.config.ts` todavía tiene un proyecto `reconexion` para `etapa5-tiempo-real.spec.ts`, una prueba del frontend anterior que ya no existe. Las pruebas E2E del frontend nuevo son la etapa 9 del proyecto PMT, en `stand_by`.

## Desplegar

```bash
cd frontend && pnpm build
```

`pmt-server` sirve `dist/` sin proxy aparte. Abre la carpeta al arrancar, así que lo seguro es reiniciarlo después de compilar. Ver [operacion.md](operacion.md).
