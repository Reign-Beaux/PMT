import type { ChangeEvent } from '../api/events'
import { parseChangeEvent } from '../api/events'
import type { ConnectionStatusProps } from '../components/feedback'

/** WebSocket mínimo que necesita la conexión; el real y el falso de pruebas lo cumplen. */
export interface SocketLike {
  onopen: ((ev: Event) => void) | null
  onmessage: ((ev: MessageEvent) => void) | null
  onerror: ((ev: Event) => void) | null
  onclose: ((ev: CloseEvent) => void) | null
  close(): void
}

export type ChangeEventListener = (event: ChangeEvent) => void

export interface EventsConnection {
  start(): void
  close(): void
  getStatus(): ConnectionStatusProps
  subscribeStatus(listener: () => void): () => void
  subscribe(listener: ChangeEventListener): () => void
}

export interface EventsConnectionOptions {
  url?: string
  createSocket?: (url: string) => SocketLike
  now?: () => Date
}

const RETRY_DELAYS_MS = [1_000, 2_000, 4_000, 8_000, 16_000, 30_000]
const STALE_AFTER_MS = 30_000

export function eventsUrl(loc: Pick<Location, 'protocol' | 'host'>): string {
  const scheme = loc.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${scheme}//${loc.host}/api/events`
}

export function createEventsConnection(
  options: EventsConnectionOptions = {},
): EventsConnection {
  const url = options.url ?? eventsUrl(window.location)
  const createSocket = options.createSocket ?? ((u: string) => new WebSocket(u))
  const now = options.now ?? (() => new Date())

  let status: ConnectionStatusProps = { status: 'reconnecting' }
  let socket: SocketLike | null = null
  let started = false
  let attempt = 0
  let retryTimer: ReturnType<typeof setTimeout> | undefined
  let staleTimer: ReturnType<typeof setTimeout> | undefined
  const statusListeners = new Set<() => void>()
  const eventListeners = new Set<ChangeEventListener>()

  function setStatus(next: ConnectionStatusProps): void {
    status = next
    for (const listener of [...statusListeners]) {
      listener()
    }
  }

  function clearRetryTimer(): void {
    if (retryTimer !== undefined) {
      clearTimeout(retryTimer)
      retryTimer = undefined
    }
  }

  function clearStaleTimer(): void {
    if (staleTimer !== undefined) {
      clearTimeout(staleTimer)
      staleTimer = undefined
    }
  }

  // El plazo de 30 s corre desde la primera pérdida (o desde start()); los
  // reintentos fallidos no lo reinician. Solo resync y close() lo limpian.
  function armStaleTimer(): void {
    if (staleTimer !== undefined || status.status === 'stale') return
    staleTimer = setTimeout(() => {
      staleTimer = undefined
      if (status.status === 'connected') return
      setStatus({ status: 'stale', lastSyncedAt: status.lastSyncedAt })
    }, STALE_AFTER_MS)
  }

  function scheduleRetry(): void {
    clearRetryTimer()
    const delay = RETRY_DELAYS_MS[Math.min(attempt, RETRY_DELAYS_MS.length - 1)]
    attempt += 1
    retryTimer = setTimeout(() => {
      retryTimer = undefined
      openSocket()
    }, delay)
  }

  function dispatch(event: ChangeEvent): void {
    for (const listener of [...eventListeners]) {
      try {
        listener(event)
      } catch (error) {
        console.error('[eventos] listener falló:', error)
      }
    }
  }

  function handleMessage(data: unknown): void {
    const result = parseChangeEvent(data)
    if (!result.ok) {
      console.warn('[eventos] frame inválido:', result.error)
      return
    }
    if (result.event.type === 'resync') {
      attempt = 0
      clearStaleTimer()
      setStatus({ status: 'connected', lastSyncedAt: now().toISOString() })
    }
    dispatch(result.event)
  }

  function openSocket(): void {
    if (!started) return
    let next: SocketLike
    try {
      next = createSocket(url)
    } catch (error) {
      console.error('[eventos] no se pudo crear el socket:', error)
      armStaleTimer()
      scheduleRetry()
      return
    }
    socket = next
    next.onmessage = (ev) => {
      if (socket === next) handleMessage(ev.data)
    }
    next.onclose = () => {
      if (!started || socket !== next) return
      socket = null
      if (status.status === 'connected') {
        setStatus({ status: 'reconnecting', lastSyncedAt: status.lastSyncedAt })
      }
      armStaleTimer()
      scheduleRetry()
    }
  }

  return {
    start() {
      if (started) return
      started = true
      attempt = 0
      if (status.status !== 'reconnecting') {
        setStatus({ status: 'reconnecting', lastSyncedAt: status.lastSyncedAt })
      }
      armStaleTimer()
      openSocket()
    },
    close() {
      started = false
      clearRetryTimer()
      clearStaleTimer()
      const current = socket
      socket = null
      if (current !== null) {
        current.onopen = null
        current.onmessage = null
        current.onerror = null
        current.onclose = null
        current.close()
      }
    },
    getStatus() {
      return status
    },
    subscribeStatus(listener) {
      statusListeners.add(listener)
      return () => {
        statusListeners.delete(listener)
      }
    },
    subscribe(listener) {
      eventListeners.add(listener)
      return () => {
        eventListeners.delete(listener)
      }
    },
  }
}
