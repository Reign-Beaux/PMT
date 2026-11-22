import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, Outlet, RouterProvider } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Project } from '../api/projects'
import { ReleaseControl } from '../components/members/ReleaseControl'
import { RoleBadge } from '../components/members/RoleBadge'
import { TakenBy } from '../components/members/TakenBy'
import { EventsContext } from '../realtime/eventsContext'
import { AuthContext } from '../shell/authContext'
import { createFakeConnection } from '../test/fakeConnection'
import { ProjectLayout } from '../shell/ProjectLayout'
import type { ProjectContext } from '../shell/useProjectContext'
import { MembersView } from './MembersView'
import { ProjectsView } from './ProjectsView'

const PROJECT_ID = '1b6f5b3a-9d2e-4c7f-8a1b-2c3d4e5f6a7b'
const me = { id: 'u1', email: 'saul@x.com', username: 'saul', name: 'Saúl', is_instance_owner: true, disabled: false, created_at: '2026-11-21T10:00:00Z' }
const baseProject: Project = { id: PROJECT_ID, name: 'PMT', description: '', status: 'working', product_version: '1.0.0', created_at: '2026-11-01T09:00:00Z', updated_at: '2026-11-02T10:00:00Z' }

type Handler = (method: string, path: string, body: unknown) => [number, unknown?]

function stubApi(handler: Handler) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => {
    const [status, body] = handler(init?.method ?? 'GET', url.replace(/^\/api/, ''), init?.body ? JSON.parse(String(init.body)) : undefined)
    return Promise.resolve({ ok: status < 300, status, text: () => Promise.resolve(body === undefined ? '' : JSON.stringify(body)) } as Response)
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function renderMembers(project: Project) {
  const idle = { status: 'success' as const, retry: () => {}, refreshing: false }
  const context: ProjectContext = { project, stages: idle, bugs: { ...idle, data: [] } }
  const router = createMemoryRouter([
    { element: <AuthContext.Provider value={me}><Outlet context={context} /></AuthContext.Provider>, children: [{ path: '/m', element: <MembersView /> }] },
  ], { initialEntries: ['/m'] })
  render(<RouterProvider router={router} />)
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('MembersView', () => {
  it('el owner agrega, cambia el rol y quita con confirmación', async () => {
    let members = [{ username: 'saul', name: 'Saúl', role: 'owner' }]
    const calls: [string, string, unknown][] = []
    stubApi((method, path, body) => {
      calls.push([method, path, body])
      const base = `/projects/${PROJECT_ID}/members`
      if (method === 'GET' && path === base) return [200, members]
      if (method === 'POST' && path === base) {
        const added = { ...(body as { username: string; role: string }), name: 'Ana' }
        members = [...members, added]
        return [201, added]
      }
      if (method === 'PUT' && path === `${base}/ana`) {
        members = members.map(m => m.username === 'ana' ? { ...m, role: (body as { role: string }).role } : m)
        return [200, members[1]]
      }
      if (method === 'DELETE' && path === `${base}/ana`) {
        members = members.filter(m => m.username !== 'ana')
        return [204]
      }
      return [500]
    })
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderMembers({ ...baseProject, role: 'owner' })
    const user = userEvent.setup()

    expect(await screen.findByText('(tú)')).toBeTruthy()
    await user.type(screen.getByLabelText('Username'), 'Ana')
    await user.selectOptions(screen.getByLabelText('Rol'), 'viewer')
    await user.click(screen.getByRole('button', { name: 'Agregar' }))
    expect(await screen.findByRole('status')).toHaveProperty('textContent', 'ana ahora es viewer del proyecto')
    expect(calls).toContainEqual(['POST', `/projects/${PROJECT_ID}/members`, { username: 'ana', role: 'viewer' }])

    await user.selectOptions(await screen.findByLabelText('Rol de ana'), 'editor')
    await waitFor(() => expect(calls).toContainEqual(['PUT', `/projects/${PROJECT_ID}/members/ana`, { role: 'editor' }]))

    await user.click(screen.getByRole('button', { name: 'Quitar a ana' }))
    expect(confirm).toHaveBeenCalled()
    await waitFor(() => expect(screen.queryByText('ana')).toBeNull())
    expect(calls.some(([method]) => method === 'DELETE')).toBe(true)
  })

  it('no quita si no se confirma, y el owner no tiene acciones', async () => {
    stubApi(() => [200, [{ username: 'saul', name: 'Saúl', role: 'owner' }, { username: 'ana', name: 'Ana', role: 'editor' }]])
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    renderMembers({ ...baseProject, role: 'owner' })
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Quitar a ana' }))
    expect(screen.queryByRole('button', { name: 'Quitar a saul' })).toBeNull()
    expect(screen.queryByLabelText('Rol de saul')).toBeNull()
    expect(screen.getByText('ana')).toBeTruthy()
  })

  it('muestra los errores del backend con su mensaje', async () => {
    stubApi((method) => method === 'POST'
      ? [404, { error: 'no hay un usuario activo con ese username' }]
      : [200, [{ username: 'saul', name: 'Saúl', role: 'owner' }]])
    renderMembers({ ...baseProject, role: 'owner' })
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Agregar' }))
    expect((await screen.findByRole('alert')).textContent).toBe('Escribe el username de la persona')
    await user.type(screen.getByLabelText('Username'), 'nadie')
    await user.click(screen.getByRole('button', { name: 'Agregar' }))
    await waitFor(() => expect(screen.getByRole('alert').textContent).toBe('no hay un usuario activo con ese username'))
  })

  it('un viewer o un editor solo ven la lista', async () => {
    stubApi(() => [200, [{ username: 'ana', name: 'Ana', role: 'owner' }, { username: 'saul', name: 'Saúl', role: 'viewer' }]])
    renderMembers({ ...baseProject, role: 'viewer' })
    expect(await screen.findByText('Ana')).toBeTruthy()
    expect(screen.queryByRole('form', { name: 'Agregar miembro' })).toBeNull()
    expect(screen.queryByRole('button')).toBeNull()
    expect(screen.queryByRole('combobox')).toBeNull()
    expect(screen.getByText(/Solo el owner administra/)).toBeTruthy()
  })
})

describe('ReleaseControl', () => {
  it('exige motivo, libera y se cierra', async () => {
    const onRelease = vi.fn(() => Promise.resolve())
    render(<ReleaseControl kind="stage" holder="ana" onRelease={onRelease} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Liberar' }))
    await user.click(screen.getByRole('button', { name: 'Confirmar liberación' }))
    expect(screen.getByRole('alert').textContent).toContain('motivo')
    expect(onRelease).not.toHaveBeenCalled()
    await user.type(screen.getByLabelText('Motivo'), '  ana está de vacaciones ')
    await user.click(screen.getByRole('button', { name: 'Confirmar liberación' }))
    expect(onRelease).toHaveBeenCalledWith('ana está de vacaciones')
    await waitFor(() => expect(screen.getByRole('button', { name: 'Liberar' })).toBeTruthy())
  })

  it('muestra el error del backend y sigue abierto', async () => {
    render(<ReleaseControl kind="bug" holder="ana" onRelease={() => Promise.reject(new Error('el bug no está tomado'))} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Liberar' }))
    await user.type(screen.getByLabelText('Motivo'), 'x')
    await user.click(screen.getByRole('button', { name: 'Confirmar liberación' }))
    expect((await screen.findByRole('alert')).textContent).toBe('el bug no está tomado')
    expect(screen.getByRole('form', { name: 'Liberar el bug' })).toBeTruthy()
  })
})

describe('TakenBy y RoleBadge', () => {
  it('sin taken_by no muestra nada', () => {
    const { container } = render(<TakenBy hold={{}} kind="stage" />)
    expect(container.textContent).toBe('')
  })

  it('muestra quién y desde cuándo, con el género del elemento', () => {
    render(<><TakenBy hold={{ taken_by: 'ana', taken_at: '2026-11-22T10:00:00Z' }} kind="stage" compact /><TakenBy hold={{ taken_by: 'leo' }} kind="bug" /></>)
    expect(screen.getByText(/Tomada por/).textContent).toMatch(/^Tomada por ana · desde /)
    expect(screen.getByText(/Tomado por/).textContent).toBe('Tomado por leo')
  })

  it('anuncia el rol propio', () => {
    render(<RoleBadge role="viewer" mine />)
    expect(screen.getByText('Tu rol:', { exact: false })).toBeTruthy()
    expect(screen.getByText('Viewer')).toBeTruthy()
  })
})

describe('Pérdida de acceso', () => {
  it('si el proyecto deja de responder lleva a la lista con un aviso', async () => {
    let removed = false
    stubApi((_method, path) => {
      if (path === '/projects') return [200, []]
      if (removed) return [404, { error: 'proyecto no encontrado' }]
      if (path === `/projects/${PROJECT_ID}`) return [200, { ...baseProject, role: 'viewer' }]
      if (path.endsWith('/stages') || path.endsWith('/bugs')) return [200, []]
      return [500, { error: 'sin datos' }]
    })
    const connection = createFakeConnection()
    const router = createMemoryRouter([
      { path: '/projects', element: <ProjectsView /> },
      { path: '/projects/:projectId', element: <ProjectLayout />, children: [{ index: true, element: <p>etapas</p> }] },
    ], { initialEntries: [`/projects/${PROJECT_ID}`] })
    render(<EventsContext.Provider value={connection}><RouterProvider router={router} /></EventsContext.Provider>)
    const heading = await screen.findByRole('heading', { name: 'PMT' })
    expect(within(heading.parentElement!).getByText('Viewer')).toBeTruthy()

    removed = true
    act(() => connection.emit({ type: 'invalidate', resource: 'project' }))
    await waitFor(() => expect(router.state.location.pathname).toBe('/projects'))
    expect(await screen.findByText('Ya no tienes acceso al proyecto «PMT».')).toBeTruthy()
    await userEvent.setup().click(screen.getByRole('button', { name: 'Entendido' }))
    await waitFor(() => expect(screen.queryByText(/Ya no tienes acceso/)).toBeNull())
  })

  it('un proyecto que nunca cargó muestra «no encontrado» sin redirigir', async () => {
    stubApi(() => [404, { error: 'proyecto no encontrado' }])
    const router = createMemoryRouter([
      { path: '/projects', element: <ProjectsView /> },
      { path: '/projects/:projectId', element: <ProjectLayout />, children: [{ index: true, element: <p>etapas</p> }] },
    ], { initialEntries: [`/projects/${PROJECT_ID}`] })
    render(<RouterProvider router={router} />)
    expect(await screen.findByRole('heading', { name: 'Proyecto no encontrado' })).toBeTruthy()
    expect(router.state.location.pathname).toBe(`/projects/${PROJECT_ID}`)
  })
})
