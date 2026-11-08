import { describe, expect, it, vi, afterEach, beforeEach } from 'vitest'
import type { ReactNode } from 'react'
import { act, renderHook, waitFor } from '@testing-library/react'
import { HttpError } from '../api/http'
import { EventsProvider } from '../realtime/EventsProvider'
import { createFakeConnection } from '../test/fakeConnection'
import type { FakeConnection } from '../test/fakeConnection'
import { useAsync } from './useAsync'

describe('useAsync', () => {
  it('empieza en loading y pasa a success con los datos', async () => {
    const { result } = renderHook(() =>
      useAsync(() => Promise.resolve('dato'), []),
    )

    expect(result.current.status).toBe('loading')

    await waitFor(() => expect(result.current.status).toBe('success'))
    expect(result.current.data).toBe('dato')
    expect(result.current.error).toBeUndefined()
  })

  it('ante un fallo pasa a error y retry vuelve a cargar', async () => {
    let calls = 0
    const { result } = renderHook(() =>
      useAsync(async () => {
        calls += 1
        if (calls === 1) {
          throw new HttpError('network', 'No se pudo conectar con el servidor')
        }
        return 'ok'
      }, []),
    )

    await waitFor(() => expect(result.current.status).toBe('error'))
    expect(result.current.error?.kind).toBe('network')
    expect(result.current.error?.message).toBe(
      'No se pudo conectar con el servidor',
    )

    act(() => result.current.retry())
    expect(result.current.status).toBe('loading')

    await waitFor(() => expect(result.current.status).toBe('success'))
    expect(result.current.data).toBe('ok')
    expect(calls).toBe(2)
  })

  it('los errores que no son HttpError se envuelven como network', async () => {
    const { result } = renderHook(() =>
      useAsync(async () => {
        throw new Error('boom')
      }, []),
    )

    await waitFor(() => expect(result.current.status).toBe('error'))
    expect(result.current.error).toBeInstanceOf(HttpError)
    expect(result.current.error?.kind).toBe('network')
  })

  it('ignora los errores kind aborted y no cambia el estado', async () => {
    const { result } = renderHook(() =>
      useAsync(async () => {
        throw new HttpError('aborted', 'Consulta cancelada')
      }, []),
    )

    await waitFor(() => expect(result.current.status).toBe('loading'))
    // Deja pasar los microtasks: el estado debe seguir en loading.
    await act(async () => {})
    expect(result.current.status).toBe('loading')
    expect(result.current.error).toBeUndefined()
  })

  it('aborta la petición al desmontar', async () => {
    let capturedSignal: AbortSignal | undefined
    const { unmount } = renderHook(() =>
      useAsync(
        (signal) => {
          capturedSignal = signal
          return new Promise<string>(() => {})
        },
        [],
      ),
    )

    await waitFor(() => expect(capturedSignal).toBeDefined())
    expect(capturedSignal?.aborted).toBe(false)

    unmount()

    expect(capturedSignal?.aborted).toBe(true)
  })

  it('al cambiar deps aborta la ejecución anterior y relanza', async () => {
    const signals: AbortSignal[] = []
    const { result, rerender } = renderHook(
      ({ dep }: { dep: number }) =>
        useAsync(async (signal) => {
          signals.push(signal)
          return `v${dep}`
        }, [dep]),
      { initialProps: { dep: 1 } },
    )

    await waitFor(() => expect(result.current.data).toBe('v1'))

    rerender({ dep: 2 })

    await waitFor(() => expect(result.current.data).toBe('v2'))
    expect(signals[0].aborted).toBe(true)
    expect(signals[1].aborted).toBe(false)
  })
})

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

function wrapperWith(connection: FakeConnection) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <EventsProvider connection={connection}>{children}</EventsProvider>
  }
}

describe('useAsync — refresco por eventos', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('un invalidate de un recurso en refreshOn hace un solo refresco tras 300 ms', async () => {
    const connection = createFakeConnection()
    let calls = 0
    const { result } = renderHook(
      () =>
        useAsync(
          async () => {
            calls += 1
            return `v${calls}`
          },
          [],
          { refreshOn: ['task'] },
        ),
      { wrapper: wrapperWith(connection) },
    )
    await act(async () => {})
    expect(result.current.status).toBe('success')
    expect(calls).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(299)
    })
    expect(calls).toBe(1)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
    })
    expect(calls).toBe(2)
    expect(result.current.data).toBe('v2')
    expect(result.current.refreshing).toBe(false)
  })

  it('un invalidate fuera de refreshOn no refresca', async () => {
    const connection = createFakeConnection()
    let calls = 0
    renderHook(
      () =>
        useAsync(
          async () => {
            calls += 1
            return 'x'
          },
          [],
          { refreshOn: ['task'] },
        ),
      { wrapper: wrapperWith(connection) },
    )
    await act(async () => {})
    expect(calls).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'bug' })
      connection.emit({ type: 'invalidate', resource: 'comment' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    expect(calls).toBe(1)
  })

  it('resync refresca', async () => {
    const connection = createFakeConnection()
    let calls = 0
    renderHook(
      () =>
        useAsync(
          async () => {
            calls += 1
            return 'x'
          },
          [],
          { refreshOn: ['task'] },
        ),
      { wrapper: wrapperWith(connection) },
    )
    await act(async () => {})
    expect(calls).toBe(1)

    act(() => {
      connection.emit({ type: 'resync' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(calls).toBe(2)
  })

  it('3 eventos seguidos dan 1 refresco', async () => {
    const connection = createFakeConnection()
    let calls = 0
    renderHook(
      () =>
        useAsync(
          async () => {
            calls += 1
            return 'x'
          },
          [],
          { refreshOn: ['task'] },
        ),
      { wrapper: wrapperWith(connection) },
    )
    await act(async () => {})
    expect(calls).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
      connection.emit({ type: 'resync' })
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    expect(calls).toBe(2)
  })

  it('un evento con refresco en curso aborta la señal anterior y data sigue visible', async () => {
    const connection = createFakeConnection()
    const pending: ReturnType<typeof deferred<string>>[] = []
    const signals: AbortSignal[] = []
    const { result } = renderHook(
      () =>
        useAsync(
          (signal) => {
            signals.push(signal)
            const next = deferred<string>()
            pending.push(next)
            return next.promise
          },
          [],
          { refreshOn: ['task'] },
        ),
      { wrapper: wrapperWith(connection) },
    )

    await act(async () => {
      pending[0].resolve('v1')
    })
    expect(result.current.data).toBe('v1')

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(signals).toHaveLength(2)
    expect(result.current.refreshing).toBe(true)
    expect(result.current.status).toBe('success')
    expect(result.current.data).toBe('v1')

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(signals).toHaveLength(3)
    expect(signals[1].aborted).toBe(true)

    await act(async () => {
      pending[2].resolve('v3')
    })
    expect(result.current.data).toBe('v3')
    expect(result.current.refreshing).toBe(false)
  })

  it('un error de refresco da status error con data previa y retry la recupera', async () => {
    const connection = createFakeConnection()
    let fail = false
    const { result } = renderHook(
      () =>
        useAsync(
          async () => {
            if (fail) {
              throw new HttpError('http', 'Error del servidor (500)', {
                status: 500,
              })
            }
            return 'dato'
          },
          [],
          { refreshOn: ['task'] },
        ),
      { wrapper: wrapperWith(connection) },
    )
    await act(async () => {})
    expect(result.current.status).toBe('success')
    expect(result.current.data).toBe('dato')

    fail = true
    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(result.current.status).toBe('error')
    expect(result.current.error?.message).toBe('Error del servidor (500)')
    expect(result.current.data).toBe('dato')
    expect(result.current.refreshing).toBe(false)

    fail = false
    await act(async () => {
      result.current.retry()
    })
    expect(result.current.status).toBe('success')
    expect(result.current.data).toBe('dato')
    expect(result.current.refreshing).toBe(false)
  })

  it('al desmontar se limpia el temporizador de refresco', async () => {
    const connection = createFakeConnection()
    const { unmount } = renderHook(
      () => useAsync(async () => 'x', [], { refreshOn: ['task'] }),
      { wrapper: wrapperWith(connection) },
    )
    await act(async () => {})

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    expect(vi.getTimerCount()).toBe(1)

    unmount()
    expect(vi.getTimerCount()).toBe(0)
  })

  it('al cambiar deps se limpia el temporizador pendiente', async () => {
    const connection = createFakeConnection()
    let calls = 0
    const { rerender } = renderHook(
      ({ dep }: { dep: number }) =>
        useAsync(
          async () => {
            calls += 1
            return `v${dep}`
          },
          [dep],
          { refreshOn: ['task'] },
        ),
      {
        wrapper: wrapperWith(connection),
        initialProps: { dep: 1 },
      },
    )
    await act(async () => {})
    expect(calls).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    rerender({ dep: 2 })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    // Solo la carga de los deps nuevos; el refresco pendiente se limpió.
    expect(calls).toBe(2)
  })

  it('sin refreshOn los eventos no refrescan', async () => {
    const connection = createFakeConnection()
    let calls = 0
    renderHook(
      () =>
        useAsync(async () => {
          calls += 1
          return 'x'
        }, []),
      { wrapper: wrapperWith(connection) },
    )
    await act(async () => {})
    expect(calls).toBe(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
      connection.emit({ type: 'resync' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    expect(calls).toBe(1)
  })
})
