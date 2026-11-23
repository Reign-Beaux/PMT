import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { RequireAuth } from '../shell/RequireAuth'
import { CliLoginView } from './CliLoginView'
import { LoginView } from './LoginView'

vi.mock('../realtime/EventsProvider', () => ({
  EventsProvider: ({ children }: { children: React.ReactNode }) => children,
}))

const owner = { id: 'u1', email: 'saul@x.com', username: 'saul', name: 'Saúl', is_instance_owner: true, disabled: false, created_at: '2026-11-21T10:00:00Z' }
const pending = { code: 'ABCD-1234', machine: 'laptop', token_name: 'pmt laptop', status: 'pending', expires_at: '2026-11-22T10:10:00Z' }

type Handler = (method: string, path: string) => [number, unknown?]

function stubApi(handler: Handler) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => {
    const [status, body] = handler(init?.method ?? 'GET', url.replace(/^\/api/, ''))
    return Promise.resolve({ ok: status < 300, status, text: () => Promise.resolve(body === undefined ? '' : JSON.stringify(body)) } as Response)
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function renderAt(path: string) {
  const router = createMemoryRouter([
    { path: '/login', element: <LoginView /> },
    { element: <RequireAuth />, children: [
      { path: '/cli', element: <CliLoginView /> },
      { path: '/projects', element: <p>proyectos</p> },
    ] },
  ], { initialEntries: [path] })
  render(<RouterProvider router={router} />)
  return router
}

afterEach(() => vi.unstubAllGlobals())

describe('CliLoginView', () => {
  it('sin sesión pasa por el login y vuelve con el código', async () => {
    let logged = false
    stubApi((_method, path) => {
      if (path === '/auth/login') { logged = true; return [200, owner] }
      if (!logged) return [401, { error: 'autenticación requerida' }]
      if (path === '/auth/me') return [200, owner]
      if (path === '/cli/login/ABCD-1234') return [200, pending]
      return [404, { error: 'no' }]
    })
    const router = renderAt('/cli?codigo=ABCD-1234')
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Correo'), 'saul@x.com')
    await user.type(screen.getByLabelText('Contraseña'), 'secreto12')
    await user.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByRole('heading', { name: 'Autorizar el CLI pmt' })).toBeTruthy()
    expect(router.state.location.pathname + router.state.location.search).toBe('/cli?codigo=ABCD-1234')
    expect(screen.getByLabelText('Código').textContent).toBe('ABCD-1234')
    expect(screen.getByText('laptop')).toBeTruthy()
    expect(screen.getByText('pmt laptop')).toBeTruthy()
  })

  it('autoriza', async () => {
    const fetchMock = stubApi((method, path) => {
      if (path === '/auth/me') return [200, owner]
      if (method === 'POST' && path === '/cli/login/ABCD-1234/authorize') return [200, { ...pending, status: 'authorized' }]
      return [200, pending]
    })
    renderAt('/cli?codigo=ABCD-1234')
    await userEvent.setup().click(await screen.findByRole('button', { name: 'Autorizar' }))
    expect(await screen.findByRole('heading', { name: 'Listo' })).toBeTruthy()
    expect(screen.getByText(/recibirá el token «pmt laptop»/)).toBeTruthy()
    expect(fetchMock.mock.calls.some(([url, init]) => url === '/api/cli/login/ABCD-1234/authorize' && init?.method === 'POST')).toBe(true)
  })

  it('cancela', async () => {
    stubApi((method, path) => {
      if (path === '/auth/me') return [200, owner]
      if (method === 'POST' && path === '/cli/login/ABCD-1234/cancel') return [200, { ...pending, status: 'cancelled' }]
      return [200, pending]
    })
    renderAt('/cli?codigo=ABCD-1234')
    await userEvent.setup().click(await screen.findByRole('button', { name: 'Cancelar' }))
    expect(await screen.findByRole('heading', { name: 'Inicio de sesión cancelado' })).toBeTruthy()
  })

  it('código caducado, ya usado o inexistente', async () => {
    stubApi((_method, path) => {
      if (path === '/auth/me') return [200, owner]
      if (path === '/cli/login/WXYZ-0000') return [200, { ...pending, code: 'WXYZ-0000', status: 'expired' }]
      return [404, { error: 'código no encontrado' }]
    })
    renderAt('/cli?codigo=WXYZ-0000')
    expect(await screen.findByRole('heading', { name: 'El código caducó' })).toBeTruthy()
  })

  it('código que no existe o falta', async () => {
    stubApi((_method, path) => path === '/auth/me' ? [200, owner] : [404, { error: 'código no encontrado' }])
    renderAt('/cli?codigo=NADA-0000')
    expect(await screen.findByRole('heading', { name: 'Código no encontrado' })).toBeTruthy()
  })

  it('si caducó mientras decidía, muestra el error y el estado nuevo', async () => {
    let expired = false
    stubApi((method, path) => {
      if (path === '/auth/me') return [200, owner]
      if (method === 'POST') { expired = true; return [409, { error: 'el código ya se usó, se canceló o caducó' }] }
      return [200, { ...pending, status: expired ? 'expired' : 'pending' }]
    })
    renderAt('/cli?codigo=ABCD-1234')
    await userEvent.setup().click(await screen.findByRole('button', { name: 'Autorizar' }))
    await waitFor(() => expect(screen.getByRole('heading', { name: 'El código caducó' })).toBeTruthy())
  })
})
