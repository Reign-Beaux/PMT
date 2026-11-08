import { act } from 'react'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { routes } from './routes'
import { useDetailSelection } from './useDetailSelection'

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
  product_version: '',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:00:00-06:00',
}

const EMPTY_PROGRESS = {
  project: {
    id: 'p1',
    name: 'Proyecto Uno',
    status: 'working',
    last_change_at: '2026-11-01T09:00:00-06:00',
  },
  stages: [],
  open_bugs: { counts: {}, items: [] },
  signals: [],
}

const STAGE_S2 = {
  id: 's2',
  project_id: 'p1',
  name: 'Etapa Dos',
  description: '',
  position: 1,
  status: 'in_dev',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:00:00-06:00',
}

const TASK_ABC = {
  id: 'abc',
  stage_id: 's2',
  title: 'Tarea Abc',
  description: '',
  status: 'in_dev',
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

// Las rutas con projectId cargan el proyecto (ShellLayout) y, en
// /projects/:id, también las etapas y el avance (StageList); la ruta de etapa
// carga además sus tareas (StageView).
beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, _init?: RequestInit) => {
      if (url === '/api/projects') {
        return Promise.resolve(jsonResponse([PROJECT_P1]))
      }
      if (url === '/api/projects/p1') {
        return Promise.resolve(jsonResponse(PROJECT_P1))
      }
      if (url === '/api/projects/p1/stages') {
        return Promise.resolve(jsonResponse([STAGE_S2]))
      }
      if (url === '/api/projects/p1/progress') {
        return Promise.resolve(jsonResponse(EMPTY_PROGRESS))
      }
      if (url === '/api/projects/p1/bugs') {
        return Promise.resolve(jsonResponse([]))
      }
      if (url === '/api/stages/s2/tasks') {
        return Promise.resolve(jsonResponse([TASK_ABC]))
      }
      return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
    }),
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

function renderAt(initialEntries: string[]) {
  const router = createMemoryRouter(routes, { initialEntries })
  render(<RouterProvider router={router} />)
  return router
}

function SelectionProbe() {
  const { selection, select } = useDetailSelection()
  return (
    <div>
      <output data-testid="selection">
        {selection ? `${selection.kind}:${selection.id}` : 'sin selección'}
      </output>
      <button
        type="button"
        onClick={() => select({ kind: 'task', id: 't1' })}
      >
        Seleccionar tarea t1
      </button>
    </div>
  )
}

describe('rutas', () => {
  it('/ renderiza la vista de Proyectos', async () => {
    renderAt(['/'])
    expect(screen.getByRole('heading', { level: 1, name: 'Proyectos' })).toBeDefined()
    expect(await screen.findByText('Proyecto Uno')).toBeDefined()
  })

  it('/projects/:projectId renderiza el encabezado con el nombre del proyecto', async () => {
    renderAt(['/projects/p1'])
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Proyecto Uno' }),
    ).toBeDefined()
  })

  it('/projects/:projectId/stages/:stageId renderiza la vista de Etapa con su tablero', async () => {
    renderAt(['/projects/p1/stages/s2'])
    expect(
      await screen.findByRole('heading', { level: 2, name: 'Etapa Dos' }),
    ).toBeDefined()
    expect(
      screen.getByRole('region', { name: 'Tablero de la etapa' }),
    ).toBeDefined()
    expect(screen.getByRole('button', { name: /Tarea Abc/ })).toBeDefined()
  })

  it('/projects/:projectId/bugs renderiza la vista de Bugs con su tablero', async () => {
    renderAt(['/projects/p1/bugs'])
    expect(await screen.findByRole('heading', { level: 2, name: 'Bugs' })).toBeDefined()
    expect(
      await screen.findByRole('region', { name: 'Tablero de bugs' }),
    ).toBeDefined()
  })

  it('una ruta desconocida muestra "Página no encontrada"', () => {
    renderAt(['/no/existe'])
    expect(
      screen.getByRole('heading', { level: 1, name: 'Página no encontrada' }),
    ).toBeDefined()
  })
})

describe('selección por URL', () => {
  it('entrada directa con ?task=abc conserva la selección en la URL', async () => {
    const router = renderAt(['/projects/p1/stages/s2?task=abc'])
    expect(
      await screen.findByRole('heading', { level: 2, name: 'Etapa Dos' }),
    ).toBeDefined()
    expect(router.state.location.search).toBe('?task=abc')
  })

  it('entrada directa con ?bug=xyz conserva la selección y abre el detalle', async () => {
    const router = renderAt(['/projects/p1/bugs?bug=xyz'])
    expect(
      await screen.findByRole('dialog', { name: 'Bug no encontrado' }),
    ).toBeDefined()
    expect(router.state.location.search).toBe('?bug=xyz')
  })

  it('?task y ?bug a la vez no rompen la vista de etapa', async () => {
    const router = renderAt(['/projects/p1/stages/s2?task=abc&bug=xyz'])
    expect(
      await screen.findByRole('heading', { level: 2, name: 'Etapa Dos' }),
    ).toBeDefined()
    expect(router.state.location.search).toBe('?task=abc&bug=xyz')
  })
})

describe('navegación', () => {
  it('clic en un enlace cambia de vista', async () => {
    const user = userEvent.setup()
    const router = renderAt(['/projects/p1'])
    // Con el Shell montado hay dos enlaces "Bugs" (pestaña y vista); cualquiera sirve
    const bugsLinks = screen.getAllByRole('link', { name: 'Bugs' })
    await user.click(bugsLinks[0])
    expect(await screen.findByRole('heading', { level: 2, name: 'Bugs' })).toBeDefined()
    expect(router.state.location.pathname).toBe('/projects/p1/bugs')
  })

  it('select() y navigate(-1)/navigate(1) restauran ruta y selección', async () => {
    const user = userEvent.setup()
    const router = createMemoryRouter(
      [...routes, { path: '/probe', element: <SelectionProbe /> }],
      { initialEntries: ['/probe'] },
    )
    render(<RouterProvider router={router} />)

    expect(screen.getByTestId('selection').textContent).toBe('sin selección')

    await user.click(screen.getByRole('button', { name: 'Seleccionar tarea t1' }))
    expect(screen.getByTestId('selection').textContent).toBe('task:t1')
    expect(router.state.location.search).toBe('?task=t1')

    await act(() => router.navigate('/projects/p1'))
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Proyecto Uno' }),
    ).toBeDefined()

    await act(() => router.navigate(-1))
    expect(screen.getByTestId('selection').textContent).toBe('task:t1')

    await act(() => router.navigate(-1))
    expect(screen.getByTestId('selection').textContent).toBe('sin selección')
    expect(router.state.location.pathname).toBe('/probe')

    await act(() => router.navigate(1))
    expect(screen.getByTestId('selection').textContent).toBe('task:t1')
  })
})
