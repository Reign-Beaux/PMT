import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { RouteObject } from 'react-router'
import { createMemoryRouter, RouterProvider, useParams } from 'react-router'
import { EventsProvider } from '../realtime/EventsProvider'
import { createFakeConnection } from '../test/fakeConnection'
import type { FakeConnection } from '../test/fakeConnection'
import { ProjectsView } from './ProjectsView'

const PROJECT_A = {
  id: 'p-a',
  name: 'Proyecto Alfa',
  description: 'Primer proyecto',
  status: 'working',
  product_version: '1.2.0',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:00:00-06:00',
}

const PROJECT_B = {
  id: 'p-b',
  name: 'Proyecto Beta',
  description: 'Segundo proyecto',
  status: 'in_dev',
  product_version: '0.1.0',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:00:00-06:00',
}

function jsonResponse(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () =>
      Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
  } as unknown as Response
}

function stubFetch(fetchMock: ReturnType<typeof vi.fn>) {
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function stubProjects(body: unknown, status = 200) {
  return stubFetch(
    vi.fn((_url: string, _init?: RequestInit) =>
      Promise.resolve(jsonResponse(body, status)),
    ),
  )
}

function RouteProbe() {
  const { projectId } = useParams()
  return <p>Abierto: {projectId}</p>
}

function renderProjects() {
  const routes: RouteObject[] = [
    { path: '/', element: <ProjectsView /> },
    { path: '/projects/:projectId', element: <RouteProbe /> },
  ]
  const router = createMemoryRouter(routes, { initialEntries: ['/'] })
  const view = render(<RouterProvider router={router} />)
  return { router, ...view }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ProjectsView', () => {
  it('con dos proyectos muestra nombre, versión y estado en español', async () => {
    stubProjects([PROJECT_A, PROJECT_B])

    renderProjects()
    expect(screen.getByText('Cargando proyectos…')).toBeDefined()

    expect(await screen.findByText('Proyecto Alfa')).toBeDefined()
    expect(screen.getByText('Versión 1.2.0')).toBeDefined()
    expect(screen.getByText('En operación')).toBeDefined()
    expect(screen.getByText('Proyecto Beta')).toBeDefined()
    expect(screen.getByText('Versión 0.1.0')).toBeDefined()
    expect(screen.getByText('En desarrollo')).toBeDefined()
  })

  it('con lista vacía muestra "No hay proyectos"', async () => {
    stubProjects([])

    renderProjects()

    expect(await screen.findByText('No hay proyectos')).toBeDefined()
  })

  it('con fallo muestra el error y Reintentar carga la lista', async () => {
    const fetchMock = stubFetch(
      vi
        .fn()
        .mockResolvedValueOnce(
          jsonResponse({ error: 'El servidor falló' }, 500),
        )
        .mockResolvedValueOnce(jsonResponse([PROJECT_A])),
    )
    const user = userEvent.setup()

    renderProjects()

    expect(await screen.findByText('El servidor falló')).toBeDefined()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(await screen.findByText('Proyecto Alfa')).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('click en una tarjeta navega a /projects/:id', async () => {
    stubProjects([PROJECT_A, PROJECT_B])
    const user = userEvent.setup()

    const { router } = renderProjects()
    await user.click(
      await screen.findByRole('button', { name: /Proyecto Alfa/ }),
    )

    expect(await screen.findByText('Abierto: p-a')).toBeDefined()
    expect(router.state.location.pathname).toBe('/projects/p-a')
  })

  it('Enter en una tarjeta navega a /projects/:id', async () => {
    stubProjects([PROJECT_A, PROJECT_B])
    const user = userEvent.setup()

    const { router } = renderProjects()
    const card = await screen.findByRole('button', { name: /Proyecto Beta/ })
    card.focus()
    await user.keyboard('{Enter}')

    expect(await screen.findByText('Abierto: p-b')).toBeDefined()
    expect(router.state.location.pathname).toBe('/projects/p-b')
  })

  it('al desmontar durante la carga aborta la petición sin errores', async () => {
    let capturedSignal: AbortSignal | undefined
    const fetchMock = stubFetch(
      vi.fn((_url: string, init?: RequestInit) => {
        capturedSignal = init?.signal ?? undefined
        return new Promise<Response>(() => {})
      }),
    )

    const { unmount } = renderProjects()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))
    expect(capturedSignal?.aborted).toBe(false)

    unmount()

    expect(capturedSignal?.aborted).toBe(true)
  })
})

describe('ProjectsView — refresco por eventos (tarea 03)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function renderWithEvents(connection: FakeConnection) {
    const routes: RouteObject[] = [{ path: '/', element: <ProjectsView /> }]
    const router = createMemoryRouter(routes, { initialEntries: ['/'] })
    render(<RouterProvider router={router} />, {
      wrapper: ({ children }) => (
        <EventsProvider connection={connection}>{children}</EventsProvider>
      ),
    })
  }

  it('un invalidate de project vuelve a pedir los proyectos; otro recurso no', async () => {
    const fetchMock = stubProjects([PROJECT_A])
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByText('Proyecto Alfa')).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'bug' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    expect(fetchMock).toHaveBeenCalledTimes(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'project' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(screen.getByText('Proyecto Alfa')).toBeDefined()
  })

  it('un error de refresco muestra ErrorState con Reintentar sin perder la lista previa', async () => {
    let calls = 0
    stubFetch(
      vi.fn((_url: string, _init?: RequestInit) => {
        calls += 1
        return Promise.resolve(
          calls === 2
            ? jsonResponse({ error: 'El servidor falló' }, 500)
            : jsonResponse([PROJECT_A]),
        )
      }),
    )
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByText('Proyecto Alfa')).toBeDefined()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'project' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(screen.getByText('El servidor falló')).toBeDefined()
    const retry = screen.getByRole('button', { name: 'Reintentar' })
    expect(screen.getByText('Proyecto Alfa')).toBeDefined()

    await act(async () => {
      fireEvent.click(retry)
    })
    expect(calls).toBe(3)
    expect(screen.queryByText('El servidor falló')).toBeNull()
    expect(screen.getByText('Proyecto Alfa')).toBeDefined()
  })
})
