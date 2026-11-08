import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { RouteObject } from 'react-router'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { EventsProvider } from '../realtime/EventsProvider'
import { createFakeConnection } from '../test/fakeConnection'
import type { FakeConnection } from '../test/fakeConnection'
import { ShellLayout } from '../shell/ShellLayout'
import { StageView } from './StageView'
import { formatDate } from '../domain/formatDate'

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

function makeTask(overrides: Record<string, unknown> = {}) {
  return {
    id: 't1',
    stage_id: 's1',
    title: 'Tarea uno',
    description: 'Descripción de la tarea',
    status: 'in_dev',
    created_at: '2026-11-01T09:00:00-06:00',
    updated_at: '2026-11-02T10:00:00-06:00',
    assignee: 'pmt-frontend-devfront',
    ...overrides,
  }
}

const TASKS = [makeTask(), makeTask({ id: 't2', title: 'Tarea dos' })]

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
  detail?: unknown
  detailStatus?: number
  failDetailOnce?: boolean
  hangTasks?: boolean
  hangDetail?: boolean
  comments?: unknown[]
  history?: unknown[]
}

function stubFetch(options: StubOptions = {}) {
  const {
    tasks = TASKS,
    detail = makeTask(),
    detailStatus = 200,
    failDetailOnce = false,
    hangTasks = false,
    hangDetail = false,
    comments = [],
    history = [],
  } = options
  let detailCalls = 0
  const fetchMock = vi.fn((url: string) => {
    if (url === '/api/projects/p1') {
      return Promise.resolve(jsonResponse(PROJECT_P1))
    }
    if (url === '/api/projects/p1/stages') {
      return Promise.resolve(jsonResponse([STAGE_S1]))
    }
    if (url === '/api/stages/s1/tasks') {
      if (hangTasks) return new Promise<Response>(() => {})
      return Promise.resolve(jsonResponse(tasks))
    }
    if (url === '/api/tasks/t1') {
      detailCalls += 1
      if (hangDetail) return new Promise<Response>(() => {})
      const fail = failDetailOnce ? detailCalls === 1 : false
      if (fail) {
        return Promise.resolve(jsonResponse({ error: 'No se pudo cargar' }, 500))
      }
      return Promise.resolve(jsonResponse(detail, detailStatus))
    }
    if (url.startsWith('/api/comments?')) {
      return Promise.resolve(jsonResponse(comments))
    }
    if (url === '/api/tasks/t1/status-changes') {
      return Promise.resolve(jsonResponse(history))
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

describe('TaskDetail — diálogo de detalle (tarea 03)', () => {
  it('abrir ?task= muestra el diálogo con título, estado en español, assignee y fechas', async () => {
    stubFetch()

    renderStageView('/projects/p1/stages/s1?task=t1')

    const dialog = await screen.findByRole('dialog', { name: 'Tarea uno' })
    expect(dialog.hasAttribute('open')).toBe(true)
    expect(within(dialog).getByText('Estado')).toBeDefined()
    expect(within(dialog).getByText('En desarrollo')).toBeDefined()
    expect(within(dialog).getByText('Asignado a')).toBeDefined()
    expect(within(dialog).getByText('pmt-frontend-devfront')).toBeDefined()
    expect(within(dialog).getByText('Creado')).toBeDefined()
    expect(
      within(dialog).getByText(formatDate('2026-11-01T09:00:00-06:00')),
    ).toBeDefined()
    expect(within(dialog).getByText('Actualizado')).toBeDefined()
    expect(
      within(dialog).getByText(formatDate('2026-11-02T10:00:00-06:00')),
    ).toBeDefined()
  })

  it('assignee ausente muestra "Sin asignar"', async () => {
    stubFetch({ detail: makeTask({ assignee: undefined }) })

    renderStageView('/projects/p1/stages/s1?task=t1')

    const dialog = await screen.findByRole('dialog', { name: 'Tarea uno' })
    expect(within(dialog).getByText('Sin asignar')).toBeDefined()
  })

  it('la descripción con HTML y saltos de línea se ve literal', async () => {
    stubFetch({
      detail: makeTask({ description: '<b>negrita</b>\nsegunda línea' }),
    })

    renderStageView('/projects/p1/stages/s1?task=t1')

    const dialog = await screen.findByRole('dialog', { name: 'Tarea uno' })
    const descripcion = within(dialog).getByRole('region', {
      name: 'Descripción',
    })
    expect(descripcion.querySelector('p')?.textContent).toBe(
      '<b>negrita</b>\nsegunda línea',
    )
    expect(document.querySelector('b')).toBeNull()
  })

  it('el detalle incluye las secciones Comentarios e Historial de estados', async () => {
    stubFetch({
      comments: [
        {
          id: 'c1',
          item_kind: 'task',
          item_id: 't1',
          body: 'Comentario de la tarea',
          actor_name: 'Ana',
          actor_profile: 'pmt-frontend-pp',
          created_at: '2026-11-01T09:00:00-06:00',
        },
      ],
      history: [
        {
          id: 'h1',
          task_id: 't1',
          from_status: 'ready',
          to_status: 'in_dev',
          actor_name: 'Dana',
          actor_profile: 'pmt-frontend-tl',
          reason: 'Empieza',
          changed_at: '2026-11-01T09:00:00-06:00',
        },
      ],
    })

    renderStageView('/projects/p1/stages/s1?task=t1')

    const dialog = await screen.findByRole('dialog', { name: 'Tarea uno' })
    expect(
      within(dialog).getByRole('region', { name: 'Comentarios' }),
    ).toBeDefined()
    expect(
      await within(dialog).findByText('Comentario de la tarea'),
    ).toBeDefined()
    expect(
      within(dialog).getByRole('region', { name: 'Historial de estados' }),
    ).toBeDefined()
    expect(
      await within(dialog).findByText('Lista → En desarrollo'),
    ).toBeDefined()
  })

  it('cerrar con el botón quita ?task= y devuelve el foco a la tarjeta', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderStageView()
    const card = await screen.findByRole('button', { name: /Tarea uno/ })
    await user.click(card)

    const dialog = await screen.findByRole('dialog', { name: 'Tarea uno' })
    expect(dialog.hasAttribute('open')).toBe(true)
    expect(router.state.location.search).toBe('?task=t1')

    await user.click(within(dialog).getByRole('button', { name: 'Cerrar' }))

    expect(router.state.location.search).toBe('')
    await act(async () => {})
    expect(document.activeElement).toBe(card)
  })

  it('cerrar con Escape quita ?task= y devuelve el foco a la tarjeta', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderStageView()
    const card = await screen.findByRole('button', { name: /Tarea uno/ })
    await user.click(card)
    await screen.findByRole('dialog', { name: 'Tarea uno' })

    fireEvent(
      screen.getByRole('dialog'),
      new Event('cancel', { bubbles: true, cancelable: true }),
    )
    await act(async () => {})

    expect(router.state.location.search).toBe('')
    expect(document.activeElement).toBe(card)
  })

  it('recargar con ?task= abre el detalle aunque el tablero siga cargando', async () => {
    stubFetch({ hangTasks: true })

    renderStageView('/projects/p1/stages/s1?task=t1')

    const dialog = await screen.findByRole('dialog', { name: 'Tarea uno' })
    expect(dialog.hasAttribute('open')).toBe(true)
  })

  it('un 404 muestra "Tarea no encontrada" y se cierra normalmente', async () => {
    stubFetch({ detailStatus: 404, detail: { error: 'tarea no encontrada' } })
    const user = userEvent.setup()

    const router = renderStageView('/projects/p1/stages/s1?task=t1')

    const dialog = await screen.findByRole('dialog', {
      name: 'Tarea no encontrada',
    })
    expect(
      within(dialog).getAllByText('Tarea no encontrada').length,
    ).toBeGreaterThanOrEqual(2)
    expect(screen.queryByRole('button', { name: 'Reintentar' })).toBeNull()

    await user.click(within(dialog).getByRole('button', { name: 'Cerrar' }))
    expect(router.state.location.search).toBe('')
  })

  it('un 500 muestra ErrorState y el reintento vuelve a pedir la tarea', async () => {
    const fetchMock = stubFetch({ failDetailOnce: true })
    const user = userEvent.setup()

    renderStageView('/projects/p1/stages/s1?task=t1')

    const dialog = await screen.findByRole('dialog', { name: 'Tarea' })
    expect(await within(dialog).findByText('No se pudo cargar')).toBeDefined()

    await user.click(within(dialog).getByRole('button', { name: 'Reintentar' }))

    expect(
      await screen.findByRole('dialog', { name: 'Tarea uno' }),
    ).toBeDefined()
    const detailCalls = fetchMock.mock.calls.filter(
      ([url]) => url === '/api/tasks/t1',
    )
    expect(detailCalls).toHaveLength(2)
  })

  it('muestra "Cargando tarea…" mientras carga el detalle', async () => {
    stubFetch({ hangDetail: true })

    renderStageView('/projects/p1/stages/s1?task=t1')

    expect(await screen.findByText('Cargando tarea…')).toBeDefined()
    expect(screen.getByRole('dialog', { name: 'Tarea' })).toBeDefined()
  })
})

describe('TaskDetail — refresco por eventos (tarea 03)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function stubFetchWithDetail(
    detailResponder: (call: number) => ReturnType<typeof jsonResponse>,
  ) {
    let detailCalls = 0
    const fetchMock = vi.fn((url: string) => {
      if (url === '/api/projects/p1') {
        return Promise.resolve(jsonResponse(PROJECT_P1))
      }
      if (url === '/api/projects/p1/stages') {
        return Promise.resolve(jsonResponse([STAGE_S1]))
      }
      if (url === '/api/stages/s1/tasks') {
        return Promise.resolve(jsonResponse(TASKS))
      }
      if (url === '/api/tasks/t1') {
        detailCalls += 1
        return Promise.resolve(detailResponder(detailCalls))
      }
      if (url.startsWith('/api/comments?')) {
        return Promise.resolve(jsonResponse([]))
      }
      if (url === '/api/tasks/t1/status-changes') {
        return Promise.resolve(jsonResponse([]))
      }
      return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
    })
    vi.stubGlobal('fetch', fetchMock)
    return { fetchMock, detailCallCount: () => detailCalls }
  }

  function renderWithEvents(
    connection: FakeConnection,
    initialEntry = '/projects/p1/stages/s1?task=t1',
  ) {
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
      initialEntries: [initialEntry],
    })
    render(<RouterProvider router={router} />, {
      wrapper: ({ children }) => (
        <EventsProvider connection={connection}>{children}</EventsProvider>
      ),
    })
    return router
  }

  it('un invalidate de task vuelve a pedir el detalle y muestra los datos actualizados; comment no', async () => {
    const { detailCallCount } = stubFetchWithDetail((call) =>
      jsonResponse(call === 1 ? makeTask() : makeTask({ title: 'Tarea renovada' })),
    )
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByRole('dialog', { name: 'Tarea uno' })).toBeDefined()
    expect(detailCallCount()).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'comment' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(detailCallCount()).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(detailCallCount()).toBe(2)
    expect(screen.getByRole('dialog', { name: 'Tarea renovada' })).toBeDefined()
  })

  it('un error de refresco muestra ErrorState con Reintentar conservando el detalle previo', async () => {
    stubFetchWithDetail((call) =>
      call === 1
        ? jsonResponse(makeTask())
        : jsonResponse({ error: 'No se pudo cargar' }, 500),
    )
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByRole('dialog', { name: 'Tarea uno' })).toBeDefined()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    const dialog = screen.getByRole('dialog', { name: 'Tarea' })
    expect(within(dialog).getByText('No se pudo cargar')).toBeDefined()
    expect(
      within(dialog).getByRole('button', { name: 'Reintentar' }),
    ).toBeDefined()
    expect(
      within(dialog).getByText('Descripción de la tarea'),
    ).toBeDefined()
  })

  it('un 404 en el refresco muestra "Ya no existe" y se cierra normalmente', async () => {
    stubFetchWithDetail((call) =>
      call === 1
        ? jsonResponse(makeTask())
        : jsonResponse({ error: 'tarea no encontrada' }, 404),
    )
    const connection = createFakeConnection()
    const router = renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByRole('dialog', { name: 'Tarea uno' })).toBeDefined()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    const dialog = screen.getByRole('dialog', { name: 'Tarea uno' })
    expect(within(dialog).getByText('Ya no existe')).toBeDefined()

    await act(async () => {
      fireEvent.click(within(dialog).getByRole('button', { name: 'Cerrar' }))
    })
    expect(router.state.location.search).toBe('')
  })
})
