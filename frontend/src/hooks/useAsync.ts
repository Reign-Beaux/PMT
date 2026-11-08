import { useCallback, useEffect, useRef, useState } from 'react'
import { HttpError } from '../api/http'
import { useChangeEvents } from '../realtime/eventsContext'
import type { Resource } from '../realtime/refreshResources'

export type AsyncStatus = 'loading' | 'success' | 'error'

export interface AsyncResult<T> {
  status: AsyncStatus
  data?: T
  error?: HttpError
  retry: () => void
  /** true mientras corre un refresco con datos previos visibles. */
  refreshing: boolean
}

export interface UseAsyncOptions {
  /** Recursos cuyo invalidate refresca esta consulta; resync la refresca siempre. */
  refreshOn?: readonly Resource[]
}

interface Completed<T> {
  runKey: string
  depsVersion: number
  data?: T
  error?: HttpError
}

const REFRESH_DEBOUNCE_MS = 300

function sameDeps(a: readonly unknown[], b: readonly unknown[]): boolean {
  return a.length === b.length && a.every((value, index) => Object.is(value, b[index]))
}

/**
 * Carga datos con un loader que recibe AbortSignal. Aborta al desmontar o al
 * cambiar deps; retry vuelve a 'loading' y relanza. Ignora los errores de
 * cancelación (kind 'aborted') y los resultados de ejecuciones ya abortadas.
 *
 * Con refreshOn y un EventsProvider activo, un invalidate de esos recursos o
 * un resync agrupan (300 ms, flanco final) un refresco que conserva los datos
 * previos visibles (refreshing=true) y no borra data si falla.
 */
export function useAsync<T>(
  loader: (signal: AbortSignal) => Promise<T>,
  deps: readonly unknown[],
  options?: UseAsyncOptions,
): AsyncResult<T> {
  const [attempt, setAttempt] = useState(0)
  const [prevDeps, setPrevDeps] = useState(deps)
  const [depsVersion, setDepsVersion] = useState(0)
  const [completed, setCompleted] = useState<Completed<T> | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const loaderRef = useRef(loader)
  const refreshOnRef = useRef(options?.refreshOn)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const completedRef = useRef(completed)
  const depsVersionRef = useRef(depsVersion)

  // Patrón de estado derivado durante el render: si cambian deps, la ejecución
  // anterior deja de ser válida y la vista vuelve a 'loading'.
  if (!sameDeps(prevDeps, deps)) {
    setPrevDeps(deps)
    setDepsVersion(depsVersion + 1)
    setRefreshing(false)
  }

  useEffect(() => {
    loaderRef.current = loader
  }, [loader])

  useEffect(() => {
    refreshOnRef.current = options?.refreshOn
  })

  useEffect(() => {
    completedRef.current = completed
    depsVersionRef.current = depsVersion
  }, [completed, depsVersion])

  const runKey = `${depsVersion}:${attempt}`

  useEffect(() => {
    const controller = new AbortController()
    let active = true
    void (async () => {
      try {
        const data = await loaderRef.current(controller.signal)
        if (active) {
          setCompleted({ runKey, depsVersion, data })
          setRefreshing(false)
        }
      } catch (error) {
        if (!active) return
        if (error instanceof HttpError && error.kind === 'aborted') return
        const httpError =
          error instanceof HttpError
            ? error
            : new HttpError('network', 'No se pudo conectar con el servidor', {
                cause: error,
              })
        // Un refresco que falla conserva los datos previos de estos deps.
        setCompleted((previous) => ({
          runKey,
          depsVersion,
          error: httpError,
          data:
            previous !== null && previous.depsVersion === depsVersion
              ? previous.data
              : undefined,
        }))
        setRefreshing(false)
      }
    })()
    return () => {
      active = false
      controller.abort()
    }
  }, [runKey, depsVersion])

  // Al desmontar o cambiar deps se limpia el refresco pendiente.
  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current)
        timerRef.current = null
      }
    }
  }, [depsVersion])

  const launchRefresh = useCallback(() => {
    setRefreshing(true)
    setAttempt((previous) => previous + 1)
  }, [])

  useChangeEvents((event) => {
    const refreshOn = refreshOnRef.current
    if (refreshOn === undefined) return
    if (event.type !== 'resync' && !refreshOn.includes(event.resource)) return
    if (timerRef.current !== null) return
    timerRef.current = setTimeout(() => {
      timerRef.current = null
      launchRefresh()
    }, REFRESH_DEBOUNCE_MS)
  })

  const retry = useCallback(() => {
    const current = completedRef.current
    if (
      current !== null &&
      current.depsVersion === depsVersionRef.current &&
      current.data !== undefined
    ) {
      // Con datos previos, el reintento es un refresco: no los borra.
      setRefreshing(true)
    }
    setAttempt((previous) => previous + 1)
  }, [])

  const current =
    completed !== null && completed.depsVersion === depsVersion
      ? completed
      : null

  if (refreshing && current !== null) {
    if (current.error !== undefined) {
      return {
        status: 'error',
        error: current.error,
        data: current.data,
        retry,
        refreshing,
      }
    }
    return { status: 'success', data: current.data, retry, refreshing }
  }
  if (current === null || current.runKey !== runKey) {
    return { status: 'loading', retry, refreshing }
  }
  if (current.error !== undefined) {
    return {
      status: 'error',
      error: current.error,
      data: current.data,
      retry,
      refreshing,
    }
  }
  return { status: 'success', data: current.data, retry, refreshing }
}
