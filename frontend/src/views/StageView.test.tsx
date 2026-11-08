import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { RouteObject } from 'react-router'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { EventsProvider } from '../realtime/EventsProvider'
import { createFakeConnection } from '../test/fakeConnection'
import type { FakeConnection } from '../test/fakeConnection'
import { ShellLayout } from '../shell/ShellLayout'
import { StageView } from './StageView'

beforeAll(() => {
  const proto = HTMLDialogElement.prototype
  if (typeof proto.showModal !== 'function') {
    Object.defineProperty(proto, 'showModal', {
      value() {
        this.setAttribute('open', '')
      },
      configurable: true,
    })
  }
  if (typeof proto.close !== 'function') {
    Object.defineProperty(proto, 'close', {
      value() {
        this.removeAttribute('open')
      },
      configurable: true,
    })
  }
})

const PROJECT_P1 = {
  id: 'p1',
  name: 'Proyecto Uno',
  description: '',
  status: 'working',
  product_version: '1.0.0',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:00:00-06:00',
}

const STAGE_S1 = {
  id: 's1',
  project_id: 'p1',
  name: 'Etapa Uno',
  description: '',
  position: 1,
  status: 'in_dev',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:00:00-06:00',
}

function makeTask(id: string, status: string, assignee?: string) {
  return {
    id,
    stage_id: 's1',
    title: `Tarea ${id}`,
    description: '',
    status,
    created_at: '2026-11-01T09:00:00-06:00',
    updated_at: '2026-11-02T10:00:00-06:00',
    ...(assignee !== undefined ? { assignee } : {}),
  }
}

// 2 done de 3 no canceladas → 67 %
const TASKS = [
  makeTask('t1', 'in_dev', 'pmt-frontend-devfront'),
  makeTask('t2', 'done'),
  makeTask('t3', 'done', 'pmt-frontend-devback'),
  makeTask('t4', 'cancelled'),
]

function jsonResponse(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () =>
      Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
  } as unknown as Response
}

interface StubOptions {
  tasks?: unknown[]
  stages?: unknown[]
  failTasksOnce?: boolean
  failStagesOnce?: boolean
  /** GET /api/stages/:id/tasks responde 404: la etapa no existe en la base. */
  tasks404?: boolean
  /** La petición de tareas nunca responde: la vista se queda cargando. */
  hangTasks?: boolean
}

function stubFetch(options: StubOptions = {}) {
  const {
    tasks = TASKS,
    stages = [STAGE_S1],
    failTasksOnce = false,
    failStagesOnce = false,
    tasks404 = false,
    hangTasks = false,
  } = options
  let tasksCalls = 0
  let stagesCalls = 0
  const fetchMock = vi.fn((url: string, _init?: RequestInit) => {
    if (url === '/api/projects/p1') {
      return Promise.resolve(jsonResponse(PROJECT_P1))
    }
    if (url === '/api/projects/p1/stages') {
      stagesCalls += 1
      const fail = failStagesOnce ? stagesCalls === 1 : false
      return Promise.resolve(
        fail
          ? jsonResponse({ error: 'No se pudo cargar' }, 500)
          : jsonResponse(stages),
      )
    }
    if (url.startsWith('/api/stages/') && url.endsWith('/tasks')) {
      tasksCalls += 1
      if (hangTasks) return new Promise<Response>(() => {})
      if (tasks404) {
        return Promise.resolve(
          jsonResponse({ error: 'etapa no encontrada' }, 404),
        )
      }
      const fail = failTasksOnce ? tasksCalls === 1 : false
      return Promise.resolve(
        fail
          ? jsonResponse({ error: 'No se pudo cargar' }, 500)
          : jsonResponse(tasks),
      )
    }
    return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function renderStageView(initialEntry = '/projects/p1/stages/s1') {
  const routes: RouteObject[] = [
    {
      element: <ShellLayout />,
      children: [
        { path: '/projects/:projectId/stages/:stageId', element: <StageView /> },
      ],
    },
  ]
  const router = createMemoryRouter(routes, { initialEntries: [initialEntry] })
  render(<RouterProvider router={router} />)
  return router
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('StageView', () => {
  it('muestra las nueve columnas con sus conteos y cada tarjeta en su columna', async () => {
    stubFetch()

    renderStageView()

    const board = await screen.findByRole('region', {
      name: 'Tablero de la etapa',
    })
    const columns = within(board).getAllByRole('region')
    expect(columns).toHaveLength(9)

    const inDev = within(board).getByRole('region', { name: /En desarrollo/ })
    expect(within(inDev).getAllByRole('listitem')).toHaveLength(1)
    expect(within(inDev).getByRole('button', { name: /Tarea t1/ })).toBeDefined()

    const done = within(board).getByRole('region', { name: /Terminada/ })
    expect(within(done).getAllByRole('listitem')).toHaveLength(2)

    const cancelled = within(board).getByRole('region', { name: /Cancelada/ })
    expect(within(cancelled).getAllByRole('listitem')).toHaveLength(1)

    const backlog = within(board).getByRole('region', { name: /Backlog/ })
    expect(within(backlog).queryByRole('listitem')).toBeNull()
    expect(within(backlog).getByText('Sin elementos')).toBeDefined()
  })

  it('encabezado con nombre, estado en español y avance; el shell muestra la etapa', async () => {
    stubFetch()

    renderStageView()

    const heading = await screen.findByRole('heading', {
      level: 2,
      name: 'Etapa Uno',
    })
    const header = heading.closest('header')
    expect(header).not.toBeNull()
    expect(within(header!).getByText('En desarrollo')).toBeDefined()
    expect(within(header!).getByText('67 % · 2/3')).toBeDefined()
    // h2 de la vista + span del contexto del shell (llega en el segundo render)
    await waitFor(() => expect(screen.getAllByText('Etapa Uno')).toHaveLength(2))
  })

  it('tarea sin assignee muestra "Sin asignar"', async () => {
    stubFetch({ tasks: [makeTask('t9', 'ready')] })

    renderStageView()

    const card = await screen.findByRole('button', { name: /Tarea t9/ })
    expect(card.textContent).toContain('Sin asignar')
  })

  it('click en una tarjeta deja ?task=:id en la URL', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderStageView()
    await user.click(await screen.findByRole('button', { name: /Tarea t1/ }))

    expect(router.state.location.search).toBe('?task=t1')
    expect(router.state.location.pathname).toBe('/projects/p1/stages/s1')
  })

  it('Enter en una tarjeta deja ?task=:id en la URL', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderStageView()
    const card = await screen.findByRole('button', { name: /Tarea t3/ })
    card.focus()
    await user.keyboard('{Enter}')

    expect(router.state.location.search).toBe('?task=t3')
  })

  it('stageId que no está en el proyecto muestra "Etapa no encontrada"', async () => {
    stubFetch({ tasks: [] })

    renderStageView('/projects/p1/stages/s9')

    expect(await screen.findByText('Etapa no encontrada')).toBeDefined()
    const link = screen.getByRole('link', { name: 'Volver al proyecto' })
    expect(link.getAttribute('href')).toBe('/projects/p1')
    expect(screen.queryByRole('region', { name: 'Tablero de la etapa' })).toBeNull()
  })

  it('stageId inexistente (tasks responde 404) muestra "Etapa no encontrada" sin Reintentar', async () => {
    stubFetch({ tasks404: true })

    renderStageView('/projects/p1/stages/s9')

    expect(await screen.findByText('Etapa no encontrada')).toBeDefined()
    const link = screen.getByRole('link', { name: 'Volver al proyecto' })
    expect(link.getAttribute('href')).toBe('/projects/p1')
    expect(screen.queryByRole('button', { name: 'Reintentar' })).toBeNull()
    expect(screen.queryByRole('region', { name: 'Tablero de la etapa' })).toBeNull()
  })

  it('si falla tasks muestra el error y Reintentar recarga ambas', async () => {
    const fetchMock = stubFetch({ failTasksOnce: true })
    const user = userEvent.setup()

    renderStageView()

    expect(await screen.findByText('No se pudo cargar')).toBeDefined()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(
      await screen.findByRole('region', { name: 'Tablero de la etapa' }),
    ).toBeDefined()
    // 1 shell + 2 iniciales (tasks falla, stages ok) + 2 del reintento
    expect(fetchMock).toHaveBeenCalledTimes(5)
  })

  it('si falla stages muestra el error y Reintentar recarga ambas', async () => {
    const fetchMock = stubFetch({ failStagesOnce: true })
    const user = userEvent.setup()

    renderStageView()

    expect(await screen.findByText('No se pudo cargar')).toBeDefined()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(
      await screen.findByRole('region', { name: 'Tablero de la etapa' }),
    ).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(5)
  })

  it('muestra "Cargando etapa…" mientras carga', async () => {
    stubFetch({ hangTasks: true })

    renderStageView()

    expect(await screen.findByText('Cargando etapa…')).toBeDefined()
  })
})

describe('StageView — refresco por eventos (tarea 03)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function renderWithEvents(connection: FakeConnection) {
    const routes: RouteObject[] = [
      {
        element: <ShellLayout />,
        children: [
          {
            path: '/projects/:projectId/stages/:stageId',
            element: <StageView />,
          },
        ],
      },
    ]
    const router = createMemoryRouter(routes, {
      initialEntries: ['/projects/p1/stages/s1'],
    })
    render(<RouterProvider router={router} />, {
      wrapper: ({ children }) => (
        <EventsProvider connection={connection}>{children}</EventsProvider>
      ),
    })
    return router
  }

  it('un invalidate de task o stage refresca el kanban; comment no', async () => {
    const fetchMock = stubFetch()
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByRole('heading', { name: 'Etapa Uno' })).toBeDefined()
    const kanbanCalls = () =>
      fetchMock.mock.calls.filter(([url]) =>
        String(url).startsWith('/api/stages/'),
      ).length
    expect(kanbanCalls()).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'comment' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(kanbanCalls()).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(kanbanCalls()).toBe(2)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'stage' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(kanbanCalls()).toBe(3)
  })

  it('tras el refresco el kanban muestra las tareas actualizadas', async () => {
    let tasksCalls = 0
    const fetchMock = vi.fn((url: string) => {
      if (url === '/api/projects/p1') {
        return Promise.resolve(jsonResponse(PROJECT_P1))
      }
      if (url === '/api/projects/p1/stages') {
        return Promise.resolve(jsonResponse([STAGE_S1]))
      }
      if (url.startsWith('/api/stages/') && url.endsWith('/tasks')) {
        tasksCalls += 1
        return Promise.resolve(
          jsonResponse(
            tasksCalls === 1 ? TASKS : [...TASKS, makeTask('t5', 'in_dev')],
          ),
        )
      }
      return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
    })
    vi.stubGlobal('fetch', fetchMock)
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.queryByText('Tarea t5')).toBeNull()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(screen.getByText('Tarea t5')).toBeDefined()
    expect(screen.getByText('Tarea t1')).toBeDefined()
  })
})
