// Regresión QA de la tarea 03 de la etapa "Publicación de cambios y WebSocket"
// (criterios de QA del plan e69790ac v1): GET /api/events entrega resync al
// conectar, y un cambio hecho por REST y otro por MCP llegan como invalidate
// con el recurso correcto a dos clientes conectados a la vez.
// Backend de pruebas en 8090 contra pmt_frontend_test; si no está levantado se
// omite. Siembra y limpia sus propios datos (agente y proyectos con prefijo
// qa-ws1-ws-, reservados de este spec) por SQL con el psql del contenedor.
import { execFileSync } from 'node:child_process'
import { expect, request, test, type APIRequestContext, type Page } from '@playwright/test'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'
const WS_URL = `${API.replace(/^http/, 'ws')}/api/events`

// Agente PP con token conocido (sha256 de TOKEN) para autenticar REST y MCP.
const AGENTE_ID = 'b1e20000-0000-4000-8000-0000000000a1'
const AGENTE_NAME = 'qa-ws1-ws-pp'
const TOKEN = 'qa-ws1-token-ws'
const TOKEN_HASH = 'a7ad25cb5588befe1b699a6c3e101a457131c7ce135ed5126e7a3212790c0637'

function psql(sql: string): string {
  return execFileSync(
    'docker',
    [
      'exec', '-i', 'postgresql',
      'psql', '-U', 'games_test', '-d', 'pmt_frontend_test',
      '-v', 'ON_ERROR_STOP=1', '-qtA', '-c', sql,
    ],
    { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] },
  ).trim()
}

function limpiar(): void {
  // comments es inmutable: el trigger se desactiva solo durante el borrado.
  psql(
    `BEGIN;
     ALTER TABLE comments DISABLE TRIGGER comments_immutable;
     DELETE FROM comments WHERE actor_name = '${AGENTE_NAME}';
     ALTER TABLE comments ENABLE TRIGGER comments_immutable;
     COMMIT;`,
  )
  psql(`DELETE FROM projects WHERE name LIKE 'qa-ws1-ws-%'`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-ws1-ws-%'`)
}

let api: APIRequestContext
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
  psql(
    `INSERT INTO agents (id, name, profile, token_hash) VALUES ('${AGENTE_ID}', '${AGENTE_NAME}', 'PP', '\\x${TOKEN_HASH}')`,
  )
  api = await request.newContext({
    baseURL: API,
    extraHTTPHeaders: { Authorization: `Bearer ${TOKEN}` },
  })
})

test.afterAll(async () => {
  if (api) await api.dispose()
  if (disponible) limpiar()
})

test.beforeEach(() => {
  test.skip(!disponible, 'El backend de pruebas en 8090 no está levantado')
})

async function abrirCliente(page: Page, slot: number): Promise<void> {
  await page.evaluate(
    ({ url, idx }) => {
      const w = window as unknown as { __qaWs?: Record<number, string[]> }
      w.__qaWs = w.__qaWs ?? {}
      const mensajes: string[] = []
      w.__qaWs[idx] = mensajes
      const socket = new WebSocket(url)
      socket.onmessage = (evento) => {
        mensajes.push(String(evento.data))
      }
    },
    { url: WS_URL, idx: slot },
  )
}

function mensajesDe(page: Page, slot: number): Promise<string[]> {
  return page.evaluate((idx) => {
    const w = window as unknown as { __qaWs?: Record<number, string[]> }
    return w.__qaWs?.[idx] ?? []
  }, slot)
}

async function esperarFrame(page: Page, slot: number, indice: number): Promise<unknown> {
  await expect
    .poll(async () => (await mensajesDe(page, slot)).length, {
      timeout: 10_000,
      message: `frames recibidos por el cliente ${slot}`,
    })
    .toBeGreaterThan(indice)
  return JSON.parse((await mensajesDe(page, slot))[indice])
}

test('WS /api/events: resync al conectar e invalidate por REST y MCP a dos clientes', async ({ page }) => {
  await page.goto('/')

  await abrirCliente(page, 0)
  await abrirCliente(page, 1)

  // Primer frame de cada cliente: resync.
  expect(await esperarFrame(page, 0, 0)).toEqual({ type: 'resync' })
  expect(await esperarFrame(page, 1, 0)).toEqual({ type: 'resync' })

  // Cambio por REST: crear proyecto → invalidate resource=project en ambos.
  const creado = await api.post('/api/projects', {
    data: { name: 'qa-ws1-ws-rest', description: 'Proyecto de prueba WS por REST' },
  })
  expect(creado.ok()).toBeTruthy()
  const proyecto = (await creado.json()) as { id: string }

  for (const slot of [0, 1]) {
    expect(await esperarFrame(page, slot, 1)).toEqual({ type: 'invalidate', resource: 'project' })
  }

  // Cambio por MCP: add_comment sobre el proyecto → invalidate resource=comment.
  // El servidor MCP es sin estado: initialize y tools/call son POST independientes.
  const cabeceras = {
    'Content-Type': 'application/json',
    Accept: 'application/json, text/event-stream',
  }
  const inicio = await api.post('/mcp', {
    headers: cabeceras,
    data: {
      jsonrpc: '2.0',
      id: 1,
      method: 'initialize',
      params: {
        protocolVersion: '2025-03-26',
        capabilities: {},
        clientInfo: { name: 'qa-ws1-ws', version: '1' },
      },
    },
  })
  expect(inicio.ok()).toBeTruthy()
  const comentario = await api.post('/mcp', {
    headers: cabeceras,
    data: {
      jsonrpc: '2.0',
      id: 2,
      method: 'tools/call',
      params: {
        name: 'add_comment',
        arguments: {
          item_kind: 'project',
          item_id: proyecto.id,
          body: 'Comentario de prueba WS por MCP',
        },
      },
    },
  })
  expect(comentario.ok()).toBeTruthy()
  const cuerpo = (await comentario.json()) as { result?: { isError?: boolean } }
  expect(cuerpo.result?.isError ?? false).toBe(false)

  for (const slot of [0, 1]) {
    expect(await esperarFrame(page, slot, 2)).toEqual({ type: 'invalidate', resource: 'comment' })
  }
})
