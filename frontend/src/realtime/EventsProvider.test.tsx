import { describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { StrictMode } from 'react'
import type { ChangeEvent } from '../api/events'
import { EventsProvider } from './EventsProvider'
import { useChangeEvents, useConnectionStatus } from './eventsContext'
import { createFakeConnection } from '../test/fakeConnection'

function StatusProbe() {
  const status = useConnectionStatus()
  if (status === null) return <p>sin proveedor</p>
  return <p>{status.status}</p>
}

function EventProbe({ onEvent }: { onEvent: (event: ChangeEvent) => void }) {
  useChangeEvents(onEvent)
  return null
}

describe('EventsProvider', () => {
  it('start al montar y close al desmontar', () => {
    const connection = createFakeConnection()
    const { unmount } = render(
      <EventsProvider connection={connection}>
        <StatusProbe />
      </EventsProvider>,
    )
    expect(connection.start).toHaveBeenCalledTimes(1)
    expect(connection.close).not.toHaveBeenCalled()
    unmount()
    expect(connection.close).toHaveBeenCalledTimes(1)
  })

  it('start/close/start con StrictMode no rompe', () => {
    const connection = createFakeConnection()
    render(
      <StrictMode>
        <EventsProvider connection={connection}>
          <StatusProbe />
        </EventsProvider>
      </StrictMode>,
    )
    expect(connection.start).toHaveBeenCalled()
  })

  it('useConnectionStatus refleja los cambios de estado', () => {
    const connection = createFakeConnection()
    render(
      <EventsProvider connection={connection}>
        <StatusProbe />
      </EventsProvider>,
    )
    expect(screen.getByText('reconnecting')).toBeDefined()
    act(() => {
      connection.setStatus({
        status: 'connected',
        lastSyncedAt: '2026-11-07T12:00:00.000Z',
      })
    })
    expect(screen.getByText('connected')).toBeDefined()
  })

  it('useChangeEvents recibe eventos y deja de recibirlos al desmontarse', () => {
    const connection = createFakeConnection()
    const onEvent = vi.fn()
    const { unmount } = render(
      <EventsProvider connection={connection}>
        <EventProbe onEvent={onEvent} />
      </EventsProvider>,
    )
    const event: ChangeEvent = { type: 'invalidate', resource: 'task' }
    act(() => {
      connection.emit(event)
    })
    expect(onEvent).toHaveBeenCalledTimes(1)
    expect(onEvent).toHaveBeenCalledWith(event)
    unmount()
    act(() => {
      connection.emit(event)
    })
    expect(onEvent).toHaveBeenCalledTimes(1)
  })

  it('useChangeEvents siempre llama al listener más reciente', () => {
    const connection = createFakeConnection()
    const first = vi.fn()
    const second = vi.fn()
    const { rerender } = render(
      <EventsProvider connection={connection}>
        <EventProbe onEvent={first} />
      </EventsProvider>,
    )
    rerender(
      <EventsProvider connection={connection}>
        <EventProbe onEvent={second} />
      </EventsProvider>,
    )
    act(() => {
      connection.emit({ type: 'resync' })
    })
    expect(first).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledTimes(1)
  })

  it('sin proveedor: useConnectionStatus da null y useChangeEvents no hace nada', () => {
    const onEvent = vi.fn()
    render(
      <div>
        <StatusProbe />
        <EventProbe onEvent={onEvent} />
      </div>,
    )
    expect(screen.getByText('sin proveedor')).toBeDefined()
    expect(onEvent).not.toHaveBeenCalled()
  })
})
