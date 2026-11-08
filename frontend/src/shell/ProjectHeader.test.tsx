import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { ReactElement, ReactNode } from 'react'
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { RouteObject } from 'react-router'
import { createMemoryRouter, MemoryRouter, RouterProvider } from 'react-router'
import { HttpError } from '../api/http'
import type { Project } from '../api/projects'
import { EventsProvider } from '../realtime/EventsProvider'
import { createFakeConnection } from '../test/fakeConnection'
import type { FakeConnection } from '../test/fakeConnection'
import { ProjectHeader } from './ProjectHeader'
import { ShellLayout } from './ShellLayout'

const PROJECT: Project = {
  id: 'p1',
  name: 'Proyecto Uno',
  description: 'Descripción del proyecto',
  status: 'working',
  product_version: '1.0.0',
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

function renderHeader(ui: ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>)
}

function renderShell() {
  const routes: RouteObject[] = [
    {
      element: <ShellLayout />,
      children: [
        {
          path: '/projects/:projectId',
          element: <p>Contenido del proyecto</p>,
        },
      ],
    },
  ]
  const router = createMemoryRouter(routes, { initialEntries: ['/projects/p1'] })
  render(<RouterProvider router={router} />)
  return router
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ProjectHeader', () => {
  it('en carga muestra "Cargando proyecto…"', () => {
    renderHeader(<ProjectHeader status="loading" onRetry={() => {}} />)
    expect(screen.getByText('Cargando proyecto…')).toBeDefined()
  })

  it('con datos muestra nombre (h1), estado en español, versión y descripción', () => {
    renderHeader(
      <ProjectHeader status="success" project={PROJECT} onRetry={() => {}} />,
    )
    expect(
      screen.getByRole('heading', { level: 1, name: 'Proyecto Uno' }),
    ).toBeDefined()
    expect(screen.getByText('En operación')).toBeDefined()
    expect(screen.getByText('Versión 1.0.0')).toBeDefined()
    expect(screen.getByText('Descripción del proyecto')).toBeDefined()
  })

  it('con descripción vacía no pinta el párrafo', () => {
    const { container } = renderHeader(
      <ProjectHeader
        status="success"
        project={{ ...PROJECT, description: '' }}
        onRetry={() => {}}
      />,
    )
    expect(container.querySelectorAll('p')).toHaveLength(0)
  })

  it('con 404 muestra "Proyecto no encontrado" y el enlace a /', () => {
    renderHeader(
      <ProjectHeader
        status="error"
        error={new HttpError('http', 'Proyecto no encontrado', { status: 404 })}
        onRetry={() => {}}
      />,
    )
    expect(screen.getByText('Proyecto no encontrado')).toBeDefined()
    const link = screen.getByRole('link', { name: 'Volver a proyectos' })
    expect(link.getAttribute('href')).toBe('/')
  })

  it('con error 500 muestra el error y Reintentar llama a onRetry', async () => {
    const onRetry = vi.fn()
    const user = userEvent.setup()
    renderHeader(
      <ProjectHeader
        status="error"
        error={new HttpError('http', 'Error del servidor (500)', { status: 500 })}
        onRetry={onRetry}
      />,
    )
    expect(screen.getByText('Error del servidor (500)')).toBeDefined()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('una descripción "<b>x</b>" se muestra literal y no crea elemento <b>', () => {
    renderHeader(
      <ProjectHeader
        status="success"
        project={{ ...PROJECT, description: '<b>x</b>' }}
        onRetry={() => {}}
      />,
    )
    expect(screen.getByText('<b>x</b>')).toBeDefined()
    expect(document.querySelector('b')).toBeNull()
  })
})

describe('ShellLayout con ProjectHeader', () => {
  it('con datos el shell muestra el nombre y pinta el Outlet', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((_url: string, _init?: RequestInit) =>
        Promise.resolve(jsonResponse(PROJECT)),
      ),
    )

    renderShell()

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Proyecto Uno' }),
    ).toBeDefined()
    // h1 del encabezado + span del contexto del shell
    expect(screen.getAllByText('Proyecto Uno')).toHaveLength(2)
    expect(screen.getByText('Contenido del proyecto')).toBeDefined()
  })

  it('con 404 no pinta el Outlet', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((_url: string, _init?: RequestInit) =>
        Promise.resolve(jsonResponse({ error: 'Proyecto no encontrado' }, 404)),
      ),
    )

    renderShell()

    expect(await screen.findByText('Proyecto no encontrado')).toBeDefined()
    expect(screen.queryByText('Contenido del proyecto')).toBeNull()
  })

  it('con error 500 Reintentar vuelve a cargar', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({ error: 'Error del servidor (500)' }, 500),
      )
      .mockResolvedValueOnce(jsonResponse(PROJECT))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()

    renderShell()

    expect(await screen.findByText('Error del servidor (500)')).toBeDefined()
    expect(screen.queryByText('Contenido del proyecto')).toBeNull()

    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Proyecto Uno' }),
    ).toBeDefined()
    expect(screen.getByText('Contenido del proyecto')).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })
})

describe('ProjectHeader — refresco por eventos (tarea 03)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function renderShellWithEvents(connection: FakeConnection) {
    const routes: RouteObject[] = [
      {
        element: <ShellLayout />,
        children: [
          {
            path: '/projects/:projectId',
            element: <p>Contenido del proyecto</p>,
          },
        ],
      },
    ]
    const router = createMemoryRouter(routes, {
      initialEntries: ['/projects/p1'],
    })
    render(<RouterProvider router={router} />, {
      wrapper: ({ children }: { children: ReactNode }) => (
        <EventsProvider connection={connection}>{children}</EventsProvider>
      ),
    })
  }

  it('un invalidate de project vuelve a pedir el encabezado; task no', async () => {
    const fetchMock = vi.fn((_url: string, _init?: RequestInit) =>
      Promise.resolve(jsonResponse(PROJECT)),
    )
    vi.stubGlobal('fetch', fetchMock)
    const connection = createFakeConnection()
    renderShellWithEvents(connection)

    await act(async () => {})
    expect(
      screen.getByRole('heading', { level: 1, name: 'Proyecto Uno' }),
    ).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(fetchMock).toHaveBeenCalledTimes(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'project' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('un error de refresco muestra ErrorState conservando el nombre y el Outlet', async () => {
    let calls = 0
    const fetchMock = vi.fn((_url: string, _init?: RequestInit) => {
      calls += 1
      return Promise.resolve(
        calls === 1
          ? jsonResponse(PROJECT)
          : jsonResponse({ error: 'Error del servidor (500)' }, 500),
      )
    })
    vi.stubGlobal('fetch', fetchMock)
    const connection = createFakeConnection()
    renderShellWithEvents(connection)

    await act(async () => {})
    expect(
      screen.getByRole('heading', { level: 1, name: 'Proyecto Uno' }),
    ).toBeDefined()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'project' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(screen.getByText('Error del servidor (500)')).toBeDefined()
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeDefined()
    expect(
      screen.getByRole('heading', { level: 1, name: 'Proyecto Uno' }),
    ).toBeDefined()
    expect(screen.getByText('Contenido del proyecto')).toBeDefined()
  })
})
