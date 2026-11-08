// Regresión QA de las tareas 15–17 (criterios 3, 6 y 9 del plan 06975790 v1):
// contrato real de avance, comentarios e historial de estados (backend de
// pruebas en 8090, base pmt_frontend_test). Siembra y limpia sus propios
// fixtures por SQL con el cliente del proyecto (docker exec postgresql psql).
// Las tablas de cambios son inmutables: la limpieza desactiva sus triggers solo
// durante el borrado de las filas del fixture y los reactiva en la misma
// transacción. Si la instancia de pruebas no está levantada, los tests se omiten.
import { execFileSync } from 'node:child_process'
import { expect, request, test, type APIRequestContext } from '@playwright/test'
import { BUG_STATUS_LABELS, TASK_STATUS_LABELS } from '../src/domain/status'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'

const PROYECTO = 'cafe1000-0000-4000-8000-000000000001'
const ETAPA_EN_CURSO = 'cafe1000-0000-4000-8000-000000000002'
const ETAPA_EN_ESPERA = 'cafe1000-0000-4000-8000-000000000003'
const TAREA_LISTA = 'cafe1000-0000-4000-8000-000000000004'
const TAREA_TERMINADA = 'cafe1000-0000-4000-8000-000000000005'
const TAREA_SIN_HISTORIAL = 'cafe1000-0000-4000-8000-000000000006'
const BUG_ABIERTO = 'cafe1000-0000-4000-8000-000000000007'
const BUG_CERRADO = 'cafe1000-0000-4000-8000-000000000008'
const INEXISTENTE = '00000000-0000-4000-8000-000000000000'

const RFC3339 = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?([+-]\d{2}:\d{2}|Z)$/
// Los estados que el frontend acepta (tarea 06): el historial real no puede salirse de ellos.
const ESTADOS_TAREA = Object.keys(TASK_STATUS_LABELS)
const ESTADOS_BUG = Object.keys(BUG_STATUS_LABELS)

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

const TABLAS_INMUTABLES = [
  'comments',
  'task_status_changes',
  'bug_status_changes',
  'stage_status_changes',
  'project_status_changes',
]

function inmutabilidad(modo: 'DISABLE' | 'ENABLE'): string {
  return TABLAS_INMUTABLES.map(
    (tabla) => `ALTER TABLE ${tabla} ${modo} TRIGGER ${tabla}_immutable;`,
  ).join('\n')
}

// Borra solo las filas del fixture: las tablas de cambios no tienen clave foránea.
function limpiar(): void {
  psql(`
BEGIN;
${inmutabilidad('DISABLE')}
DELETE FROM comments WHERE item_id IN ('${TAREA_LISTA}', '${TAREA_TERMINADA}', '${TAREA_SIN_HISTORIAL}', '${BUG_ABIERTO}', '${BUG_CERRADO}', '${PROYECTO}', '${ETAPA_EN_CURSO}');
DELETE FROM task_status_changes WHERE task_id IN ('${TAREA_LISTA}', '${TAREA_TERMINADA}', '${TAREA_SIN_HISTORIAL}');
DELETE FROM bug_status_changes WHERE bug_id IN ('${BUG_ABIERTO}', '${BUG_CERRADO}');
DELETE FROM stage_status_changes WHERE stage_id IN ('${ETAPA_EN_CURSO}', '${ETAPA_EN_ESPERA}');
DELETE FROM project_status_changes WHERE project_id = '${PROYECTO}';
${inmutabilidad('ENABLE')}
COMMIT;`)
  psql(`DELETE FROM bugs WHERE project_id = '${PROYECTO}'`)
  psql(
    `DELETE FROM tasks WHERE stage_id IN ('${ETAPA_EN_CURSO}', '${ETAPA_EN_ESPERA}', '${PROYECTO}')`,
  )
  psql(`DELETE FROM stage_members WHERE stage_id IN ('${ETAPA_EN_CURSO}', '${ETAPA_EN_ESPERA}')`)
  psql(`DELETE FROM stages WHERE project_id = '${PROYECTO}'`)
  psql(`DELETE FROM projects WHERE id = '${PROYECTO}'`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-e2e2-%'`)
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
      ('cafe1000-0000-4000-8000-0000000000a1', 'qa-e2e2-tl', 'TL', '\\x2a'),
      ('cafe1000-0000-4000-8000-0000000000a2', 'qa-e2e2-dev', 'DevFrontend', '\\x2b'),
      ('cafe1000-0000-4000-8000-0000000000a3', 'qa-e2e2-qa', 'QA', '\\x2c')`,
  )
  psql(
    `INSERT INTO projects (id, name, description, status, product_version, created_at, updated_at) VALUES
      ('${PROYECTO}', 'Proyecto QA avance', 'Fixture de regresión QA', 'working', '9.9.9',
       '2026-11-01 09:00:00-06', '2026-11-06 08:00:00-06')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, description, position, status, created_at, updated_at, lead) VALUES
      ('${ETAPA_EN_CURSO}', '${PROYECTO}', 'Etapa en curso', 'Con lead y miembros', 1, 'in_dev',
       '2026-11-01 09:05:00-06', '2026-11-05 11:00:00-06', 'qa-e2e2-tl'),
      ('${ETAPA_EN_ESPERA}', '${PROYECTO}', 'Etapa en espera', 'Sin lead ni miembros ni tareas', 2, 'stand_by',
       '2026-11-01 09:06:00-06', '2026-11-01 09:06:00-06', NULL)`,
  )
  psql(
    `INSERT INTO stage_members (stage_id, agent, added_by) VALUES
      ('${ETAPA_EN_CURSO}', 'qa-e2e2-dev', 'qa-e2e2-tl'),
      ('${ETAPA_EN_CURSO}', 'qa-e2e2-qa', 'qa-e2e2-tl')`,
  )
  psql(
    `INSERT INTO tasks (id, stage_id, title, description, status, created_at, updated_at, assignee) VALUES
      ('${TAREA_LISTA}', '${ETAPA_EN_CURSO}', 'Tarea e2e lista', 'Sin dueño', 'ready',
       '2026-11-03 10:00:00-06', '2026-11-04 10:00:00-06', NULL),
      ('${TAREA_TERMINADA}', '${ETAPA_EN_CURSO}', 'Tarea e2e terminada', 'Con dueño', 'done',
       '2026-11-03 10:01:00-06', '2026-11-07 05:00:00-06', 'qa-e2e2-dev'),
      ('${TAREA_SIN_HISTORIAL}', '${ETAPA_EN_CURSO}', 'Tarea e2e sin historial', 'Sin movimientos', 'backlog',
       '2026-11-03 10:02:00-06', '2026-11-03 10:02:00-06', NULL)`,
  )
  psql(
    `INSERT INTO bugs (id, project_id, title, description, steps_to_reproduce, product_version, status, created_at, updated_at) VALUES
      ('${BUG_ABIERTO}', '${PROYECTO}', 'Bug e2e abierto', 'Confirmado', '1. Abrir.', '9.9.9', 'confirmed',
       '2026-11-05 08:00:00-06', '2026-11-06 09:00:00-06'),
      ('${BUG_CERRADO}', '${PROYECTO}', 'Bug e2e cerrado', 'Corregido', '1. Pulsar.', '9.9.9', 'done',
       '2026-11-04 08:00:00-06', '2026-11-04 09:00:00-06')`,
  )
  psql(
    `INSERT INTO comments (id, item_kind, item_id, body, actor_name, actor_profile, created_at) VALUES
      ('cafe1000-0000-4000-8000-0000000000b1', 'task', '${TAREA_TERMINADA}',
       'Evidencia: <b>texto</b> con & y salto.\nSegunda línea.', 'qa-e2e2-dev', 'DevFrontend',
       '2026-11-06 10:00:00-06'),
      ('cafe1000-0000-4000-8000-0000000000b2', 'task', '${TAREA_TERMINADA}',
       'Segundo comentario, más reciente.', 'qa-e2e2-tl', 'TL', '2026-11-07 09:00:00-06'),
      ('cafe1000-0000-4000-8000-0000000000b3', 'bug', '${BUG_ABIERTO}',
       'Comentario de QA en el bug.', 'qa-e2e2-qa', 'QA', '2026-11-06 11:00:00-06')`,
  )
  psql(
    `INSERT INTO task_status_changes (id, task_id, from_status, to_status, actor_name, actor_profile, reason, changed_at) VALUES
      ('cafe1000-0000-4000-8000-0000000000c1', '${TAREA_TERMINADA}', 'backlog', 'ready',
       'qa-e2e2-tl', 'TL', 'Arranque de la etapa', '2026-11-04 09:00:00-06'),
      ('cafe1000-0000-4000-8000-0000000000c2', '${TAREA_TERMINADA}', 'pending_close', 'done',
       'qa-e2e2-tl', 'TL', '', '2026-11-07 05:00:00-06')`,
  )
  psql(
    `INSERT INTO bug_status_changes (id, bug_id, from_status, to_status, actor_name, actor_profile, reason, changed_at) VALUES
      ('cafe1000-0000-4000-8000-0000000000d1', '${BUG_ABIERTO}', 'reported', 'confirmed',
       'qa-e2e2-qa', 'QA', 'Reproducido en la instancia de pruebas', '2026-11-06 09:00:00-06')`,
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

test.describe('criterio 3 — avance del proyecto', () => {
  test('GET /projects/{id}/progress devuelve los campos y tipos del fixture', async () => {
    const respuesta = await api.get(url(`/projects/${PROYECTO}/progress`))
    expect(respuesta.status()).toBe(200)
    const avance = await respuesta.json()

    expect(avance.project).toEqual({
      id: PROYECTO,
      name: 'Proyecto QA avance',
      status: 'working',
      // El último cambio del proyecto, sus etapas, tareas o bugs: la tarea terminada.
      last_change_at: '2026-11-07T05:00:00-06:00',
    })

    expect(avance.stages.map((e: { id: string }) => e.id)).toEqual([
      ETAPA_EN_CURSO,
      ETAPA_EN_ESPERA,
    ])
    const [enCurso, enEspera] = avance.stages
    expect(enCurso).toMatchObject({
      id: ETAPA_EN_CURSO,
      name: 'Etapa en curso',
      status: 'in_dev',
      lead: 'qa-e2e2-tl',
      members: ['qa-e2e2-dev', 'qa-e2e2-qa'],
      counts: { backlog: 1, ready: 1, done: 1 },
    })
    expect(Object.values(enCurso.counts).every((n) => Number.isInteger(n) && n >= 0)).toBe(true)
    expect(enCurso.tasks.map((t: { id: string }) => t.id)).toEqual([
      TAREA_LISTA,
      TAREA_TERMINADA,
      TAREA_SIN_HISTORIAL,
    ])
    const [lista, terminada] = enCurso.tasks
    expect(lista).toMatchObject({
      id: TAREA_LISTA,
      title: 'Tarea e2e lista',
      status: 'ready',
    })
    expect(lista.updated_at).toMatch(RFC3339)
    expect(lista).not.toHaveProperty('assignee')
    expect(terminada).toMatchObject({ status: 'done', assignee: 'qa-e2e2-dev' })
    expect(ESTADOS_TAREA).toContain(lista.status)

    // Sin lead ni miembros: los opcionales se omiten, las listas llegan vacías.
    expect(enEspera).toMatchObject({
      id: ETAPA_EN_ESPERA,
      name: 'Etapa en espera',
      status: 'stand_by',
      members: [],
      counts: {},
      tasks: [],
    })
    expect(enEspera).not.toHaveProperty('lead')

    expect(avance.open_bugs.counts).toEqual({ confirmed: 1 })
    expect(avance.open_bugs.items).toEqual([
      {
        id: BUG_ABIERTO,
        title: 'Bug e2e abierto',
        status: 'confirmed',
        updated_at: '2026-11-06T09:00:00-06:00',
      },
    ])

    // ready sin nada en curso en la etapa en in_dev.
    expect(avance.signals).toEqual([
      { code: 'ready_without_work', stage_id: ETAPA_EN_CURSO, stage_name: 'Etapa en curso' },
    ])
  })

  test('GET /projects/{id}/progress de un proyecto inexistente responde 404 con error', async () => {
    const respuesta = await api.get(url(`/projects/${INEXISTENTE}/progress`))
    expect(respuesta.status()).toBe(404)
    const cuerpo = await respuesta.json()
    expect(typeof cuerpo.error).toBe('string')
    expect(cuerpo.error.length).toBeGreaterThan(0)
  })
})

test.describe('criterio 6 — comentarios', () => {
  test('GET /comments?item_kind=task devuelve los campos y tipos del fixture, por fecha', async () => {
    const respuesta = await api.get(
      url(`/comments?item_kind=task&item_id=${TAREA_TERMINADA}`),
    )
    expect(respuesta.status()).toBe(200)
    const comentarios = await respuesta.json()
    expect(comentarios).toHaveLength(2)
    expect(comentarios[0]).toEqual({
      id: 'cafe1000-0000-4000-8000-0000000000b1',
      item_kind: 'task',
      item_id: TAREA_TERMINADA,
      // El cuerpo llega literal, sin interpretar el HTML ni perder el salto de línea.
      body: 'Evidencia: <b>texto</b> con & y salto.\nSegunda línea.',
      actor_name: 'qa-e2e2-dev',
      actor_profile: 'DevFrontend',
      created_at: '2026-11-06T10:00:00-06:00',
    })
    expect(comentarios[1].id).toBe('cafe1000-0000-4000-8000-0000000000b2')
  })

  test('GET /comments?item_kind=bug filtra por elemento y devuelve [] sin comentarios', async () => {
    const delBug = await api.get(url(`/comments?item_kind=bug&item_id=${BUG_ABIERTO}`))
    expect(delBug.status()).toBe(200)
    const comentarios = await delBug.json()
    expect(comentarios).toHaveLength(1)
    expect(comentarios[0]).toMatchObject({
      item_kind: 'bug',
      item_id: BUG_ABIERTO,
      body: 'Comentario de QA en el bug.',
      actor_name: 'qa-e2e2-qa',
      actor_profile: 'QA',
    })
    expect(comentarios[0].created_at).toMatch(RFC3339)

    const sinComentarios = await api.get(
      url(`/comments?item_kind=task&item_id=${TAREA_SIN_HISTORIAL}`),
    )
    expect(sinComentarios.status()).toBe(200)
    expect(await sinComentarios.json()).toEqual([])
  })

  test('GET /comments con item_id que no es UUID responde 400 con error', async () => {
    const respuesta = await api.get(url('/comments?item_kind=task&item_id=abc'))
    expect(respuesta.status()).toBe(400)
    const cuerpo = await respuesta.json()
    expect(typeof cuerpo.error).toBe('string')
    expect(cuerpo.error.length).toBeGreaterThan(0)
  })
})

test.describe('criterio 9 — historial de estados', () => {
  test('GET /tasks/{id}/status-changes devuelve los campos y tipos del fixture', async () => {
    const respuesta = await api.get(url(`/tasks/${TAREA_TERMINADA}/status-changes`))
    expect(respuesta.status()).toBe(200)
    const historial = await respuesta.json()
    expect(historial).toHaveLength(2)
    expect(historial[0]).toEqual({
      id: 'cafe1000-0000-4000-8000-0000000000c1',
      task_id: TAREA_TERMINADA,
      from_status: 'backlog',
      to_status: 'ready',
      actor_name: 'qa-e2e2-tl',
      actor_profile: 'TL',
      reason: 'Arranque de la etapa',
      changed_at: '2026-11-04T09:00:00-06:00',
    })
    // Motivo vacío válido y solo estados de tarea en las dos puntas.
    expect(historial[1].reason).toBe('')
    for (const cambio of historial) {
      expect(ESTADOS_TAREA).toContain(cambio.from_status)
      expect(ESTADOS_TAREA).toContain(cambio.to_status)
      expect(cambio.changed_at).toMatch(RFC3339)
    }

    const sinHistorial = await api.get(url(`/tasks/${TAREA_SIN_HISTORIAL}/status-changes`))
    expect(sinHistorial.status()).toBe(200)
    expect(await sinHistorial.json()).toEqual([])
  })

  test('GET /bugs/{id}/status-changes devuelve los campos y tipos del fixture', async () => {
    const respuesta = await api.get(url(`/bugs/${BUG_ABIERTO}/status-changes`))
    expect(respuesta.status()).toBe(200)
    const historial = await respuesta.json()
    expect(historial).toEqual([
      {
        id: 'cafe1000-0000-4000-8000-0000000000d1',
        bug_id: BUG_ABIERTO,
        from_status: 'reported',
        to_status: 'confirmed',
        actor_name: 'qa-e2e2-qa',
        actor_profile: 'QA',
        reason: 'Reproducido en la instancia de pruebas',
        changed_at: '2026-11-06T09:00:00-06:00',
      },
    ])
    for (const cambio of historial) {
      expect(ESTADOS_BUG).toContain(cambio.from_status)
      expect(ESTADOS_BUG).toContain(cambio.to_status)
      expect(cambio).not.toHaveProperty('task_id')
    }

    const sinHistorial = await api.get(url(`/bugs/${BUG_CERRADO}/status-changes`))
    expect(sinHistorial.status()).toBe(200)
    expect(await sinHistorial.json()).toEqual([])
  })
})
