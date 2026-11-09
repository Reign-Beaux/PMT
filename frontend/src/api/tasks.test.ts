import { afterEach, describe, expect, it, vi } from 'vitest'
import { getStageTasks, getTask } from './tasks'

const STAGE_ID = 'f27b13db-6a93-4878-aae6-465827b849f9'

// Fixtures con la forma real del backend; QA los usa como referencia (criterio 10).
const TASK_WITH_ASSIGNEE = {
  id: '964f5d78-a902-400a-b971-2cd5e3fe3368',
  stage_id: STAGE_ID,
  title: 'Frontend — 11 — Cliente HTTP común',
  description: 'Helper GET con fetch, timeout y HttpError.',
  status: 'in_dev',
  created_at: '2026-11-03T10:39:47-06:00',
  updated_at: '2026-11-07T05:39:08-06:00',
  assignee: 'pmt-frontend-devfront',
}

const TASK_WITHOUT_ASSIGNEE = {
  id: 'a00b2fd2-8f9d-4447-89c3-4432aa31bfec',
  stage_id: STAGE_ID,
  title: 'Frontend — 13 — Consultas de tareas',
  description: 'Adaptadores GET de tareas.',
  status: 'ready',
  created_at: '2026-11-03T10:39:47-06:00',
  updated_at: '2026-11-03T10:39:47-06:00',
}

function stubFetch(body: unknown, status = 200) {
  const fetchMock = vi.fn((_url: string, _init?: RequestInit) =>
    Promise.resolve({
      ok: status >= 200 && status < 300,
      status,
      text: () =>
        Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
    } as unknown as Response),
  )
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('rutas y datos', () => {
  it('getStageTasks pide /api/stages/{id}/tasks sin query y devuelve la lista', async () => {
    const fetchMock = stubFetch([TASK_WITH_ASSIGNEE, TASK_WITHOUT_ASSIGNEE])

    const tasks = await getStageTasks(STAGE_ID)

    expect(fetchMock.mock.calls[0][0]).toBe(`/api/stages/${STAGE_ID}/tasks`)
    expect(fetchMock.mock.calls[0][0]).not.toContain('?')
    expect(tasks).toEqual([TASK_WITH_ASSIGNEE, TASK_WITHOUT_ASSIGNEE])
  })

  it('getTask pide /api/tasks/{id} y devuelve la tarea', async () => {
    const fetchMock = stubFetch(TASK_WITH_ASSIGNEE)

    const task = await getTask(TASK_WITH_ASSIGNEE.id)

    expect(fetchMock.mock.calls[0][0]).toBe(`/api/tasks/${TASK_WITH_ASSIGNEE.id}`)
    expect(task).toEqual(TASK_WITH_ASSIGNEE)
  })

  it('la lista vacía es válida', async () => {
    stubFetch([])

    await expect(getStageTasks(STAGE_ID)).resolves.toEqual([])
  })

  it('assignee ausente es válido', async () => {
    stubFetch(TASK_WITHOUT_ASSIGNEE)

    const task = await getTask(TASK_WITHOUT_ASSIGNEE.id)

    expect(task).toEqual(TASK_WITHOUT_ASSIGNEE)
    expect(task).not.toHaveProperty('assignee')
  })

  it('done_at string se conserva', async () => {
    const doneAt = '2026-11-05T18:20:30-06:00'
    stubFetch({ ...TASK_WITH_ASSIGNEE, done_at: doneAt })

    const task = await getTask(TASK_WITH_ASSIGNEE.id)

    expect(task.done_at).toBe(doneAt)
  })

  it('done_at ausente se omite', async () => {
    stubFetch(TASK_WITH_ASSIGNEE)

    const task = await getTask(TASK_WITH_ASSIGNEE.id)

    expect(task).not.toHaveProperty('done_at')
  })
})

describe('datos inválidos', () => {
  it.each([
    ['estado desconocido', { ...TASK_WITH_ASSIGNEE, status: 'archived' }],
    ['title que no es string', { ...TASK_WITH_ASSIGNEE, title: 42 }],
    ['done_at que no es string', { ...TASK_WITH_ASSIGNEE, done_at: 17 }],
  ])('tarea con %s rechaza con invalid_response', async (_caso, body) => {
    stubFetch(body)

    await expect(getTask(TASK_WITH_ASSIGNEE.id)).rejects.toMatchObject({
      kind: 'invalid_response',
      message: 'Respuesta no válida del servidor',
    })
  })

  it('objeto en lugar de array rechaza con invalid_response', async () => {
    stubFetch({ tasks: [TASK_WITH_ASSIGNEE] })

    await expect(getStageTasks(STAGE_ID)).rejects.toMatchObject({
      kind: 'invalid_response',
    })
  })
})

describe('propagación de errores', () => {
  it('404 se propaga sin cambio', async () => {
    stubFetch({ error: 'Tarea no encontrada' }, 404)

    await expect(getTask('no-existe')).rejects.toMatchObject({
      kind: 'http',
      status: 404,
      message: 'Tarea no encontrada',
    })
  })

  it('abort se propaga sin cambio', async () => {
    const fetchMock = stubFetch([TASK_WITH_ASSIGNEE])
    const controller = new AbortController()
    controller.abort()

    await expect(getStageTasks(STAGE_ID, { signal: controller.signal })).rejects.toMatchObject({
      kind: 'aborted',
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
