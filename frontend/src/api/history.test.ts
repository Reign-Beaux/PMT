import { afterEach, describe, expect, it, vi } from 'vitest'
import { getBugHistory, getTaskHistory } from './history'

const TASK_ID = '70aece42-3e49-4723-87a1-035a17c5af79'
const BUG_ID = '7c9e6679-7425-40de-a82f-38cd8fdc1b08'

// Fixtures con la forma real del backend; QA los usa como referencia (criterio 9).
const TASK_HISTORY = {
  id: '3f2a1b0c-4d5e-6f70-8192-a3b4c5d6e7f8',
  task_id: TASK_ID,
  from_status: 'ready',
  to_status: 'in_dev',
  actor_name: 'pmt-frontend-devfront',
  actor_profile: 'DevFrontend',
  reason: '',
  changed_at: '2026-11-07T11:04:04-06:00',
}

const BUG_HISTORY = {
  id: '4a3b2c1d-0e9f-8a7b-6c5d-4e3f2a1b0c9d',
  bug_id: BUG_ID,
  from_status: 'reported',
  to_status: 'confirmed',
  actor_name: 'pmt-frontend-qa',
  actor_profile: 'QA',
  reason: 'Reproducido en pantalla',
  changed_at: '2026-11-06T19:00:00-06:00',
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
  it('getTaskHistory pide /api/tasks/{id}/status-changes y devuelve la lista', async () => {
    const fetchMock = stubFetch([TASK_HISTORY])

    const history = await getTaskHistory(TASK_ID)

    expect(fetchMock.mock.calls[0][0]).toBe(`/api/tasks/${TASK_ID}/status-changes`)
    expect(history).toEqual([TASK_HISTORY])
  })

  it('getBugHistory pide /api/bugs/{id}/status-changes y devuelve la lista', async () => {
    const fetchMock = stubFetch([BUG_HISTORY])

    const history = await getBugHistory(BUG_ID)

    expect(fetchMock.mock.calls[0][0]).toBe(`/api/bugs/${BUG_ID}/status-changes`)
    expect(history).toEqual([BUG_HISTORY])
  })

  it('la lista vacía es válida', async () => {
    stubFetch([])

    await expect(getTaskHistory(TASK_ID)).resolves.toEqual([])
    await expect(getBugHistory(BUG_ID)).resolves.toEqual([])
  })

  it('reason vacío es válido', async () => {
    stubFetch([{ ...BUG_HISTORY, reason: '' }])

    const history = await getBugHistory(BUG_ID)

    expect(history[0].reason).toBe('')
  })
})

describe('datos inválidos', () => {
  it.each([
    [
      'historial de tarea con estado de bug',
      () => getTaskHistory(TASK_ID),
      [{ ...TASK_HISTORY, to_status: 'reported' }],
    ],
    [
      'historial de bug con estado de tarea',
      () => getBugHistory(BUG_ID),
      [{ ...BUG_HISTORY, from_status: 'backlog' }],
    ],
    [
      'historial de tarea sin task_id',
      () => getTaskHistory(TASK_ID),
      [{ ...TASK_HISTORY, task_id: undefined }],
    ],
    [
      'historial de bug sin bug_id',
      () => getBugHistory(BUG_ID),
      [{ ...BUG_HISTORY, bug_id: undefined }],
    ],
  ])('%s rechaza con invalid_response', async (_caso, consulta, body) => {
    stubFetch(body)

    await expect(consulta()).rejects.toMatchObject({
      kind: 'invalid_response',
      message: 'Respuesta no válida del servidor',
    })
  })

  it('objeto en lugar de array rechaza con invalid_response', async () => {
    stubFetch({ changes: [TASK_HISTORY] })

    await expect(getTaskHistory(TASK_ID)).rejects.toMatchObject({
      kind: 'invalid_response',
    })
  })
})

describe('propagación de errores', () => {
  it('404 se propaga sin cambio', async () => {
    stubFetch({ error: 'Tarea no encontrada' }, 404)

    await expect(getTaskHistory('no-existe')).rejects.toMatchObject({
      kind: 'http',
      status: 404,
      message: 'Tarea no encontrada',
    })
  })

  it('abort se propaga sin cambio', async () => {
    const fetchMock = stubFetch([BUG_HISTORY])
    const controller = new AbortController()
    controller.abort()

    await expect(
      getBugHistory(BUG_ID, { signal: controller.signal }),
    ).rejects.toMatchObject({ kind: 'aborted' })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
