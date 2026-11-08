import { afterEach, describe, expect, it, vi } from 'vitest'
import { get, HttpError } from './http'

function mockResponse(status: number, text: string): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(text),
  } as unknown as Response
}

function fetchThatRejectsOnAbort(): ReturnType<typeof vi.fn> {
  return vi.fn(
    (_url: string, init: RequestInit) =>
      new Promise<Response>((_resolve, reject) => {
        init.signal?.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'))
        })
      }),
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('get — éxito y cabeceras', () => {
  it('llama a fetch con /api/x y Accept, sin Authorization, y devuelve el JSON', async () => {
    const fetchMock = vi.fn((_url: string, _init?: RequestInit) =>
      Promise.resolve(mockResponse(200, JSON.stringify({ hola: 'mundo' }))),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await get<{ hola: string }>('/x')

    expect(data).toEqual({ hola: 'mundo' })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/x',
      expect.objectContaining({ headers: { Accept: 'application/json' } }),
    )
    const init = fetchMock.mock.calls[0][1] as { headers: Record<string, string> }
    expect(init.headers).not.toHaveProperty('Authorization')
  })
})

describe('get — errores HTTP', () => {
  it('404 con {"error":...} rechaza con kind http, status 404 y ese mensaje', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(
          mockResponse(404, JSON.stringify({ error: 'Proyecto no encontrado' })),
        ),
      ),
    )

    const promise = get('/projects/1')

    await expect(promise).rejects.toBeInstanceOf(HttpError)
    await expect(promise).rejects.toMatchObject({
      kind: 'http',
      status: 404,
      message: 'Proyecto no encontrado',
    })
  })

  it('500 con cuerpo HTML rechaza con kind http y mensaje genérico', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(mockResponse(500, '<html><body>Oops</body></html>'))),
    )

    await expect(get('/x')).rejects.toMatchObject({
      kind: 'http',
      status: 500,
      message: 'Error del servidor (500)',
    })
  })
})

describe('get — red y JSON inválido', () => {
  it('fetch rechazado produce kind network', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.reject(new TypeError('fetch failed'))),
    )

    await expect(get('/x')).rejects.toMatchObject({
      kind: 'network',
      message: 'No se pudo conectar con el servidor',
    })
  })

  it.each([
    ['cuerpo vacío', ''],
    ['cuerpo HTML', '<!doctype html><html></html>'],
  ])('2xx con %s produce kind invalid_json', async (_caso, text) => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(mockResponse(200, text))),
    )

    await expect(get('/x')).rejects.toMatchObject({
      kind: 'invalid_json',
      message: 'Respuesta no válida del servidor',
    })
  })
})

describe('get — timeout', () => {
  it('sin respuesta a los 15 s rechaza con kind timeout', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('fetch', fetchThatRejectsOnAbort())

    const promise = get('/x')
    const assertion = expect(promise).rejects.toMatchObject({
      kind: 'timeout',
      message: 'El servidor tardó demasiado en responder',
    })
    await vi.advanceTimersByTimeAsync(15_000)

    await assertion
    expect(vi.getTimerCount()).toBe(0)
  })

  it('antes de 15 s no rechaza', async () => {
    vi.useFakeTimers()
    let resolveResponse: ((value: Response) => void) | undefined
    const fetchMock = vi.fn(
      (_url: string, init: RequestInit) =>
        new Promise<Response>((resolve, reject) => {
          resolveResponse = resolve
          init.signal?.addEventListener('abort', () => {
            reject(new DOMException('Aborted', 'AbortError'))
          })
        }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const promise = get<{ ok: boolean }>('/x')
    let settled = false
    void promise.then(
      () => (settled = true),
      () => (settled = true),
    )

    await vi.advanceTimersByTimeAsync(14_999)
    expect(settled).toBe(false)

    resolveResponse?.(mockResponse(200, JSON.stringify({ ok: true })))
    await expect(promise).resolves.toEqual({ ok: true })
  })
})

describe('get — cancelación externa', () => {
  it('señal ya abortada rechaza con kind aborted sin llamar a fetch', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const controller = new AbortController()
    controller.abort()

    await expect(get('/x', { signal: controller.signal })).rejects.toMatchObject({
      kind: 'aborted',
      message: 'Consulta cancelada',
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('abort durante la consulta rechaza con kind aborted, no timeout ni network', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('fetch', fetchThatRejectsOnAbort())
    const controller = new AbortController()

    const promise = get('/x', { signal: controller.signal })
    const assertion = expect(promise).rejects.toMatchObject({ kind: 'aborted' })
    controller.abort()

    await assertion
    expect(vi.getTimerCount()).toBe(0)
  })

  it('abort externo durante la lectura del cuerpo rechaza con kind aborted', async () => {
    const controller = new AbortController()
    let textCalls = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({
          ok: true,
          status: 200,
          text: () => {
            textCalls += 1
            controller.abort()
            return Promise.reject(new DOMException('Aborted', 'AbortError'))
          },
        } as unknown as Response),
      ),
    )

    await expect(get('/x', { signal: controller.signal })).rejects.toMatchObject({
      kind: 'aborted',
    })
    expect(textCalls).toBe(1)
  })
})

describe('get — limpieza de temporizador y listener', () => {
  it('tras éxito, error, timeout y abort no quedan temporizadores y se remueve el listener', async () => {
    vi.useFakeTimers()

    // Éxito
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(mockResponse(200, '{"a":1}'))))
    const okController = new AbortController()
    const okSpy = vi.spyOn(okController.signal, 'removeEventListener')
    await get('/x', { signal: okController.signal })
    expect(vi.getTimerCount()).toBe(0)
    expect(okSpy).toHaveBeenCalledWith('abort', expect.any(Function))

    // Error HTTP
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(mockResponse(404, '{}'))))
    const errController = new AbortController()
    const errSpy = vi.spyOn(errController.signal, 'removeEventListener')
    await expect(get('/x', { signal: errController.signal })).rejects.toMatchObject({
      kind: 'http',
    })
    expect(vi.getTimerCount()).toBe(0)
    expect(errSpy).toHaveBeenCalledWith('abort', expect.any(Function))

    // Timeout
    vi.stubGlobal('fetch', fetchThatRejectsOnAbort())
    const toController = new AbortController()
    const toSpy = vi.spyOn(toController.signal, 'removeEventListener')
    const timeoutPromise = get('/x', { signal: toController.signal })
    const timeoutAssertion = expect(timeoutPromise).rejects.toMatchObject({
      kind: 'timeout',
    })
    await vi.advanceTimersByTimeAsync(15_000)
    await timeoutAssertion
    expect(vi.getTimerCount()).toBe(0)
    expect(toSpy).toHaveBeenCalledWith('abort', expect.any(Function))

    // Abort externo
    vi.stubGlobal('fetch', fetchThatRejectsOnAbort())
    const abController = new AbortController()
    const abSpy = vi.spyOn(abController.signal, 'removeEventListener')
    const abortPromise = get('/x', { signal: abController.signal })
    const abortAssertion = expect(abortPromise).rejects.toMatchObject({
      kind: 'aborted',
    })
    abController.abort()
    await abortAssertion
    expect(vi.getTimerCount()).toBe(0)
    expect(abSpy).toHaveBeenCalledWith('abort', expect.any(Function))
  })
})
