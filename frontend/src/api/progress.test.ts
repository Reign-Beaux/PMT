import { afterEach, describe, expect, it, vi } from 'vitest'
import { countOf, getProjectProgress } from './progress'

const PROJECT_ID = 'b7d03d95-2c39-4a51-a35c-04a55a03acdd'

// Fixtures con la forma real del backend; QA los usa como referencia (criterio 3).
const PROGRESS = {
  project: {
    id: PROJECT_ID,
    name: 'PMT',
    status: 'working',
    last_change_at: '2026-11-07T09:00:00-06:00',
  },
  stages: [
    {
      id: 'f27b13db-6a93-4878-aae6-465827b849f9',
      name: 'Frontend',
      status: 'in_dev',
      lead: 'pmt-frontend-tl',
      members: ['pmt-frontend-devfront', 'pmt-frontend-devback'],
      counts: { backlog: 2, ready: 5, in_dev: 3, done: 8 },
      tasks: [
        {
          id: '70aece42-3e49-4723-87a1-035a17c5af79',
          title: 'Frontend — 15 — Consulta de avance',
          status: 'in_dev',
          assignee: 'pmt-frontend-devfront',
          updated_at: '2026-11-07T11:04:04-06:00',
        },
        {
          id: 'a00b2fd2-8f9d-4447-89c3-4432aa31bfec',
          title: 'Frontend — 13 — Consultas de tareas',
          status: 'ready',
          updated_at: '2026-11-03T10:39:47-06:00',
        },
      ],
    },
  ],
  open_bugs: {
    counts: { reported: 1, confirmed: 1 },
    items: [
      {
        id: '7c9e6679-7425-40de-a82f-38cd8fdc1b08',
        title: 'El kanban no refresca',
        status: 'reported',
        updated_at: '2026-11-06T18:20:00-06:00',
      },
    ],
  },
  signals: [
    {
      code: 'stage_not_closed',
      stage_id: 'f27b13db-6a93-4878-aae6-465827b849f9',
      stage_name: 'Frontend',
    },
  ],
}

const EMPTY_PROGRESS = {
  project: {
    id: PROJECT_ID,
    name: 'PMT',
    status: 'planning',
    last_change_at: '2026-11-07T09:00:00-06:00',
  },
  stages: [],
  open_bugs: { counts: {}, items: [] },
  signals: [],
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

describe('ruta y datos', () => {
  it('pide /api/projects/{id}/progress y devuelve el objeto', async () => {
    const fetchMock = stubFetch(PROGRESS)

    const progress = await getProjectProgress(PROJECT_ID)

    expect(fetchMock.mock.calls[0][0]).toBe(`/api/projects/${PROJECT_ID}/progress`)
    expect(progress).toEqual(PROGRESS)
  })

  it('codifica el id con caracteres especiales', async () => {
    const fetchMock = stubFetch(EMPTY_PROGRESS)

    await getProjectProgress('id raro/&=?')

    expect(fetchMock.mock.calls[0][0]).toBe('/api/projects/id%20raro%2F%26%3D%3F/progress')
  })

  it('proyecto sin etapas, sin bugs abiertos ni señales es válido', async () => {
    stubFetch(EMPTY_PROGRESS)

    await expect(getProjectProgress(PROJECT_ID)).resolves.toEqual(EMPTY_PROGRESS)
  })

  it('lead y assignee ausentes son válidos', async () => {
    const stage = PROGRESS.stages[0]
    const { lead: _lead, ...stageWithoutLead } = stage
    const body = {
      ...PROGRESS,
      stages: [
        {
          ...stageWithoutLead,
          tasks: stage.tasks.map(({ assignee: _assignee, ...task }) => task),
        },
      ],
    }
    stubFetch(body)

    const progress = await getProjectProgress(PROJECT_ID)

    expect(progress.stages[0]).not.toHaveProperty('lead')
    expect(progress.stages[0].tasks[0]).not.toHaveProperty('assignee')
  })
})

describe('datos inválidos', () => {
  const stage = PROGRESS.stages[0]

  it.each([
    ['stages que no es array', { ...PROGRESS, stages: {} }],
    [
      'members ausente',
      { ...PROGRESS, stages: [{ ...stage, members: undefined }] },
    ],
    [
      'count negativo',
      { ...PROGRESS, stages: [{ ...stage, counts: { ready: -1 } }] },
    ],
    [
      'count que es string',
      { ...PROGRESS, stages: [{ ...stage, counts: { ready: '3' } }] },
    ],
    [
      'estado de proyecto desconocido',
      { ...PROGRESS, project: { ...PROGRESS.project, status: 'archived' } },
    ],
    [
      'estado de etapa desconocido',
      { ...PROGRESS, stages: [{ ...stage, status: 'finished' }] },
    ],
    [
      'estado de tarea desconocido',
      {
        ...PROGRESS,
        stages: [
          { ...stage, tasks: [{ ...stage.tasks[0], status: 'archived' }] },
        ],
      },
    ],
    [
      'estado de bug desconocido',
      {
        ...PROGRESS,
        open_bugs: {
          ...PROGRESS.open_bugs,
          items: [{ ...PROGRESS.open_bugs.items[0], status: 'closed' }],
        },
      },
    ],
    [
      'signal sin stage_name',
      {
        ...PROGRESS,
        signals: [{ code: 'stage_not_closed', stage_id: stage.id }],
      },
    ],
  ])('%s rechaza con invalid_response', async (_caso, body) => {
    stubFetch(body)

    await expect(getProjectProgress(PROJECT_ID)).rejects.toMatchObject({
      kind: 'invalid_response',
      message: 'Respuesta no válida del servidor',
    })
  })
})

describe('propagación de errores', () => {
  it('404 se propaga sin cambio', async () => {
    stubFetch({ error: 'Proyecto no encontrado' }, 404)

    await expect(getProjectProgress('no-existe')).rejects.toMatchObject({
      kind: 'http',
      status: 404,
      message: 'Proyecto no encontrado',
    })
  })

  it('abort se propaga sin cambio', async () => {
    const fetchMock = stubFetch(PROGRESS)
    const controller = new AbortController()
    controller.abort()

    await expect(
      getProjectProgress(PROJECT_ID, { signal: controller.signal }),
    ).rejects.toMatchObject({ kind: 'aborted' })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('countOf', () => {
  it('devuelve 0 para una clave ausente', () => {
    expect(countOf({ ready: 3 }, 'done')).toBe(0)
  })

  it('devuelve el valor si la clave existe', () => {
    expect(countOf({ ready: 3 }, 'ready')).toBe(3)
  })
})
