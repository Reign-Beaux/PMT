import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EventsProvider } from '../../realtime/EventsProvider'
import { createFakeConnection } from '../../test/fakeConnection'
import type { FakeConnection } from '../../test/fakeConnection'
import { StatusHistory } from './StatusHistory'
import { formatDate } from '../../domain/formatDate'

function makeTaskChange(overrides: Record<string, unknown> = {}) {
  return {
    id: 'h1',
    task_id: 't1',
    from_status: 'ready',
    to_status: 'in_dev',
    actor_name: 'Dana',
    actor_profile: 'pmt-frontend-tl',
    reason: 'Empieza el desarrollo',
    changed_at: '2026-11-01T09:00:00-06:00',
    ...overrides,
  }
}

function makeBugChange(overrides: Record<string, unknown> = {}) {
  return {
    id: 'h1',
    bug_id: 'b1',
    from_status: 'confirmed',
    to_status: 'rejected',
    actor_name: 'Eva',
    actor_profile: 'pmt-frontend-qa',
    reason: 'No es un fallo',
    changed_at: '2026-11-01T09:00:00-06:00',
    ...overrides,
  }
}

function jsonResponse(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () =>
      Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
  } as unknown as Response
}

interface StubOptions {
  taskChanges?: unknown[]
  bugChanges?: unknown[]
  failOnce?: boolean
  hang?: boolean
}

function stubFetch(options: StubOptions = {}) {
  const { taskChanges = [], bugChanges = [], failOnce = false, hang = false } =
    options
  let calls = 0
  const fetchMock = vi.fn((url: string) => {
    if (url === '/api/tasks/t1/status-changes') {
      calls += 1
      if (hang) return new Promise<Response>(() => {})
      const fail = failOnce ? calls === 1 : false
      return Promise.resolve(
        fail
          ? jsonResponse({ error: 'No se pudo cargar' }, 500)
          : jsonResponse(taskChanges),
      )
    }
    if (url === '/api/bugs/b1/status-changes') {
      calls += 1
      if (hang) return new Promise<Response>(() => {})
      const fail = failOnce ? calls === 1 : false
      return Promise.resolve(
        fail
          ? jsonResponse({ error: 'No se pudo cargar' }, 500)
          : jsonResponse(bugChanges),
      )
    }
    return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('StatusHistory (tarea 06)', () => {
  it('variante tarea pide /tasks/{id}/status-changes y muestra etiquetas de tarea', async () => {
    const fetchMock = stubFetch({
      taskChanges: [
        makeTaskChange(),
        makeTaskChange({
          id: 'h2',
          from_status: 'in_dev',
          to_status: 'cancelled',
          reason: 'Fuera de alcance',
          changed_at: '2026-11-02T09:00:00-06:00',
        }),
      ],
    })

    render(<StatusHistory itemKind="task" itemId="t1" />)

    const section = await screen.findByRole('region', {
      name: 'Historial de estados',
    })
    const items = await within(section).findAllByRole('listitem')
    expect(items).toHaveLength(2)
    expect(items[1].textContent).toContain('En desarrollo → Cancelada')
    expect(items[1].textContent).toContain('Dana (pmt-frontend-tl)')
    expect(items[1].textContent).toContain(
      formatDate('2026-11-02T09:00:00-06:00'),
    )
    expect(items[1].textContent).toContain('Fuera de alcance')
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tasks/t1/status-changes',
      expect.anything(),
    )
  })

  it('variante bug pide /bugs/{id}/status-changes y muestra etiquetas de bug', async () => {
    const fetchMock = stubFetch({ bugChanges: [makeBugChange()] })

    render(<StatusHistory itemKind="bug" itemId="b1" />)

    const section = await screen.findByRole('region', {
      name: 'Historial de estados',
    })
    const items = await within(section).findAllByRole('listitem')
    expect(items).toHaveLength(1)
    expect(items[0].textContent).toContain('Confirmado → Rechazado')
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/bugs/b1/status-changes',
      expect.anything(),
    )
  })

  it('ordena del más antiguo al más reciente aunque la API responda desordenado', async () => {
    stubFetch({
      taskChanges: [
        makeTaskChange({
          id: 'h2',
          from_status: 'in_dev',
          to_status: 'in_review',
          changed_at: '2026-11-05T09:00:00-06:00',
        }),
        makeTaskChange(),
      ],
    })

    render(<StatusHistory itemKind="task" itemId="t1" />)

    const section = await screen.findByRole('region', {
      name: 'Historial de estados',
    })
    const items = await within(section).findAllByRole('listitem')
    expect(items[0].textContent).toContain('Lista → En desarrollo')
    expect(items[1].textContent).toContain('En desarrollo → En revisión')
  })

  it('motivo vacío o solo espacios no se muestra', async () => {
    stubFetch({
      taskChanges: [
        makeTaskChange({ reason: '' }),
        makeTaskChange({
          id: 'h2',
          from_status: 'in_dev',
          to_status: 'done',
          reason: '   ',
          changed_at: '2026-11-02T09:00:00-06:00',
        }),
      ],
    })

    render(<StatusHistory itemKind="task" itemId="t1" />)

    const section = await screen.findByRole('region', {
      name: 'Historial de estados',
    })
    const items = await within(section).findAllByRole('listitem')
    expect(items).toHaveLength(2)
    for (const item of items) {
      expect(item.querySelectorAll('p')).toHaveLength(1)
    }
  })

  it('el motivo con HTML y saltos de línea se muestra literal', async () => {
    stubFetch({
      taskChanges: [makeTaskChange({ reason: '<i>x</i>\nlínea 2' })],
    })

    render(<StatusHistory itemKind="task" itemId="t1" />)

    const section = await screen.findByRole('region', {
      name: 'Historial de estados',
    })
    const item = await within(section).findByRole('listitem')
    const paragraphs = item.querySelectorAll('p')
    expect(paragraphs[paragraphs.length - 1].textContent).toBe(
      '<i>x</i>\nlínea 2',
    )
    expect(document.querySelector('i')).toBeNull()
  })

  it('lista vacía muestra "Sin cambios de estado"', async () => {
    stubFetch({ taskChanges: [] })

    render(<StatusHistory itemKind="task" itemId="t1" />)

    expect(await screen.findByText('Sin cambios de estado')).toBeDefined()
  })

  it('un error muestra ErrorState y el reintento vuelve a pedir', async () => {
    const fetchMock = stubFetch({
      taskChanges: [makeTaskChange()],
      failOnce: true,
    })
    const user = userEvent.setup()

    render(<StatusHistory itemKind="task" itemId="t1" />)

    expect(await screen.findByText('No se pudo cargar')).toBeDefined()
    expect(screen.queryByText('Sin cambios de estado')).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(await screen.findByText('Empieza el desarrollo')).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('muestra "Cargando historial…" mientras carga', async () => {
    stubFetch({ hang: true })

    render(<StatusHistory itemKind="task" itemId="t1" />)

    expect(await screen.findByText('Cargando historial…')).toBeDefined()
  })
})

describe('StatusHistory — refresco por eventos (tarea 03)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function renderWithEvents(connection: FakeConnection) {
    return render(<StatusHistory itemKind="task" itemId="t1" />, {
      wrapper: ({ children }) => (
        <EventsProvider connection={connection}>{children}</EventsProvider>
      ),
    })
  }

  it('un invalidate de status_change vuelve a pedir el historial; otro recurso no', async () => {
    const fetchMock = stubFetch({ taskChanges: [makeTaskChange()] })
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByText('Empieza el desarrollo')).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'comment' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    expect(fetchMock).toHaveBeenCalledTimes(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'status_change' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('tras el refresco muestra los datos actualizados', async () => {
    let calls = 0
    const fetchMock = vi.fn((url: string) => {
      if (url === '/api/tasks/t1/status-changes') {
        calls += 1
        return Promise.resolve(
          jsonResponse(
            calls === 1
              ? [makeTaskChange()]
              : [
                  makeTaskChange(),
                  makeTaskChange({
                    id: 'h2',
                    from_status: 'in_dev',
                    to_status: 'done',
                    reason: 'Terminada',
                    changed_at: '2026-11-02T09:00:00-06:00',
                  }),
                ],
          ),
        )
      }
      return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
    })
    vi.stubGlobal('fetch', fetchMock)
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.queryByText('Terminada')).toBeNull()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'status_change' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(screen.getByText('Terminada')).toBeDefined()
    expect(screen.getByText('Empieza el desarrollo')).toBeDefined()
  })
})
