import type { ChangeEvent } from '../api/events'
import type { ConnectionStatusProps } from '../components/feedback'
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useSyncExternalStore,
} from 'react'
import type { EventsConnection } from './eventsConnection'

export const EventsContext = createContext<EventsConnection | null>(null)

/** Estado de la conexión de eventos; null si no hay EventsProvider. */
export function useConnectionStatus(): ConnectionStatusProps | null {
  const connection = useContext(EventsContext)
  const subscribe = useCallback(
    (onChange: () => void) =>
      connection === null ? () => {} : connection.subscribeStatus(onChange),
    [connection],
  )
  const getSnapshot = useCallback(
    () => (connection === null ? null : connection.getStatus()),
    [connection],
  )
  return useSyncExternalStore(subscribe, getSnapshot)
}

/** Recibe los eventos de cambio; sin EventsProvider no hace nada. */
export function useChangeEvents(listener: (event: ChangeEvent) => void): void {
  const connection = useContext(EventsContext)
  const listenerRef = useRef(listener)
  useEffect(() => {
    listenerRef.current = listener
  })
  useEffect(() => {
    if (connection === null) return
    return connection.subscribe((event) => {
      listenerRef.current(event)
    })
  }, [connection])
}
