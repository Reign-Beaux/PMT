// Regresión QA de la tarea Backend 07 de la etapa "Tiempo real e integración"
// (criterios de QA 7 y 8 del plan 9351a74a v2): pmt-server sirve el frontend
// compilado. http://127.0.0.1:8090/ abre la UI en el navegador y recargar una
// ruta profunda de tarea carga la app con el detalle; la API responde igual
// con token (endpoint de lista) y el WebSocket /api/events entrega resync al
// conectar e invalidate al hacer un cambio por MCP, con la UI en "En vivo".
// Backend de pruebas en 8090 contra pmt_frontend_test sirviendo
// frontend/dist; si no está levantado se omite. Siembra y limpia sus propios
// datos (prefijo qa-r8- e ids c8c80000-, reservados de este spec) por SQL con
// el psql del contenedor.
import { execFileSync } from 'node:child_process'
import { expect, test } from '@playwright/test'

test.describe.configure({ mode: 'serial' })

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'
const PROJECT_ID = 'c8c80000-0000-4000-8000-000000000001'
const STAGE_ID = 'c8c80000-0000-4000-8000-000000000002'
const TASK_ID = 'c8c80000-0000-4000-8000-000000000003'
const PROJECT_NAME = 'qa-r8-Proyecto servido'
const STAGE_NAME = 'qa-r8-Etapa servida'
const TASK_NAME = 'qa-r8-Tarea servida'
const TASK_DESC = 'Tarea para la ruta profunda'
const RUTA_PROFUNDA = `${API}/projects/${PROJECT_ID}/stages/${STAGE_ID}?task=${TASK_ID}`

const TOKEN_PP = 'qa-r8-token-pp'

const AGENTES: { id: string; name: string; profile: string; token: string }[] = [
  { id: 'c8c80000-0000-4000-8000-0000000000a1', name: 'qa-r8-pp', profile: 'PP', token: TOKEN_PP },
  { id: 'c8c80000-0000-4000-8000-0000000000a2', name: 'qa-r8-dev', profile: 'DevFrontend', token: 'qa-r8-token-dev' },
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

async function mcp(tool: string, args: Record<string, unknown>): Promise<void> {
  const headers = {
    Authorization: `Bearer ${TOKEN_PP}`,
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
        clientInfo: { name: 'qa-r8', version: '1' },
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

function limpiar(): void {
  // comments es inmutable: el trigger se desactiva solo durante el borrado.
  psql(
    `BEGIN;
     ALTER TABLE comments DISABLE TRIGGER comments_immutable;
     DELETE FROM comments WHERE actor_name LIKE 'qa-r8-%';
     ALTER TABLE comments ENABLE TRIGGER comments_immutable;
     COMMIT;`,
  )
  psql(`DELETE FROM tasks WHERE title LIKE 'qa-r8-%'`)
  psql(`DELETE FROM stage_members WHERE stage_id = '${STAGE_ID}'`)
  psql(`DELETE FROM stages WHERE name LIKE 'qa-r8-%'`)
  psql(`DELETE FROM projects WHERE name LIKE 'qa-r8-%'`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-r8-%'`)
}

function sembrar(): void {
  const agentes = AGENTES.map(
    (a) => `('${a.id}', '${a.name}', '${a.profile}', '\\x${sha256(a.token)}')`,
  ).join(', ')
  psql(`INSERT INTO agents (id, name, profile, token_hash) VALUES ${agentes}`)
  psql(
    `INSERT INTO projects (id, name, description, status) VALUES ('${PROJECT_ID}', '${PROJECT_NAME}', 'Regresión etapa 5 Backend 07', 'working')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, position, status, lead) VALUES ('${STAGE_ID}', '${PROJECT_ID}', '${STAGE_NAME}', 1, 'in_dev', 'qa-r8-pp')`,
  )
  psql(
    `INSERT INTO stage_members (stage_id, agent, added_by) VALUES ('${STAGE_ID}', 'qa-r8-dev', 'qa-r8-pp')`,
  )
  psql(
    `INSERT INTO tasks (id, stage_id, title, description, status, assignee) VALUES ('${TASK_ID}', '${STAGE_ID}', '${TASK_NAME}', '${TASK_DESC}', 'in_dev', 'qa-r8-dev')`,
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

test.describe('Etapa 5 — pmt-server sirve el frontend compilado', () => {
  test('criterio 7: 8090/ abre la UI y recargar la ruta profunda de tarea carga el detalle', async ({
    page,
  }) => {
    // La UI la sirve pmt-server (no Vite): navegación absoluta a 8090.
    await page.goto(`${API}/`)
    await expect(page.getByRole('button', { name: new RegExp(PROJECT_NAME) })).toBeVisible()

    // Ruta profunda con query, cargada y luego recargada: el fallback SPA
    // responde index.html y la app abre el detalle de la tarea.
    await page.goto(RUTA_PROFUNDA)
    const dialogo = page.getByRole('dialog', { name: TASK_NAME })
    await expect(dialogo).toBeVisible()

    await page.reload()
    await expect(dialogo).toBeVisible()
    await expect(dialogo.getByText(TASK_DESC)).toBeVisible()
    await expect(dialogo.getByText('En desarrollo', { exact: true })).toBeVisible()
  })

  test('criterio 8: la API responde igual con token en un endpoint de lista', async ({
    request,
  }) => {
    const respuesta = await request.get(`${API}/api/projects`, {
      headers: { Authorization: `Bearer ${TOKEN_PP}` },
    })
    expect(respuesta.status()).toBe(200)
    expect(respuesta.headers()['content-type']).toContain('application/json')
    const proyectos = (await respuesta.json()) as { name: string }[]
    expect(proyectos.some((p) => p.name === PROJECT_NAME)).toBe(true)

    // Token inválido: 401 de la API, nunca index.html.
    const rechazada = await request.get(`${API}/api/projects`, {
      headers: { Authorization: 'Bearer qa-r8-token-malo' },
    })
    expect(rechazada.status()).toBe(401)
  })

  test('criterio 8: WS entrega resync e invalidate ante un cambio y la UI muestra "En vivo"', async ({
    page,
  }) => {
    const frames: string[] = []
    page.on('websocket', (ws) => {
      if (!ws.url().endsWith('/api/events')) return
      ws.on('framereceived', (frame) => {
        if (typeof frame.payload === 'string') frames.push(frame.payload)
      })
    })

    await page.goto(`${API}/projects/${PROJECT_ID}`)
    await expect(page.getByRole('banner').getByRole('status')).toHaveText('En vivo', { timeout: 10_000 })

    // Primer frame del WS de la app: resync.
    await expect
      .poll(() => frames.length, { timeout: 10_000, message: 'frames WS recibidos' })
      .toBeGreaterThan(0)
    expect(JSON.parse(frames[0])).toEqual({ type: 'resync' })

    // Cambio por MCP → invalidate resource=comment por el mismo WS.
    await mcp('add_comment', {
      item_kind: 'task',
      item_id: TASK_ID,
      body: 'comentario de regresión qa-r8',
    })
    await expect
      .poll(
        () => frames.some((f) => {
          const m = JSON.parse(f) as { type?: string; resource?: string }
          return m.type === 'invalidate' && m.resource === 'comment'
        }),
        { timeout: 10_000, message: 'invalidate comment por WS' },
      )
      .toBe(true)
    await expect(page.getByRole('banner').getByRole('status')).toHaveText('En vivo')
  })
})
