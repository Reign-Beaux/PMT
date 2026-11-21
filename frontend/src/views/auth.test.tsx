import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { RequireAuth } from '../shell/RequireAuth'
import { AccountView } from './AccountView'
import { InvitationView } from './InvitationView'
import { LoginView } from './LoginView'

vi.mock('../realtime/EventsProvider', () => ({
  EventsProvider: ({ children }: { children: React.ReactNode }) => children,
}))

const owner = { id: 'u1', email: 'saul@x.com', username: 'saul', name: 'Saúl', is_instance_owner: true, disabled: false, created_at: '2026-11-21T10:00:00Z' }

type Handler = (method: string, path: string, body: unknown) => [number, unknown?]

function stubApi(handler: Handler) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => {
    const [status, body] = handler(init?.method ?? 'GET', url.replace(/^\/api/, ''), init?.body ? JSON.parse(String(init.body)) : undefined)
    return Promise.resolve({ ok: status < 300, status, text: () => Promise.resolve(body === undefined ? '' : JSON.stringify(body)) } as Response)
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function renderAt(path: string) {
  const router = createMemoryRouter([
    { path: '/login', element: <LoginView /> },
    { path: '/invitacion/:token', element: <InvitationView /> },
    { element: <RequireAuth />, children: [
      { path: '/projects', element: <p>proyectos</p> },
      { path: '/cuenta', element: <AccountView /> },
    ] },
  ], { initialEntries: [path] })
  render(<RouterProvider router={router} />)
  return router
}

afterEach(() => vi.unstubAllGlobals())

describe('RequireAuth', () => {
  it('sin sesión lleva a /login', async () => {
    stubApi(() => [401, { error: 'autenticación requerida' }])
    const router = renderAt('/projects')
    await screen.findByRole('heading', { name: 'Entra a PMT' })
    expect(router.state.location.pathname).toBe('/login')
  })

  it('con sesión muestra la ruta', async () => {
    stubApi(() => [200, owner])
    renderAt('/projects')
    expect(await screen.findByText('proyectos')).toBeTruthy()
  })
})

describe('LoginView', () => {
  it('entra y vuelve a la ruta pedida', async () => {
    let logged = false
    stubApi((_method, path) => {
      if (path === '/auth/login') { logged = true; return [200, owner] }
      return logged ? [200, owner] : [401, { error: 'autenticación requerida' }]
    })
    const router = renderAt('/projects')
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Correo'), 'saul@x.com')
    await user.type(screen.getByLabelText('Contraseña'), 'secreto12')
    await user.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByText('proyectos')).toBeTruthy()
    expect(router.state.location.pathname).toBe('/projects')
  })

  it('credenciales malas y usuario desactivado', async () => {
    let status = 401
    stubApi(() => [status, { error: 'x' }])
    renderAt('/login')
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Correo'), 'a@x.com')
    await user.type(screen.getByLabelText('Contraseña'), 'mala-clave')
    await user.click(screen.getByRole('button', { name: 'Entrar' }))
    expect((await screen.findByRole('alert')).textContent).toBe('Correo o contraseña incorrectos')
    status = 403
    await user.type(screen.getByLabelText('Contraseña'), 'otra-clave')
    await user.click(screen.getByRole('button', { name: 'Entrar' }))
    await waitFor(() => expect(screen.getByRole('alert').textContent).toContain('desactivado'))
  })
})

describe('InvitationView', () => {
  it('valida el username y crea la cuenta', async () => {
    const calls: unknown[] = []
    stubApi((_method, path, body) => {
      if (path === '/invitations/tok') return [200, { email: 'ana@x.com', valid: true }]
      if (path === '/invitations/tok/accept') { calls.push(body); return [201, { ...owner, username: 'ana' }] }
      return [200, { ...owner, username: 'ana', is_instance_owner: false }]
    })
    const router = renderAt('/invitacion/tok')
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Nombre'), 'Ana')
    await user.type(screen.getByLabelText('Username'), 'Ana')
    await user.type(screen.getByLabelText('Contraseña'), 'secreto12')
    await user.type(screen.getByLabelText('Confirmar contraseña'), 'secreto12')
    await user.click(screen.getByRole('button', { name: 'Crear cuenta' }))
    expect((await screen.findByRole('alert')).textContent).toContain('minúsculas')
    expect(calls).toHaveLength(0)
    await user.clear(screen.getByLabelText('Username'))
    await user.type(screen.getByLabelText('Username'), 'ana')
    await user.click(screen.getByRole('button', { name: 'Crear cuenta' }))
    await screen.findByText('proyectos')
    expect(calls).toEqual([{ name: 'Ana', username: 'ana', password: 'secreto12' }])
    expect(router.state.location.pathname).toBe('/projects')
  })

  it('una invitación vencida no muestra el formulario', async () => {
    stubApi(() => [200, { email: 'ana@x.com', valid: false }])
    renderAt('/invitacion/tok')
    expect(await screen.findByRole('heading', { name: 'Invitación no válida' })).toBeTruthy()
  })
})

describe('AccountView', () => {
  it('muestra el token nuevo una sola vez y lo quita de la vista al confirmarlo', async () => {
    const tokens: { id: string; name: string; created_at: string; last_used_at: null }[] = []
    stubApi((method, path, body) => {
      if (path === '/auth/me') return [200, owner]
      if (path === '/tokens' && method === 'POST') {
        const name = (body as { name: string }).name
        tokens.push({ id: 't1', name, created_at: '2026-11-21T10:00:00Z', last_used_at: null })
        return [201, { id: 't1', name, created_at: '2026-11-21T10:00:00Z', token: 'pmt_secreto' }]
      }
      if (path === '/tokens') return [200, tokens]
      return [200, []]
    })
    renderAt('/cuenta')
    const user = userEvent.setup()
    await screen.findByRole('heading', { name: 'Tokens de API' })
    expect(screen.getByRole('heading', { name: 'Invitaciones' })).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Usuarios' })).toBeTruthy()
    await user.type(screen.getByLabelText('Nombre'), 'laptop')
    await user.click(screen.getByRole('button', { name: 'Crear token' }))
    expect((await screen.findByLabelText('Token nuevo')).textContent).toBe('pmt_secreto')
    await screen.findByRole('button', { name: 'Revocar laptop' })
    await user.click(screen.getByRole('button', { name: 'Ya lo copié' }))
    expect(screen.queryByText('pmt_secreto')).toBeNull()
  })

  it('un usuario que no es dueño no ve invitaciones ni usuarios', async () => {
    stubApi((_method, path) => path === '/auth/me' ? [200, { ...owner, is_instance_owner: false }] : [200, []])
    renderAt('/cuenta')
    await screen.findByText('No tienes tokens.')
    expect(screen.queryByRole('heading', { name: 'Invitaciones' })).toBeNull()
    expect(screen.queryByRole('heading', { name: 'Usuarios' })).toBeNull()
  })

  it('cerrar sesión lleva a /login', async () => {
    let out = false
    stubApi((_method, path) => {
      if (path === '/auth/logout') { out = true; return [204] }
      if (path === '/auth/me') return out ? [401, { error: 'x' }] : [200, owner]
      return [200, []]
    })
    const router = renderAt('/cuenta')
    await userEvent.setup().click(await screen.findByRole('button', { name: 'Cerrar sesión' }))
    await screen.findByRole('heading', { name: 'Entra a PMT' })
    expect(router.state.location.pathname).toBe('/login')
  })
})
