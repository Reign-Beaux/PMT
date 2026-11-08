// Regresión QA de la tarea 03 de la etapa "Tiempo real e integración"
// (criterios de QA del plan 1b3d34e7 v1): con la UI abierta y contra la API
// real de 8090, cambiar el estado de una tarea por API mueve la tarjeta y
// actualiza el avance sin recargar; crear un bug lo muestra en su columna;
// un comentario aparece en el detalle abierto; el cambio de estado aparece
// en el historial; retirar el elemento abierto muestra "Ya no existe"; un
// fallo de red durante el refresco muestra el ErrorState sin perder los
// datos (el backend detenido también corta el WS y no dispara refrescos, así
// que el fallo se simula abortando las peticiones REST, que es lo que la
// vista percibe); a 360 px no hay overflow, tampoco con el ErrorState.
// Backend de pruebas en 8090 contra pmt_frontend_test; si no está levantado
// se omite. Siembra y limpia sus propios datos (prefijo qa-r7- e ids
// c7c70000-, reservados de este spec) por SQL con el psql del contenedor.
// Las escrituras van por MCP (change_task_status, add_comment, create_bug,
// retire_task) con agentes propios: ejercitan la cadena real
// REST/MCP → NOTIFY → WS → refresco.
import { execFileSync } from 'node:child_process'
import { expect, test, type Page } from '@playwright/test'

test.describe.configure({ mode: 'serial' })

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'
const PROJECT_ID = 'c7c70000-0000-4000-8000-000000000001'
const STAGE_ID = 'c7c70000-0000-4000-8000-000000000002'
const TASK_KANBAN = 'c7c70000-0000-4000-8000-000000000003'
const TASK_DETALLE = 'c7c70000-0000-4000-8000-000000000004'
const PROJECT_NAME = 'qa-r7-Proyecto refresco'
const STAGE_NAME = 'qa-r7-Etapa refresco'
const TASK_KANBAN_NAME = 'qa-r7-Tarea kanban'
const TASK_DETALLE_NAME = 'qa-r7-Tarea detalle'

const TOKENS = {
  tl: 'qa-r7-token-tl',
  dev: 'qa-r7-token-dev',
  pp: 'qa-r7-token-pp',
  cr: 'qa-r7-token-cr',
  qa: 'qa-r7-token-qa',
} as const

const AGENTES: { id: string; name: string; profile: string; token: string }[] = [
  { id: 'c7c70000-0000-4000-8000-0000000000a1', name: 'qa-r7-tl', profile: 'TL', token: TOKENS.tl },
  { id: 'c7c70000-0000-4000-8000-0000000000a2', name: 'qa-r7-dev', profile: 'DevFrontend', token: TOKENS.dev },
  { id: 'c7c70000-0000-4000-8000-0000000000a3', name: 'qa-r7-pp', profile: 'PP', token: TOKENS.pp },
  { id: 'c7c70000-0000-4000-8000-0000000000a4', name: 'qa-r7-cr', profile: 'CR', token: TOKENS.cr },
  { id: 'c7c70000-0000-4000-8000-0000000000a5', name: 'qa-r7-qa', profile: 'QA', token: TOKENS.qa },
]

function psql(sql: string): void {
  execFileSync(
    'docker',
    [
      'exec', '-i', 'postgresql',
      'psql', '-U', 'games_test', '-d', 'pmt_frontend_test',
      '-v', 'ON_ERROR_STOP=1', '-qtA', '-c', sql,
    ],
    { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] },
  )
}

function sha256(value: string): string {
  return execFileSync('openssl', ['dgst', '-sha256', '-r'], {
    input: value,
    encoding: 'utf8',
  }).split(' ')[0]
}

async function mcp(
  token: string,
  tool: string,
  args: Record<string, unknown>,
): Promise<void> {
  const headers = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    Accept: 'application/json, text/event-stream',
  }
  const inicio = await fetch(`${API}/mcp`, {
    method: 'POST',
    headers,
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'initialize',
      params: {
        protocolVersion: '2025-03-26',
        capabilities: {},
        clientInfo: { name: 'qa-r7', version: '1' },
      },
    }),
  })
  expect(inicio.ok, `initialize MCP con ${tool}`).toBeTruthy()
  const respuesta = await fetch(`${API}/mcp`, {
    method: 'POST',
    headers,
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 2,
      method: 'tools/call',
      params: { name: tool, arguments: args },
    }),
  })
  expect(respuesta.ok, `${tool} HTTP`).toBeTruthy()
  const cuerpo = (await respuesta.json()) as { result?: { isError?: boolean; content?: { text?: string }[] } }
  expect(cuerpo.result, `${tool}: respuesta sin result`).toBeDefined()
  expect(cuerpo.result?.isError ?? false, `${tool}: ${cuerpo.result?.content?.[0]?.text ?? 'sin resultado'}`).toBe(false)
}

function cambiarEstado(
  token: string,
  taskId: string,
  status: string,
): Promise<void> {
  return mcp(token, 'change_task_status', {
    task_id: taskId,
    status,
    reason: `regresión qa-r7 → ${status}`,
  })
}

function limpiar(): void {
  // comments, task_status_changes, bug_status_changes y task_changes son
  // inmutables: los triggers se desactivan solo durante el borrado.
  psql(
    `BEGIN;
     ALTER TABLE comments DISABLE TRIGGER comments_immutable;
     ALTER TABLE task_status_changes DISABLE TRIGGER task_status_changes_immutable;
     ALTER TABLE bug_status_changes DISABLE TRIGGER bug_status_changes_immutable;
     ALTER TABLE task_changes DISABLE TRIGGER task_changes_immutable;
     DELETE FROM comments WHERE actor_name LIKE 'qa-r7-%';
     DELETE FROM task_status_changes WHERE actor_name LIKE 'qa-r7-%';
     DELETE FROM bug_status_changes WHERE actor_name LIKE 'qa-r7-%';
     DELETE FROM task_changes WHERE task_id IN (SELECT id FROM tasks WHERE title LIKE 'qa-r7-%');
     ALTER TABLE comments ENABLE TRIGGER comments_immutable;
     ALTER TABLE task_status_changes ENABLE TRIGGER task_status_changes_immutable;
     ALTER TABLE bug_status_changes ENABLE TRIGGER bug_status_changes_immutable;
     ALTER TABLE task_changes ENABLE TRIGGER task_changes_immutable;
     COMMIT;`,
  )
  psql(`DELETE FROM bugs WHERE title LIKE 'qa-r7-%'`)
  psql(`DELETE FROM tasks WHERE title LIKE 'qa-r7-%'`)
  psql(`DELETE FROM stage_members WHERE stage_id = '${STAGE_ID}'`)
  psql(`DELETE FROM stages WHERE name LIKE 'qa-r7-%'`)
  psql(`DELETE FROM projects WHERE name LIKE 'qa-r7-%'`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-r7-%'`)
}

function sembrar(): void {
  const agentes = AGENTES.map(
    (a) => `('${a.id}', '${a.name}', '${a.profile}', '\\x${sha256(a.token)}')`,
  ).join(', ')
  psql(`INSERT INTO agents (id, name, profile, token_hash) VALUES ${agentes}`)
  psql(
    `INSERT INTO projects (id, name, description, status) VALUES ('${PROJECT_ID}', '${PROJECT_NAME}', 'Regresión etapa 5 tarea 03', 'working')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, position, status, lead) VALUES ('${STAGE_ID}', '${PROJECT_ID}', '${STAGE_NAME}', 1, 'in_dev', 'qa-r7-tl')`,
  )
  psql(
    `INSERT INTO stage_members (stage_id, agent, added_by) VALUES ('${STAGE_ID}', 'qa-r7-dev', 'qa-r7-tl')`,
  )
  psql(
    `INSERT INTO tasks (id, stage_id, title, description, status, assignee) VALUES
       ('${TASK_KANBAN}', '${STAGE_ID}', '${TASK_KANBAN_NAME}', 'Tarea para mover en el kanban', 'in_dev', 'qa-r7-dev'),
       ('${TASK_DETALLE}', '${STAGE_ID}', '${TASK_DETALLE_NAME}', 'Tarea para el diálogo de detalle', 'in_review', 'qa-r7-dev')`,
  )
}

let disponible = false

test.beforeAll(async () => {
  try {
    const respuesta = await fetch(`${API}/api/projects`)
    disponible = respuesta.ok
  } catch {
    disponible = false
  }
  if (!disponible) return
  limpiar()
  sembrar()
})

test.afterAll(() => {
  if (disponible) limpiar()
})

test.beforeEach(() => {
  test.skip(!disponible, 'backend de pruebas no disponible en 8090')
})

/** Columna del kanban cuyo encabezado empieza por la etiqueta dada. */
function columna(page: Page, tablero: string, etiqueta: string) {
  return page
    .getByRole('region', { name: tablero })
    .locator('section')
    .filter({ has: page.locator('h3', { hasText: etiqueta }) })
}

async function marcarSinRecarga(page: Page): Promise<void> {
  await page.evaluate(() => {
    ;(window as unknown as { __qaSinRecarga?: boolean }).__qaSinRecarga = true
  })
}

async function noHuboRecarga(page: Page): Promise<void> {
  expect(
    await page.evaluate(() => (window as unknown as { __qaSinRecarga?: boolean }).__qaSinRecarga === true),
    'la página no debe haberse recargado',
  ).toBe(true)
}

const RUTA_KANBAN = `/projects/${PROJECT_ID}/stages/${STAGE_ID}`

test.describe('Etapa 5 — refresco de vistas por evento', () => {
  test('cambiar el estado por API mueve la tarjeta y actualiza el avance sin recargar', async ({
    page,
    browser,
  }) => {
    const proyecto = await browser.newPage()
    try {
      await proyecto.goto(`/projects/${PROJECT_ID}`)
      const avance = proyecto.getByRole('progressbar', { name: 'Avance de la etapa' })
      await expect(avance).toHaveAttribute('aria-valuenow', '0')

      await page.goto(RUTA_KANBAN)
      const tarjeta = page.getByRole('button', { name: new RegExp(TASK_KANBAN_NAME) })
      await expect(tarjeta).toBeVisible()
      await expect(columna(page, 'Tablero de la etapa', 'En desarrollo')).toContainText(TASK_KANBAN_NAME)
      await marcarSinRecarga(page)
      await marcarSinRecarga(proyecto)

      // La cadena completa de transiciones por MCP: dev → CR → QA → TL.
      await cambiarEstado(TOKENS.dev, TASK_KANBAN, 'in_review')
      await expect(columna(page, 'Tablero de la etapa', 'En revisión')).toContainText(TASK_KANBAN_NAME, { timeout: 10_000 })

      await cambiarEstado(TOKENS.cr, TASK_KANBAN, 'in_qa')
      await expect(columna(page, 'Tablero de la etapa', 'En QA')).toContainText(TASK_KANBAN_NAME, { timeout: 10_000 })

      await cambiarEstado(TOKENS.qa, TASK_KANBAN, 'pending_close')
      await expect(columna(page, 'Tablero de la etapa', 'Pendiente de cierre')).toContainText(TASK_KANBAN_NAME, { timeout: 10_000 })

      await cambiarEstado(TOKENS.tl, TASK_KANBAN, 'done')
      await expect(columna(page, 'Tablero de la etapa', 'Terminada')).toContainText(TASK_KANBAN_NAME, { timeout: 10_000 })

      // El avance de la lista de etapas se actualiza solo: 1 de 2 tareas.
      await expect(avance).toHaveAttribute('aria-valuenow', '50', { timeout: 10_000 })
      await expect(proyecto.getByText('50 % · 1/2')).toBeVisible()

      await noHuboRecarga(page)
      await noHuboRecarga(proyecto)
    } finally {
      await proyecto.close()
    }
  })

  test('crear un bug por API lo muestra en su columna sin recargar', async ({ page }) => {
    await page.goto(`/projects/${PROJECT_ID}/bugs`)
    await expect(page.getByText('Este proyecto no tiene bugs')).toBeVisible()
    await marcarSinRecarga(page)

    await mcp(TOKENS.pp, 'create_bug', {
      project_id: PROJECT_ID,
      title: 'qa-r7-Bug de prueba',
      description: 'Bug creado por la regresión',
      steps_to_reproduce: 'Abrir el tablero de bugs',
      product_version: '1.0.0',
    })

    await expect(columna(page, 'Tablero de bugs', 'Reportado')).toContainText('qa-r7-Bug de prueba', { timeout: 10_000 })
    await noHuboRecarga(page)
  })

  test('comentario y cambio de estado refrescan el detalle abierto', async ({ page }) => {
    await page.goto(`${RUTA_KANBAN}?task=${TASK_DETALLE}`)
    const dialogo = page.getByRole('dialog', { name: TASK_DETALLE_NAME })
    await expect(dialogo).toBeVisible()
    await expect(dialogo.getByText('En revisión', { exact: true })).toBeVisible()
    await marcarSinRecarga(page)

    await mcp(TOKENS.dev, 'add_comment', {
      item_kind: 'task',
      item_id: TASK_DETALLE,
      body: 'comentario visible qa-r7',
    })
    await expect(dialogo.getByText('comentario visible qa-r7')).toBeVisible({ timeout: 10_000 })

    await cambiarEstado(TOKENS.cr, TASK_DETALLE, 'in_qa')
    await expect(dialogo.getByText('En QA', { exact: true })).toBeVisible({ timeout: 10_000 })
    const historial = dialogo.locator('section[aria-label="Historial de estados"]')
    await expect(historial.getByText('En revisión → En QA')).toBeVisible()
    await expect(historial.getByText('qa-r7-cr (CR)')).toBeVisible()
    await noHuboRecarga(page)
  })

  test('retirar la tarea abierta muestra "Ya no existe" en el detalle', async ({ page }) => {
    await page.goto(`${RUTA_KANBAN}?task=${TASK_DETALLE}`)
    const dialogo = page.getByRole('dialog', { name: TASK_DETALLE_NAME })
    await expect(dialogo).toBeVisible()
    await expect(dialogo.getByText('En QA', { exact: true })).toBeVisible()

    await mcp(TOKENS.pp, 'retire_task', {
      task_id: TASK_DETALLE,
      reason: 'Regresión qa-r7: retirar la tarea abierta',
      requested_by: 'qa-r7-pp',
    })

    await expect(dialogo.getByText('Ya no existe')).toBeVisible({ timeout: 10_000 })
    // El diálogo conserva el título previo y se cierra normalmente.
    await expect(dialogo).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(dialogo).toBeHidden()
  })

  test('fallo de red en el refresco: ErrorState con "Reintentar" sin perder los datos, a 360 px', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto(RUTA_KANBAN)
    await expect(page.getByRole('button', { name: new RegExp(TASK_KANBAN_NAME) })).toBeVisible()
    // El resync que llega al conectar el WS refresca todas las vistas ~300 ms
    // después: hay que dejarlo terminar antes de abortar las consultas.
    await page.waitForLoadState('networkidle')
    await marcarSinRecarga(page)

    // El backend "detenido" para las consultas REST: se abortan todas las
    // peticiones /api/** (el WS de /api/events no pasa por page.route y sigue
    // vivo, que es justo lo que dispara el refresco).
    await page.route('**/api/**', (route) => route.abort())

    // Cambio directo en la base: el trigger publica task → invalidate → refresco.
    psql(`UPDATE tasks SET updated_at = now() WHERE id = '${TASK_KANBAN}'`)

    // El ErrorState de la vista de etapa, acotado a su sección: con specs en
    // paralelo un invalidate 'project' ajeno puede poner otro ErrorState en el
    // encabezado, que no es el que este criterio verifica.
    const vista = page
      .locator('section')
      .filter({ has: page.getByRole('region', { name: 'Tablero de la etapa' }) })
    await expect(vista.getByRole('button', { name: 'Reintentar' })).toBeVisible({ timeout: 10_000 })
    // Los datos previos siguen en pantalla junto al error.
    await expect(page.getByRole('button', { name: new RegExp(TASK_KANBAN_NAME) })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(360)
    await noHuboRecarga(page)
  })

  test('sin overflow horizontal a 360 px en las vistas principales', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    const sinOverflow = async () => {
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(360)
    }

    await page.goto('/')
    await expect(page.getByRole('button', { name: new RegExp(PROJECT_NAME) })).toBeVisible()
    await sinOverflow()

    await page.goto(`/projects/${PROJECT_ID}`)
    await expect(page.getByRole('progressbar', { name: 'Avance de la etapa' })).toBeVisible()
    await sinOverflow()

    await page.goto(RUTA_KANBAN)
    await expect(page.getByRole('button', { name: new RegExp(TASK_KANBAN_NAME) })).toBeVisible()
    await sinOverflow()

    await page.goto(`${RUTA_KANBAN}?task=${TASK_KANBAN}`)
    await expect(page.getByRole('dialog', { name: TASK_KANBAN_NAME })).toBeVisible()
    await sinOverflow()

    await page.goto(`/projects/${PROJECT_ID}/bugs`)
    await expect(page.getByText('qa-r7-Bug de prueba')).toBeVisible()
    await sinOverflow()
  })
})
