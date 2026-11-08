import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { RouteObject } from 'react-router'
import { createMemoryRouter, RouterProvider, useParams } from 'react-router'
import { EventsProvider } from '../realtime/EventsProvider'
import { createFakeConnection } from '../test/fakeConnection'
import type { FakeConnection } from '../test/fakeConnection'
import { StageList } from './StageList'

const STAGE_1 = {
  id: 's1',
  project_id: 'p1',
  name: 'Etapa Uno',
  description: '',
  position: 1,
  status: 'in_dev',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:00:00-06:00',
  lead: 'pmt-frontend-tl',
}

const STAGE_2 = {
  id: 's2',
  project_id: 'p1',
  name: 'Etapa Dos',
  description: '',
  position: 2,
  status: 'done',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:00:00-06:00',
}

const PROGRESS = {
  project: {
    id: 'p1',
    name: 'Proyecto Uno',
    status: 'working',
    last_change_at: '2026-11-01T09:00:00-06:00',
  },
  stages: [
    {
      id: 's1',
      name: 'Etapa Uno',
      status: 'in_dev',
      lead: 'pmt-frontend-tl',
      members: [],
      counts: { done: 2, in_dev: 1 },
      tasks: [],
    },
    {
      id: 's2',
      name: 'Etapa Dos',
      status: 'done',
      members: [],
      counts: { done: 1 },
      tasks: [],
    },
  ],
  open_bugs: { counts: {}, items: [] },
  signals: [],
}

function jsonResponse(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () =>
      Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
  } as unknown as Response
}

interface StubOptions {
  stages?: unknown[]
  progress?: unknown
  stagesStatus?: number
  progressStatus?: number
  /** 1 = la primera petición de stages falla con stagesStatus, la segunda responde ok */
  failStagesOnce?: boolean
  failProgressOnce?: boolean
}

function stubStageListFetch(options: StubOptions = {}) {
  const {
    stages = [STAGE_2, STAGE_1],
    progress = PROGRESS,
    stagesStatus = 500,
    progressStatus = 500,
    failStagesOnce = false,
    failProgressOnce = false,
  } = options
  let stagesCalls = 0
  let progressCalls = 0
  const fetchMock = vi.fn((url: string, _init?: RequestInit) => {
    if (url === '/api/projects/p1/stages') {
      stagesCalls += 1
      const fail = failStagesOnce ? stagesCalls === 1 : false
      return Promise.resolve(
        fail
          ? jsonResponse({ error: 'No se pudo cargar' }, stagesStatus)
          : jsonResponse(stages),
      )
    }
    if (url === '/api/projects/p1/progress') {
      progressCalls += 1
      const fail = failProgressOnce ? progressCalls === 1 : false
      return Promise.resolve(
        fail
          ? jsonResponse({ error: 'No se pudo cargar' }, progressStatus)
          : jsonResponse(progress),
      )
    }
    return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function StageProbe() {
  const { projectId, stageId } = useParams()
  return (
    <p>
      Etapa abierta: {projectId}/{stageId}
    </p>
  )
}

function renderStageList() {
  const routes: RouteObject[] = [
    { path: '/', element: <StageList projectId="p1" /> },
    { path: '/projects/:projectId/stages/:stageId', element: <StageProbe /> },
  ]
  const router = createMemoryRouter(routes, { initialEntries: ['/'] })
  render(<RouterProvider router={router} />)
  return router
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('StageList', () => {
  it('pinta las etapas desordenadas por position con estado y TL', async () => {
    stubStageListFetch()

    renderStageList()
    expect(screen.getByText('Cargando etapas…')).toBeDefined()

    const items = await screen.findAllByRole('listitem')
    expect(items).toHaveLength(2)
    expect(items[0].textContent).toContain('Etapa Uno')
    expect(items[1].textContent).toContain('Etapa Dos')
    expect(screen.getByRole('heading', { level: 2, name: 'Etapas' })).toBeDefined()
    expect(screen.getByText('En desarrollo')).toBeDefined()
    expect(screen.getByText('Terminada')).toBeDefined()
    expect(screen.getByText('TL: pmt-frontend-tl')).toBeDefined()
    expect(screen.getAllByText(/^TL:/)).toHaveLength(1)
  })

  it('muestra el avance calculado con los counts de progress', async () => {
    stubStageListFetch()

    renderStageList()

    const items = await screen.findAllByRole('listitem')
    expect(items[0].textContent).toContain('67 % · 2/3')
    expect(items[1].textContent).toContain('100 % · 1/1')
  })

  it('etapa sin entrada en progress muestra 0 %', async () => {
    stubStageListFetch({
      progress: { ...PROGRESS, stages: [] },
    })

    renderStageList()

    const items = await screen.findAllByRole('listitem')
    expect(items[0].textContent).toContain('0 % · 0/0')
  })

  it('con [] muestra "Este proyecto no tiene etapas"', async () => {
    stubStageListFetch({ stages: [], progress: { ...PROGRESS, stages: [] } })

    renderStageList()

    expect(await screen.findByText('Este proyecto no tiene etapas')).toBeDefined()
  })

  it('si falla stages muestra el error y Reintentar recarga ambas', async () => {
    const fetchMock = stubStageListFetch({ failStagesOnce: true })
    const user = userEvent.setup()

    renderStageList()

    expect(await screen.findByText('No se pudo cargar')).toBeDefined()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(await screen.findByText('Etapa Uno')).toBeDefined()
    // 2 iniciales (stages + progress) + 2 del reintento
    expect(fetchMock).toHaveBeenCalledTimes(4)
  })

  it('si falla progress muestra el error y Reintentar recarga ambas', async () => {
    const fetchMock = stubStageListFetch({ failProgressOnce: true })
    const user = userEvent.setup()

    renderStageList()

    expect(await screen.findByText('No se pudo cargar')).toBeDefined()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(await screen.findByText('Etapa Uno')).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(4)
  })

  it('click en una etapa navega a /projects/:projectId/stages/:stageId', async () => {
    stubStageListFetch()
    const user = userEvent.setup()

    const router = renderStageList()
    await user.click(await screen.findByRole('button', { name: /Etapa Uno/ }))

    expect(await screen.findByText(/Etapa abierta: p1\/s1/)).toBeDefined()
    expect(router.state.location.pathname).toBe('/projects/p1/stages/s1')
  })

  it('Enter en una etapa navega a /projects/:projectId/stages/:stageId', async () => {
    stubStageListFetch()
    const user = userEvent.setup()

    const router = renderStageList()
    const card = await screen.findByRole('button', { name: /Etapa Dos/ })
    card.focus()
    await user.keyboard('{Enter}')

    expect(await screen.findByText(/Etapa abierta: p1\/s2/)).toBeDefined()
    expect(router.state.location.pathname).toBe('/projects/p1/stages/s2')
  })
})

describe('StageList — refresco por eventos (tarea 03)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function renderWithEvents(connection: FakeConnection) {
    const routes: RouteObject[] = [
      { path: '/', element: <StageList projectId="p1" /> },
    ]
    const router = createMemoryRouter(routes, { initialEntries: ['/'] })
    render(<RouterProvider router={router} />, {
      wrapper: ({ children }) => (
        <EventsProvider connection={connection}>{children}</EventsProvider>
      ),
    })
  }

  it('un invalidate de stage o task vuelve a pedir etapas y avance; project no', async () => {
    const fetchMock = stubStageListFetch()
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getAllByRole('listitem')).toHaveLength(2)
    expect(fetchMock).toHaveBeenCalledTimes(2)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'project' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    expect(fetchMock).toHaveBeenCalledTimes(2)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(fetchMock).toHaveBeenCalledTimes(4)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'stage' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(fetchMock).toHaveBeenCalledTimes(6)
  })
})
