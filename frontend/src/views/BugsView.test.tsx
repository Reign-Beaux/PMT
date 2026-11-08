import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { RouteObject } from 'react-router'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { EventsProvider } from '../realtime/EventsProvider'
import { createFakeConnection } from '../test/fakeConnection'
import type { FakeConnection } from '../test/fakeConnection'
import { ShellLayout } from '../shell/ShellLayout'
import { BugsView } from './BugsView'
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

function makeBug(overrides: Record<string, unknown> = {}) {
  return {
    id: 'b1',
    project_id: 'p1',
    title: 'Bug uno',
    description: 'Descripción del bug',
    steps_to_reproduce: 'Paso 1',
    product_version: '1.2.3',
    status: 'in_dev',
    reported_by: 'pmt-frontend-pp',
    created_at: '2026-11-01T09:00:00-06:00',
    updated_at: '2026-11-02T10:00:00-06:00',
    ...overrides,
  }
}

const BUGS = [
  makeBug(),
  makeBug({ id: 'b2', title: 'Bug dos', status: 'done' }),
  makeBug({
    id: 'b3',
    title: 'Bug tres',
    status: 'rejected',
    product_version: '',
  }),
  makeBug({ id: 'b4', title: 'Bug cuatro', status: 'reported', reported_by: undefined }),
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
  bugs?: unknown[]
  failOnce?: boolean
  hang?: boolean
  comments?: unknown[]
  history?: unknown[]
}

function stubFetch(options: StubOptions = {}) {
  const {
    bugs = BUGS,
    failOnce = false,
    hang = false,
    comments = [],
    history = [],
  } = options
  let bugsCalls = 0
  const fetchMock = vi.fn((url: string, _init?: RequestInit) => {
    if (url === '/api/projects/p1') {
      return Promise.resolve(jsonResponse(PROJECT_P1))
    }
    if (url === '/api/projects/p1/bugs') {
      bugsCalls += 1
      if (hang) return new Promise<Response>(() => {})
      const fail = failOnce ? bugsCalls === 1 : false
      return Promise.resolve(
        fail
          ? jsonResponse({ error: 'No se pudo cargar' }, 500)
          : jsonResponse(bugs),
      )
    }
    if (url.startsWith('/api/comments?')) {
      return Promise.resolve(jsonResponse(comments))
    }
    if (url.startsWith('/api/bugs/') && url.endsWith('/status-changes')) {
      return Promise.resolve(jsonResponse(history))
    }
    return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function renderBugsView(initialEntry = '/projects/p1/bugs') {
  const routes: RouteObject[] = [
    {
      element: <ShellLayout />,
      children: [{ path: '/projects/:projectId/bugs', element: <BugsView /> }],
    },
  ]
  const router = createMemoryRouter(routes, { initialEntries: [initialEntry] })
  render(<RouterProvider router={router} />)
  return router
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('BugsView — kanban (tarea 02)', () => {
  it('muestra las nueve columnas con sus conteos, incluidas done y rejected', async () => {
    stubFetch()

    renderBugsView()

    const board = await screen.findByRole('region', {
      name: 'Tablero de bugs',
    })
    const columns = within(board).getAllByRole('region')
    expect(columns).toHaveLength(9)

    const inDev = within(board).getByRole('region', { name: /En desarrollo/ })
    expect(within(inDev).getAllByRole('listitem')).toHaveLength(1)
    expect(
      within(inDev).getByRole('button', { name: /Bug uno/ }),
    ).toBeDefined()

    const done = within(board).getByRole('region', { name: /Corregido/ })
    expect(within(done).getAllByRole('listitem')).toHaveLength(1)

    const rejected = within(board).getByRole('region', { name: /Rechazado/ })
    expect(within(rejected).getAllByRole('listitem')).toHaveLength(1)

    const confirmed = within(board).getByRole('region', { name: /Confirmado/ })
    expect(within(confirmed).queryByRole('listitem')).toBeNull()
    expect(within(confirmed).getByText('Sin elementos')).toBeDefined()
  })

  it('la tarjeta muestra el título y la versión; versión vacía muestra "Versión —"', async () => {
    stubFetch()

    renderBugsView()

    const card = await screen.findByRole('button', { name: /Bug uno/ })
    expect(card.textContent).toContain('Versión 1.2.3')

    const sinVersion = screen.getByRole('button', { name: /Bug tres/ })
    expect(sinVersion.textContent).toContain('Versión —')
  })

  it('con lista vacía muestra el mensaje y las nueve columnas en 0', async () => {
    stubFetch({ bugs: [] })

    renderBugsView()

    expect(
      await screen.findByText('Este proyecto no tiene bugs'),
    ).toBeDefined()
    const board = screen.getByRole('region', { name: 'Tablero de bugs' })
    const columns = within(board).getAllByRole('region')
    expect(columns).toHaveLength(9)
    for (const column of columns) {
      expect(within(column).queryByRole('listitem')).toBeNull()
    }
  })

  it('ante un error muestra el ErrorState y el reintento vuelve a pedir la lista', async () => {
    const fetchMock = stubFetch({ failOnce: true })
    const user = userEvent.setup()

    renderBugsView()

    expect(await screen.findByText('No se pudo cargar')).toBeDefined()
    expect(screen.queryByRole('region', { name: 'Tablero de bugs' })).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(
      await screen.findByRole('region', { name: 'Tablero de bugs' }),
    ).toBeDefined()
    // 1 shell + 1 fallo + 1 reintento
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('click en una tarjeta deja ?bug=:id en la URL', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderBugsView()
    await user.click(await screen.findByRole('button', { name: /Bug uno/ }))

    expect(router.state.location.search).toBe('?bug=b1')
    expect(router.state.location.pathname).toBe('/projects/p1/bugs')
  })

  it('Enter en una tarjeta deja ?bug=:id en la URL', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderBugsView()
    const card = await screen.findByRole('button', { name: /Bug dos/ })
    card.focus()
    await user.keyboard('{Enter}')

    expect(router.state.location.search).toBe('?bug=b2')
  })

  it('muestra "Cargando bugs…" mientras carga', async () => {
    stubFetch({ hang: true })

    renderBugsView()

    expect(await screen.findByText('Cargando bugs…')).toBeDefined()
  })
})

describe('BugsView — detalle de bug (tarea 04)', () => {
  it('abrir la URL con ?bug= muestra el diálogo con los datos del bug', async () => {
    stubFetch()

    renderBugsView('/projects/p1/bugs?bug=b1')

    const dialog = await screen.findByRole('dialog', { name: 'Bug uno' })
    expect(dialog.hasAttribute('open')).toBe(true)
    expect(within(dialog).getByText('En desarrollo')).toBeDefined()
    expect(within(dialog).getByText('Estado')).toBeDefined()
    expect(within(dialog).getByText('1.2.3')).toBeDefined()
    expect(within(dialog).getByText('Reportado por')).toBeDefined()
    expect(within(dialog).getByText('pmt-frontend-pp')).toBeDefined()
    expect(within(dialog).getByText('Creado')).toBeDefined()
    expect(
      within(dialog).getByText(formatDate('2026-11-01T09:00:00-06:00')),
    ).toBeDefined()
    expect(within(dialog).getByText('Actualizado')).toBeDefined()
    expect(
      within(dialog).getByText(formatDate('2026-11-02T10:00:00-06:00')),
    ).toBeDefined()
  })

  it('el detalle incluye las secciones Comentarios e Historial de estados', async () => {
    stubFetch({
      bugs: [makeBug()],
      comments: [
        {
          id: 'c1',
          item_kind: 'bug',
          item_id: 'b1',
          body: 'Comentario del bug',
          actor_name: 'Ana',
          actor_profile: 'pmt-frontend-pp',
          created_at: '2026-11-01T09:00:00-06:00',
        },
      ],
      history: [
        {
          id: 'h1',
          bug_id: 'b1',
          from_status: 'reported',
          to_status: 'confirmed',
          actor_name: 'Eva',
          actor_profile: 'pmt-frontend-qa',
          reason: 'Confirmado',
          changed_at: '2026-11-01T09:00:00-06:00',
        },
      ],
    })

    renderBugsView('/projects/p1/bugs?bug=b1')

    const dialog = await screen.findByRole('dialog', { name: 'Bug uno' })
    expect(
      within(dialog).getByRole('region', { name: 'Comentarios' }),
    ).toBeDefined()
    expect(
      await within(dialog).findByText('Comentario del bug'),
    ).toBeDefined()
    expect(
      within(dialog).getByRole('region', { name: 'Historial de estados' }),
    ).toBeDefined()
    expect(
      await within(dialog).findByText('Reportado → Confirmado'),
    ).toBeDefined()
    expect(within(dialog).queryByText('Comentarios e historial')).toBeNull()
  })

  it('la descripción y los pasos con HTML y saltos de línea se ven literales', async () => {
    stubFetch({
      bugs: [
        makeBug({
          description: '<b>negrita</b>\nsegunda línea',
          steps_to_reproduce: '1. abrir\n2. <script>clic</script>',
        }),
      ],
    })

    renderBugsView('/projects/p1/bugs?bug=b1')

    const dialog = await screen.findByRole('dialog', { name: 'Bug uno' })
    // getByText normaliza espacios; se compara el textContent crudo para
    // verificar que los saltos de línea y el HTML se conservan literales.
    const descripcion = within(dialog).getByRole('region', {
      name: 'Descripción',
    })
    expect(descripcion.querySelector('p')?.textContent).toBe(
      '<b>negrita</b>\nsegunda línea',
    )
    const pasos = within(dialog).getByRole('region', {
      name: 'Pasos para reproducir',
    })
    expect(pasos.querySelector('p')?.textContent).toBe(
      '1. abrir\n2. <script>clic</script>',
    )
    expect(document.querySelector('b')).toBeNull()
    expect(document.querySelector('script')).toBeNull()
  })

  it('sin reported_by la fila no aparece', async () => {
    stubFetch({ bugs: [makeBug({ id: 'b4', reported_by: undefined })] })

    renderBugsView('/projects/p1/bugs?bug=b4')

    const dialog = await screen.findByRole('dialog', { name: 'Bug uno' })
    expect(within(dialog).queryByText('Reportado por')).toBeNull()
  })

  it('cerrar con el botón quita ?bug= y devuelve el foco a la tarjeta', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderBugsView()
    const card = await screen.findByRole('button', { name: /Bug uno/ })
    await user.click(card)

    const dialog = await screen.findByRole('dialog', { name: 'Bug uno' })
    expect(dialog.hasAttribute('open')).toBe(true)

    await user.click(screen.getByRole('button', { name: 'Cerrar' }))

    expect(router.state.location.search).toBe('')
    await act(async () => {})
    expect(document.activeElement).toBe(card)
  })

  it('cerrar con Escape quita ?bug= y devuelve el foco a la tarjeta', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderBugsView()
    const card = await screen.findByRole('button', { name: /Bug uno/ })
    await user.click(card)
    await screen.findByRole('dialog', { name: 'Bug uno' })

    fireEvent(
      screen.getByRole('dialog'),
      new Event('cancel', { bubbles: true, cancelable: true }),
    )
    await act(async () => {})

    expect(router.state.location.search).toBe('')
    expect(document.activeElement).toBe(card)
  })

  it('un id que no está en la lista muestra "Bug no encontrado" y se cierra normalmente', async () => {
    stubFetch()
    const user = userEvent.setup()

    const router = renderBugsView('/projects/p1/bugs?bug=b999')

    const dialog = await screen.findByRole('dialog', {
      name: 'Bug no encontrado',
    })
    // h2 del título + párrafo del contenido
    expect(
      within(dialog).getAllByText('Bug no encontrado').length,
    ).toBeGreaterThanOrEqual(2)

    await user.click(screen.getByRole('button', { name: 'Cerrar' }))
    expect(router.state.location.search).toBe('')
  })

  it('con ?bug= mientras la lista carga solo se ve el Loading, sin diálogo', async () => {
    stubFetch({ hang: true })

    renderBugsView('/projects/p1/bugs?bug=b1')

    expect(await screen.findByText('Cargando bugs…')).toBeDefined()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('si la lista falla con ?bug= solo se ve el ErrorState, sin diálogo', async () => {
    stubFetch({ failOnce: true, bugs: [] })

    renderBugsView('/projects/p1/bugs?bug=b1')

    expect(await screen.findByText('No se pudo cargar')).toBeDefined()
    expect(screen.queryByRole('dialog')).toBeNull()
  })
})

describe('BugsView — refresco por eventos (tarea 03)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function renderWithEvents(
    connection: FakeConnection,
    initialEntry = '/projects/p1/bugs',
  ) {
    const routes: RouteObject[] = [
      {
        element: <ShellLayout />,
        children: [
          { path: '/projects/:projectId/bugs', element: <BugsView /> },
        ],
      },
    ]
    const router = createMemoryRouter(routes, { initialEntries: [initialEntry] })
    render(<RouterProvider router={router} />, {
      wrapper: ({ children }) => (
        <EventsProvider connection={connection}>{children}</EventsProvider>
      ),
    })
    return router
  }

  function stubFetchWithBugs(
    bugsResponder: (call: number) => unknown[] | { error: string; status: number },
  ) {
    let bugsCalls = 0
    const fetchMock = vi.fn((url: string) => {
      if (url === '/api/projects/p1') {
        return Promise.resolve(jsonResponse(PROJECT_P1))
      }
      if (url === '/api/projects/p1/bugs') {
        bugsCalls += 1
        const body = bugsResponder(bugsCalls)
        if (Array.isArray(body)) {
          return Promise.resolve(jsonResponse(body))
        }
        return Promise.resolve(
          jsonResponse({ error: body.error }, body.status),
        )
      }
      if (url.startsWith('/api/comments?')) {
        return Promise.resolve(jsonResponse([]))
      }
      if (url.startsWith('/api/bugs/') && url.endsWith('/status-changes')) {
        return Promise.resolve(jsonResponse([]))
      }
      return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
    })
    vi.stubGlobal('fetch', fetchMock)
    return { fetchMock, bugsCallCount: () => bugsCalls }
  }

  it('un invalidate de bug vuelve a pedir el kanban de bugs; task no', async () => {
    const { bugsCallCount } = stubFetchWithBugs(() => BUGS)
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByRole('region', { name: 'Tablero de bugs' })).toBeDefined()
    expect(bugsCallCount()).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(bugsCallCount()).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'bug' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(bugsCallCount()).toBe(2)
  })

  it('si un refresco quita el bug seleccionado, el diálogo muestra "Ya no existe"', async () => {
    stubFetchWithBugs((call) => (call === 1 ? [makeBug()] : []))
    const connection = createFakeConnection()
    renderWithEvents(connection, '/projects/p1/bugs?bug=b1')

    await act(async () => {})
    expect(screen.getByRole('dialog', { name: 'Bug uno' })).toBeDefined()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'bug' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    const dialog = screen.getByRole('dialog', { name: 'Ya no existe' })
    // h2 del título + párrafo del contenido
    expect(
      within(dialog).getAllByText('Ya no existe').length,
    ).toBeGreaterThanOrEqual(2)
  })

  it('un bug que nunca estuvo en la lista sigue mostrando "Bug no encontrado"', async () => {
    stubFetchWithBugs(() => BUGS)
    const connection = createFakeConnection()
    renderWithEvents(connection, '/projects/p1/bugs?bug=b999')

    await act(async () => {})
    expect(
      screen.getByRole('dialog', { name: 'Bug no encontrado' }),
    ).toBeDefined()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'bug' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(
      screen.getByRole('dialog', { name: 'Bug no encontrado' }),
    ).toBeDefined()
  })

  it('un error de refresco muestra ErrorState sin perder el kanban ni el diálogo', async () => {
    stubFetchWithBugs((call) =>
      call === 1 ? [makeBug()] : { error: 'No se pudo cargar', status: 500 },
    )
    const connection = createFakeConnection()
    renderWithEvents(connection, '/projects/p1/bugs?bug=b1')

    await act(async () => {})
    expect(screen.getByRole('dialog', { name: 'Bug uno' })).toBeDefined()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'bug' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(screen.getByText('No se pudo cargar')).toBeDefined()
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeDefined()
    expect(screen.getByRole('region', { name: 'Tablero de bugs' })).toBeDefined()
    expect(screen.getByRole('dialog', { name: 'Bug uno' })).toBeDefined()
  })
})
