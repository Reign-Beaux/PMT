// Regresión QA de la tarea 05 de la etapa "Tiempo real e integración"
// (criterio 8 del plan 1a56f403 v1): pmt-server sirve frontend/dist en su
// mismo puerto, sin nginx ni proxy aparte. Contra http://127.0.0.1:8090 (no
// el 5173 de Vite): la UI abre, una ruta profunda recargada carga (fallback
// SPA), el indicador muestra "En vivo" y un cambio hecho por la API (MCP
// real) se refleja sin recargar. Si el 8090 no sirve el compilado (falta
// `pnpm build`), el spec se salta con el motivo.
// Siembra y limpia sus propios datos (prefijo qa-r9- e ids c9c90000-,
// reservados de este spec) por SQL con el psql del contenedor.
import { execFileSync } from 'node:child_process'
import { expect, test, type Page } from '@playwright/test'

test.describe.configure({ mode: 'serial' })

const ORIGEN = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'
const PROJECT_ID = 'c9c90000-0000-4000-8000-000000000001'
const STAGE_ID = 'c9c90000-0000-4000-8000-000000000002'
const TASK_ID = 'c9c90000-0000-4000-8000-000000000003'
const PROJECT_NAME = 'qa-r9-Proyecto despliegue'
const STAGE_NAME = 'qa-r9-Etapa despliegue'
const TASK_NAME = 'qa-r9-Tarea despliegue'

const TOKEN_DEV = 'qa-r9-token-dev'

const AGENTES: { id: string; name: string; profile: string; token: string }[] = [
  { id: 'c9c90000-0000-4000-8000-0000000000a1', name: 'qa-r9-tl', profile: 'TL', token: 'qa-r9-token-tl' },
  { id: 'c9c90000-0000-4000-8000-0000000000a2', name: 'qa-r9-dev', profile: 'DevFrontend', token: TOKEN_DEV },
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
  const inicio = await fetch(`${ORIGEN}/mcp`, {
    method: 'POST',
    headers,
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'initialize',
      params: {
        protocolVersion: '2025-03-26',
        capabilities: {},
        clientInfo: { name: 'qa-r9', version: '1' },
      },
    }),
  })
  expect(inicio.ok, `initialize MCP con ${tool}`).toBeTruthy()
  const respuesta = await fetch(`${ORIGEN}/mcp`, {
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

function limpiar(): void {
  // comments y task_status_changes son inmutables: los triggers se desactivan
  // solo durante el borrado.
  psql(
    `BEGIN;
     ALTER TABLE comments DISABLE TRIGGER comments_immutable;
     ALTER TABLE task_status_changes DISABLE TRIGGER task_status_changes_immutable;
     ALTER TABLE task_changes DISABLE TRIGGER task_changes_immutable;
     DELETE FROM comments WHERE actor_name LIKE 'qa-r9-%';
     DELETE FROM task_status_changes WHERE actor_name LIKE 'qa-r9-%';
     DELETE FROM task_changes WHERE task_id IN (SELECT id FROM tasks WHERE title LIKE 'qa-r9-%');
     ALTER TABLE comments ENABLE TRIGGER comments_immutable;
     ALTER TABLE task_status_changes ENABLE TRIGGER task_status_changes_immutable;
     ALTER TABLE task_changes ENABLE TRIGGER task_changes_immutable;
     COMMIT;`,
  )
  psql(`DELETE FROM tasks WHERE title LIKE 'qa-r9-%'`)
  psql(`DELETE FROM stage_members WHERE stage_id = '${STAGE_ID}'`)
  psql(`DELETE FROM stages WHERE name LIKE 'qa-r9-%'`)
  psql(`DELETE FROM projects WHERE name LIKE 'qa-r9-%'`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-r9-%'`)
}

function sembrar(): void {
  const agentes = AGENTES.map(
    (a) => `('${a.id}', '${a.name}', '${a.profile}', '\\x${sha256(a.token)}')`,
  ).join(', ')
  psql(`INSERT INTO agents (id, name, profile, token_hash) VALUES ${agentes}`)
  psql(
    `INSERT INTO projects (id, name, description, status) VALUES ('${PROJECT_ID}', '${PROJECT_NAME}', 'Regresión etapa 5 tarea 05', 'working')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, position, status, lead) VALUES ('${STAGE_ID}', '${PROJECT_ID}', '${STAGE_NAME}', 1, 'in_dev', 'qa-r9-tl')`,
  )
  psql(
    `INSERT INTO stage_members (stage_id, agent, added_by) VALUES ('${STAGE_ID}', 'qa-r9-dev', 'qa-r9-tl')`,
  )
  psql(
    `INSERT INTO tasks (id, stage_id, title, description, status, assignee) VALUES ('${TASK_ID}', '${STAGE_ID}', '${TASK_NAME}', 'Tarea del spec de despliegue', 'in_dev', 'qa-r9-dev')`,
  )
}

let disponible = false
let motivoSkip = ''

test.beforeAll(async () => {
  try {
    const respuesta = await fetch(`${ORIGEN}/`)
    if (!respuesta.ok) {
      motivoSkip = `el 8090 respondió ${respuesta.status} en /`
      return
    }
    const cuerpo = await respuesta.text()
    if (!cuerpo.includes('<div id="root">')) {
      motivoSkip = 'el 8090 no sirve el compilado (falta pnpm build): ' +
        cuerpo.slice(0, 80)
      return
    }
    disponible = true
  } catch {
    motivoSkip = 'backend de pruebas no disponible en 8090'
  }
  if (!disponible) return
  limpiar()
  sembrar()
})

test.afterAll(() => {
  if (disponible) limpiar()
})

test.beforeEach(() => {
  test.skip(!disponible, motivoSkip || 'backend de pruebas no disponible en 8090')
})

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

function columna(page: Page, etiqueta: string) {
  return page
    .getByRole('region', { name: 'Tablero de la etapa' })
    .locator('section')
    .filter({ has: page.locator('h3', { hasText: etiqueta }) })
}

const RUTA_KANBAN = `${ORIGEN}/projects/${PROJECT_ID}/stages/${STAGE_ID}`

test.describe('Etapa 5 — UI servida por pmt-server en 8090', () => {
  test('la UI abre, la ruta profunda recargada carga, "En vivo" y refresco sin recargar', async ({
    page,
  }) => {
    // La raíz sirve la UI compilada.
    await page.goto(`${ORIGEN}/`)
    await expect(page.getByRole('button', { name: new RegExp(PROJECT_NAME) })).toBeVisible()

    // Ruta profunda recargada: el fallback SPA del servidor la carga.
    await page.goto(RUTA_KANBAN)
    await page.reload()
    const tarjeta = page.getByRole('button', { name: new RegExp(TASK_NAME) })
    await expect(tarjeta).toBeVisible()
    await expect(columna(page, 'En desarrollo')).toContainText(TASK_NAME)

    // El WebSocket de /api/events lo sirve el mismo proceso.
    await expect(page.getByRole('banner').getByRole('status')).toHaveText('En vivo', { timeout: 10_000 })
    await marcarSinRecarga(page)

    // Un cambio real por MCP se refleja sin recargar.
    await mcp(TOKEN_DEV, 'change_task_status', {
      task_id: TASK_ID,
      status: 'in_review',
      reason: 'regresión qa-r9 → in_review',
    })
    await expect(columna(page, 'En revisión')).toContainText(TASK_NAME, { timeout: 10_000 })
    await noHuboRecarga(page)
  })
})
