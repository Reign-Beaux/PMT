import { afterEach, describe, expect, it, vi } from 'vitest'
import { createToken, getMe, login, logout, revokeToken } from './auth'
import { get, UNAUTHORIZED_EVENT } from './http'

function response(status: number, body?: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(body === undefined ? '' : JSON.stringify(body)),
  } as unknown as Response
}

afterEach(() => vi.unstubAllGlobals())

describe('auth api', () => {
  it('login hace POST con JSON y sin Authorization', async () => {
    const fetchMock = vi.fn((_url: string, _init?: RequestInit) => Promise.resolve(response(200, { username: 'ana' })))
    vi.stubGlobal('fetch', fetchMock)

    await login('ana@x.com', 'secreto12')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/auth/login')
    expect(init?.method).toBe('POST')
    expect(init?.body).toBe(JSON.stringify({ email: 'ana@x.com', password: 'secreto12' }))
    expect(init?.headers).toEqual({ Accept: 'application/json', 'Content-Type': 'application/json' })
  })

  it('logout y revocar resuelven undefined con 204', async () => {
    const fetchMock = vi.fn((_url: string, _init?: RequestInit) => Promise.resolve(response(204)))
    vi.stubGlobal('fetch', fetchMock)

    await expect(logout()).resolves.toBeUndefined()
    await expect(revokeToken('t 1')).resolves.toBeUndefined()
    expect(fetchMock.mock.calls[1][0]).toBe('/api/tokens/t%201')
    expect(fetchMock.mock.calls[1][1]?.method).toBe('DELETE')
  })

  it('crear token devuelve el token una vez', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(201, { id: '1', name: 'laptop', created_at: 'x', token: 'pmt_abc' }))))
    await expect(createToken('laptop')).resolves.toMatchObject({ token: 'pmt_abc' })
  })

  it('un 401 de la API avisa que la sesión venció; uno de /auth no', async () => {
    const listener = vi.fn()
    window.addEventListener(UNAUTHORIZED_EVENT, listener)
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(401, { error: 'sesión no válida' }))))

    await expect(get('/projects')).rejects.toMatchObject({ status: 401, message: 'sesión no válida' })
    await expect(getMe(new AbortController().signal)).rejects.toMatchObject({ status: 401 })

    expect(listener).toHaveBeenCalledTimes(1)
    window.removeEventListener(UNAUTHORIZED_EVENT, listener)
  })
})
