// Regresión QA de las tareas 12–14 (criterios 8, 10 y 12 del plan de0928aa v1):
// contrato de lectura de la API real (backend de pruebas en 8090, base
// pmt_frontend_test). Siembra y limpia sus propios fixtures por SQL con el
// cliente del proyecto (docker exec postgresql psql), como prevé la tarea 20.
// Si la instancia de pruebas no está levantada, los tests se omiten: el humo
// con harness aislado corresponde a la tarea 21.
import { execFileSync } from 'node:child_process'
import { expect, request, test, type APIRequestContext } from '@playwright/test'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'

const PROYECTO = 'cafe0000-0000-4000-8000-000000000001'
const ETAPA_CON_OPCIONALES = 'cafe0000-0000-4000-8000-000000000002'
const ETAPA_SIN_OPCIONALES = 'cafe0000-0000-4000-8000-000000000003'
const TAREA_CON_ASSIGNEE = 'cafe0000-0000-4000-8000-000000000004'
const TAREA_SIN_ASSIGNEE = 'cafe0000-0000-4000-8000-000000000005'
const BUG_DONE = 'cafe0000-0000-4000-8000-000000000006'
const BUG_REJECTED = 'cafe0000-0000-4000-8000-000000000007'
const BUG_REPORTED = 'cafe0000-0000-4000-8000-000000000008'
const INEXISTENTE = '00000000-0000-4000-8000-000000000000'

const RFC3339 = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?([+-]\d{2}:\d{2}|Z)$/

const url = (ruta: string): string => `${API}/api${ruta}`

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
  psql(`DELETE FROM bugs WHERE project_id = '${PROYECTO}'`)
  psql(`DELETE FROM tasks WHERE stage_id IN ('${ETAPA_CON_OPCIONALES}', '${ETAPA_SIN_OPCIONALES}')`)
  psql(`DELETE FROM stages WHERE project_id = '${PROYECTO}'`)
  psql(`DELETE FROM projects WHERE id = '${PROYECTO}'`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-e2e-%'`)
}

let api: APIRequestContext
let apiDisponible = false

test.beforeAll(async () => {
  try {
    const respuesta = await fetch(`${API}/api/projects`)
    apiDisponible = respuesta.ok
  } catch {
    apiDisponible = false
  }
  if (!apiDisponible) return

  limpiar()
  psql(
    `INSERT INTO agents (id, name, profile, token_hash) VALUES
      ('cafe0000-0000-4000-8000-0000000000a1', 'qa-e2e-tl', 'TL', '\\x1a'),
      ('cafe0000-0000-4000-8000-0000000000a2', 'qa-e2e-dev', 'DevFrontend', '\\x1b'),
      ('cafe0000-0000-4000-8000-0000000000a3', 'qa-e2e-dev2', 'DevBackend', '\\x1c')`,
  )
  psql(
    `INSERT INTO projects (id, name, description, status, product_version, created_at, updated_at) VALUES
      ('${PROYECTO}', 'Proyecto QA e2e', 'Fixture de regresión QA', 'working', '9.9.9',
       '2026-11-01 09:00:00-06', '2026-11-02 10:30:00-06')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, description, position, status, created_at, updated_at, lead, add_reason, add_requested_by) VALUES
      ('${ETAPA_CON_OPCIONALES}', '${PROYECTO}', 'Etapa e2e con opcionales', 'Con lead y miembros', 1, 'in_dev',
       '2026-11-01 09:05:00-06', '2026-11-03 11:00:00-06', 'qa-e2e-tl', 'Refuerzo e2e', 'qa-e2e-tl'),
      ('${ETAPA_SIN_OPCIONALES}', '${PROYECTO}', 'Etapa e2e sin opcionales', 'Sin lead ni miembros', 2, 'planning',
       '2026-11-01 09:06:00-06', '2026-11-01 09:06:00-06', NULL, NULL, NULL)`,
  )
  psql(
    `INSERT INTO stage_members (stage_id, agent, added_by) VALUES
      ('${ETAPA_CON_OPCIONALES}', 'qa-e2e-dev', 'qa-e2e-tl'),
      ('${ETAPA_CON_OPCIONALES}', 'qa-e2e-dev2', 'qa-e2e-tl')`,
  )
  psql(
    `INSERT INTO tasks (id, stage_id, title, description, status, created_at, updated_at, assignee) VALUES
      ('${TAREA_CON_ASSIGNEE}', '${ETAPA_CON_OPCIONALES}', 'Tarea e2e con assignee', 'Con dueño', 'in_dev',
       '2026-11-03 10:00:00-06', '2026-11-07 05:00:00-06', 'qa-e2e-dev'),
      ('${TAREA_SIN_ASSIGNEE}', '${ETAPA_CON_OPCIONALES}', 'Tarea e2e sin assignee', 'Sin dueño', 'ready',
       '2026-11-03 10:01:00-06', '2026-11-03 10:01:00-06', NULL)`,
  )
  psql(
    `INSERT INTO bugs (id, project_id, title, description, steps_to_reproduce, product_version, status, created_at, updated_at, reported_by) VALUES
      ('${BUG_DONE}', '${PROYECTO}', 'Bug e2e done', 'Corregido', '1. Abrir. 2. Ver.', '9.9.9', 'done',
       '2026-11-04 08:00:00-06', '2026-11-05 09:30:00-06', 'qa-e2e-reportador'),
      ('${BUG_REJECTED}', '${PROYECTO}', 'Bug e2e rejected', 'Rechazado', '1. Abrir.', '9.9.9', 'rejected',
       '2026-11-04 10:00:00-06', '2026-11-04 12:00:00-06', NULL),
      ('${BUG_REPORTED}', '${PROYECTO}', 'Bug e2e reported', 'Sin autor', '1. Pulsar.', '9.9.9', 'reported',
       '2026-11-06 08:00:00-06', '2026-11-06 08:00:00-06', NULL)`,
  )
  api = await request.newContext()
})

test.afterAll(async () => {
  if (!apiDisponible) return
  await api.dispose()
  limpiar()
})

test.beforeEach(() => {
  test.skip(!apiDisponible, 'Instancia de pruebas (8090/pmt_frontend_test) no disponible')
})

test.describe('criterio 8 — proyectos y etapas', () => {
  test('GET /projects devuelve los campos y tipos del fixture', async () => {
    const respuesta = await api.get(url('/projects'))
    expect(respuesta.status()).toBe(200)
    const proyectos = await respuesta.json()
    expect(Array.isArray(proyectos)).toBe(true)
    const proyecto = proyectos.find((p: { id: string }) => p.id === PROYECTO)
    expect(proyecto).toEqual({
      id: PROYECTO,
      name: 'Proyecto QA e2e',
      description: 'Fixture de regresión QA',
      status: 'working',
      product_version: '9.9.9',
      created_at: expect.stringMatching(RFC3339),
      updated_at: expect.stringMatching(RFC3339),
    })
  })

  test('GET /projects/{id} devuelve el proyecto; inexistente responde 404 con error', async () => {
    const respuesta = await api.get(url(`/projects/${PROYECTO}`))
    expect(respuesta.status()).toBe(200)
    const proyecto = await respuesta.json()
    expect(proyecto.id).toBe(PROYECTO)
    expect(proyecto.status).toBe('working')
    expect(proyecto.created_at).toMatch(RFC3339)

    const inexistente = await api.get(url(`/projects/${INEXISTENTE}`))
    expect(inexistente.status()).toBe(404)
    const cuerpo = await inexistente.json()
    expect(typeof cuerpo.error).toBe('string')
    expect(cuerpo.error.length).toBeGreaterThan(0)
  })

  test('GET /projects/{id}/stages devuelve etapas con y sin opcionales', async () => {
    const respuesta = await api.get(url(`/projects/${PROYECTO}/stages`))
    expect(respuesta.status()).toBe(200)
    const etapas = await respuesta.json()
    expect(etapas.map((e: { id: string }) => e.id)).toEqual([
      ETAPA_CON_OPCIONALES,
      ETAPA_SIN_OPCIONALES,
    ])

    const [conOpcionales, sinOpcionales] = etapas
    expect(conOpcionales).toMatchObject({
      id: ETAPA_CON_OPCIONALES,
      project_id: PROYECTO,
      name: 'Etapa e2e con opcionales',
      description: 'Con lead y miembros',
      position: 1,
      status: 'in_dev',
      lead: 'qa-e2e-tl',
      members: ['qa-e2e-dev', 'qa-e2e-dev2'],
      add_reason: 'Refuerzo e2e',
      add_requested_by: 'qa-e2e-tl',
    })
    expect(typeof conOpcionales.position).toBe('number')
    expect(conOpcionales.created_at).toMatch(RFC3339)
    expect(conOpcionales.updated_at).toMatch(RFC3339)

    expect(sinOpcionales).not.toHaveProperty('lead')
    expect(sinOpcionales).not.toHaveProperty('members')
    expect(sinOpcionales).not.toHaveProperty('add_reason')
    expect(sinOpcionales).not.toHaveProperty('add_requested_by')
    expect(sinOpcionales.position).toBe(2)
  })
})

test.describe('criterio 10 — tareas', () => {
  test('GET /stages/{id}/tasks devuelve tareas con y sin assignee', async () => {
    const respuesta = await api.get(url(`/stages/${ETAPA_CON_OPCIONALES}/tasks`))
    expect(respuesta.status()).toBe(200)
    const tareas = await respuesta.json()
    expect(tareas.map((t: { id: string }) => t.id)).toEqual([
      TAREA_CON_ASSIGNEE,
      TAREA_SIN_ASSIGNEE,
    ])

    const [conAssignee, sinAssignee] = tareas
    expect(conAssignee).toMatchObject({
      id: TAREA_CON_ASSIGNEE,
      stage_id: ETAPA_CON_OPCIONALES,
      title: 'Tarea e2e con assignee',
      description: 'Con dueño',
      status: 'in_dev',
      assignee: 'qa-e2e-dev',
    })
    expect(conAssignee.created_at).toMatch(RFC3339)

    expect(sinAssignee).not.toHaveProperty('assignee')
    expect(sinAssignee.status).toBe('ready')
  })

  test('GET /tasks/{id} devuelve la tarea; inexistente responde 404', async () => {
    const respuesta = await api.get(url(`/tasks/${TAREA_CON_ASSIGNEE}`))
    expect(respuesta.status()).toBe(200)
    const tarea = await respuesta.json()
    expect(tarea).toMatchObject({
      id: TAREA_CON_ASSIGNEE,
      stage_id: ETAPA_CON_OPCIONALES,
      title: 'Tarea e2e con assignee',
      status: 'in_dev',
      assignee: 'qa-e2e-dev',
    })

    const inexistente = await api.get(url(`/tasks/${INEXISTENTE}`))
    expect(inexistente.status()).toBe(404)
    const cuerpo = await inexistente.json()
    expect(typeof cuerpo.error).toBe('string')
  })
})

test.describe('criterio 12 — bugs', () => {
  test('GET /projects/{id}/bugs incluye done y rejected con los campos del fixture', async () => {
    const respuesta = await api.get(url(`/projects/${PROYECTO}/bugs`))
    expect(respuesta.status()).toBe(200)
    const bugs = await respuesta.json()
    expect(bugs.map((b: { id: string }) => b.id)).toEqual([BUG_DONE, BUG_REJECTED, BUG_REPORTED])

    const [done, rejected, reported] = bugs
    expect(done).toMatchObject({
      id: BUG_DONE,
      project_id: PROYECTO,
      title: 'Bug e2e done',
      description: 'Corregido',
      steps_to_reproduce: '1. Abrir. 2. Ver.',
      product_version: '9.9.9',
      status: 'done',
      reported_by: 'qa-e2e-reportador',
    })
    expect(done.created_at).toMatch(RFC3339)
    expect(done.updated_at).toMatch(RFC3339)

    expect(rejected.status).toBe('rejected')
    expect(rejected).not.toHaveProperty('reported_by')
    expect(reported.status).toBe('reported')
    expect(reported).not.toHaveProperty('reported_by')
  })
})
