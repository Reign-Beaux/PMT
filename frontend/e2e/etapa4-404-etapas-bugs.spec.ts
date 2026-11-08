// Regresión QA de la tarea 05 de la etapa 4 (criterios 6-8 del plan f3c3dc30
// v1): listar etapas y bugs de un proyecto inexistente responde 404 en REST y
// error en MCP; proyecto existente sin datos responde 200 []; con datos, los
// mismos datos. API real en 8090 contra pmt_frontend_test. Siembra y limpia
// sus propios fixtures por SQL (prefijos qa-404- / a1b20000-, sin choques con
// otros specs). Si la instancia de pruebas no está levantada, se omite.
import { execFileSync } from 'node:child_process'
import { expect, request, test, type APIRequestContext } from '@playwright/test'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'
const TOKEN = 'qa-404-token'
const TOKEN_HASH = 'e382ca3eccede8c57eb2a904c44e1b5f725b5f7492c6244f6c38fbf7253d3d76'

const AGENTE = 'a1b20000-0000-4000-8000-0000000000a1'
const PROYECTO_VACIO = 'a1b20000-0000-4000-8000-0000000000e1'
const PROYECTO_DATOS = 'a1b20000-0000-4000-8000-0000000000d1'
const ETAPA = 'a1b20000-0000-4000-8000-000000000051'
const BUG = 'a1b20000-0000-4000-8000-0000000000b1'
const INEXISTENTE = '00000000-0000-4000-8000-000000000000'

function psql(sql: string): void {
  execFileSync(
    'docker',
    [
      'exec', '-i', 'postgresql',
      'psql', '-U', 'games_test', '-d', 'pmt_frontend_test',
      '-v', 'ON_ERROR_STOP=1', '-q', '-c', sql,
    ],
    { stdio: ['ignore', 'ignore', 'inherit'] },
  )
}

function limpiar(): void {
  psql(`DELETE FROM bugs WHERE project_id IN ('${PROYECTO_DATOS}', '${PROYECTO_VACIO}')`)
  psql(`DELETE FROM stages WHERE project_id IN ('${PROYECTO_DATOS}', '${PROYECTO_VACIO}')`)
  psql(`DELETE FROM projects WHERE id IN ('${PROYECTO_DATOS}', '${PROYECTO_VACIO}')`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-404-%'`)
}

async function mcp(
  api: APIRequestContext,
  tool: string,
  projectId: string,
): Promise<{ isError?: boolean; text: string; structured?: Record<string, unknown> }> {
  const respuesta = await api.post(`${API}/mcp`, {
    data: {
      jsonrpc: '2.0',
      id: 1,
      method: 'tools/call',
      params: { name: tool, arguments: { project_id: projectId } },
    },
  })
  expect(respuesta.status()).toBe(200)
  const cuerpo = await respuesta.json()
  const resultado = cuerpo.result
  return {
    isError: resultado?.isError,
    text: resultado?.content?.[0]?.text ?? '',
    structured: resultado?.structuredContent,
  }
}

let api: APIRequestContext
let apiDisponible = false

test.beforeAll(async () => {
  try {
    apiDisponible = (await fetch(`${API}/api/projects`)).ok
  } catch {
    apiDisponible = false
  }
  if (!apiDisponible) return

  limpiar()
  psql(
    `INSERT INTO agents (id, name, profile, token_hash) VALUES
      ('${AGENTE}', 'qa-404-agente', 'PP', '\\x${TOKEN_HASH}')`,
  )
  psql(
    `INSERT INTO projects (id, name, description, status, product_version) VALUES
      ('${PROYECTO_VACIO}', 'qa-404-proyecto-vacio', 'Sin etapas ni bugs', 'planning', ''),
      ('${PROYECTO_DATOS}', 'qa-404-proyecto-datos', 'Con etapa y bug', 'working', '9.9.9')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, description, position, status) VALUES
      ('${ETAPA}', '${PROYECTO_DATOS}', 'qa-404-etapa', 'Etapa de regresión', 1, 'planning')`,
  )
  psql(
    `INSERT INTO bugs (id, project_id, title, description, steps_to_reproduce, product_version, status, reported_by) VALUES
      ('${BUG}', '${PROYECTO_DATOS}', 'qa-404-bug', 'Bug de regresión', '1. paso', '9.9.9', 'reported', 'qa-404-agente')`,
  )
  api = await request.newContext({
    extraHTTPHeaders: { Authorization: `Bearer ${TOKEN}` },
  })
})

test.afterAll(async () => {
  if (!apiDisponible) return
  await api.dispose()
  limpiar()
})

test.beforeEach(() => {
  test.skip(!apiDisponible, 'Instancia de pruebas (8090/pmt_frontend_test) no disponible')
})

test.describe('criterio 6 — REST, proyecto inexistente', () => {
  for (const ruta of ['stages', 'bugs'] as const) {
    test(`GET /projects/{inexistente}/${ruta} responde 404 proyecto no encontrado`, async () => {
      const respuesta = await api.get(`${API}/api/projects/${INEXISTENTE}/${ruta}`)
      expect(respuesta.status()).toBe(404)
      expect(await respuesta.json()).toEqual({ error: 'proyecto no encontrado' })
    })
  }

  test('GET /projects/{uuid inválido}/stages responde 400', async () => {
    const respuesta = await api.get(`${API}/api/projects/no-es-uuid/stages`)
    expect(respuesta.status()).toBe(400)
    expect(await respuesta.json()).toEqual({ error: 'project_id inválido' })
  })
})

test.describe('criterio 7 — MCP, proyecto inexistente', () => {
  for (const tool of ['list_stages', 'list_bugs'] as const) {
    test(`${tool} con proyecto inexistente devuelve error de herramienta`, async () => {
      const resultado = await mcp(api, tool, INEXISTENTE)
      expect(resultado.isError).toBe(true)
      expect(resultado.text).toContain('proyecto no encontrado')
    })
  }
})

test.describe('criterio 8 — proyectos existentes', () => {
  test('proyecto sin etapas ni bugs responde 200 [] en ambas rutas', async () => {
    for (const ruta of ['stages', 'bugs'] as const) {
      const respuesta = await api.get(`${API}/api/projects/${PROYECTO_VACIO}/${ruta}`)
      expect(respuesta.status()).toBe(200)
      expect(await respuesta.json()).toEqual([])
    }
  })

  test('proyecto con datos responde los mismos datos en REST', async () => {
    const etapas = await api.get(`${API}/api/projects/${PROYECTO_DATOS}/stages`)
    expect(etapas.status()).toBe(200)
    const listaEtapas = await etapas.json()
    expect(listaEtapas.map((e: { id: string }) => e.id)).toEqual([ETAPA])
    expect(listaEtapas[0]).toMatchObject({
      id: ETAPA,
      project_id: PROYECTO_DATOS,
      name: 'qa-404-etapa',
      position: 1,
      status: 'planning',
    })

    const bugs = await api.get(`${API}/api/projects/${PROYECTO_DATOS}/bugs`)
    expect(bugs.status()).toBe(200)
    const listaBugs = await bugs.json()
    expect(listaBugs.map((b: { id: string }) => b.id)).toEqual([BUG])
    expect(listaBugs[0]).toMatchObject({
      id: BUG,
      project_id: PROYECTO_DATOS,
      title: 'qa-404-bug',
      status: 'reported',
      reported_by: 'qa-404-agente',
    })
  })

  test('proyecto existente devuelve las listas en MCP', async () => {
    const vacioEtapas = await mcp(api, 'list_stages', PROYECTO_VACIO)
    expect(vacioEtapas.isError).toBeUndefined()
    expect(vacioEtapas.structured).toEqual({ stages: [] })

    const vacioBugs = await mcp(api, 'list_bugs', PROYECTO_VACIO)
    expect(vacioBugs.isError).toBeUndefined()
    expect(vacioBugs.structured).toEqual({ bugs: [] })

    const conEtapas = await mcp(api, 'list_stages', PROYECTO_DATOS)
    const etapas = (conEtapas.structured?.stages ?? []) as { id: string; name: string }[]
    expect(etapas.map((e) => e.id)).toEqual([ETAPA])
    expect(etapas[0].name).toBe('qa-404-etapa')

    const conBugs = await mcp(api, 'list_bugs', PROYECTO_DATOS)
    const bugs = (conBugs.structured?.bugs ?? []) as { id: string; title: string }[]
    expect(bugs.map((b) => b.id)).toEqual([BUG])
    expect(bugs[0].title).toBe('qa-404-bug')
  })
})
