import { vi } from 'vitest'
import type { ChangeEvent } from '../api/events'
import type { ConnectionStatusProps } from '../components/feedback'
import type { EventsConnection } from '../realtime/eventsConnection'

export type FakeConnection = ReturnType<typeof createFakeConnection>

/** EventsConnection falso: start/close son spies; el estado y los eventos se mueven a mano. */
export function createFakeConnection() {
  let status: ConnectionStatusProps = { status: 'reconnecting' }
  const statusListeners = new Set<() => void>()
  const listeners = new Set<(event: ChangeEvent) => void>()
  return {
    start: vi.fn(),
    close: vi.fn(),
    getStatus: (): ConnectionStatusProps => status,
    subscribeStatus(listener: () => void) {
      statusListeners.add(listener)
      return () => {
        statusListeners.delete(listener)
      }
    },
    subscribe(listener: (event: ChangeEvent) => void) {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
    setStatus(next: ConnectionStatusProps): void {
      status = next
      for (const listener of [...statusListeners]) listener()
    },
    emit(event: ChangeEvent): void {
      for (const listener of [...listeners]) listener(event)
    },
  } satisfies EventsConnection & {
    setStatus(next: ConnectionStatusProps): void
    emit(event: ChangeEvent): void
  }
}
