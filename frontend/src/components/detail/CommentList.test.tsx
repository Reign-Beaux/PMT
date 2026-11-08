import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EventsProvider } from '../../realtime/EventsProvider'
import { createFakeConnection } from '../../test/fakeConnection'
import type { FakeConnection } from '../../test/fakeConnection'
import { CommentList } from './CommentList'
import { formatDate } from '../../domain/formatDate'

function makeComment(overrides: Record<string, unknown> = {}) {
  return {
    id: 'c1',
    item_kind: 'task',
    item_id: 't1',
    body: 'Primer comentario',
    actor_name: 'Ana',
    actor_profile: 'pmt-frontend-pp',
    created_at: '2026-11-01T09:00:00-06:00',
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
  comments?: unknown[]
  failOnce?: boolean
  hang?: boolean
}

function stubFetch(options: StubOptions = {}) {
  const { comments = [], failOnce = false, hang = false } = options
  let calls = 0
  const fetchMock = vi.fn((url: string) => {
    if (url.startsWith('/api/comments?')) {
      calls += 1
      if (hang) return new Promise<Response>(() => {})
      const fail = failOnce ? calls === 1 : false
      return Promise.resolve(
        fail
          ? jsonResponse({ error: 'No se pudo cargar' }, 500)
          : jsonResponse(comments),
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

describe('CommentList (tarea 05)', () => {
  it('comentarios desordenados se muestran del más antiguo al más reciente con autor y fecha', async () => {
    stubFetch({
      comments: [
        makeComment({
          id: 'c2',
          body: 'Segundo',
          actor_name: 'Beto',
          actor_profile: 'pmt-frontend-tl',
          created_at: '2026-11-03T09:00:00-06:00',
        }),
        makeComment({ id: 'c1', body: 'Primero' }),
        makeComment({
          id: 'c3',
          body: 'Tercero',
          created_at: '2026-11-05T09:00:00-06:00',
        }),
      ],
    })

    render(<CommentList itemKind="task" itemId="t1" />)

    const section = await screen.findByRole('region', { name: 'Comentarios' })
    const items = await within(section).findAllByRole('listitem')
    expect(items).toHaveLength(3)
    expect(items[0].textContent).toContain('Primero')
    expect(items[0].textContent).toContain('Ana (pmt-frontend-pp)')
    expect(items[0].textContent).toContain(
      formatDate('2026-11-01T09:00:00-06:00'),
    )
    expect(items[1].textContent).toContain('Segundo')
    expect(items[1].textContent).toContain('Beto (pmt-frontend-tl)')
    expect(items[2].textContent).toContain('Tercero')
  })

  it('un body con HTML y saltos de línea se muestra como texto literal', async () => {
    stubFetch({
      comments: [makeComment({ body: '<b>x</b>\nsegunda línea' })],
    })

    render(<CommentList itemKind="task" itemId="t1" />)

    const section = await screen.findByRole('region', { name: 'Comentarios' })
    const item = await within(section).findByRole('listitem')
    const paragraphs = item.querySelectorAll('p')
    expect(paragraphs[paragraphs.length - 1].textContent).toBe(
      '<b>x</b>\nsegunda línea',
    )
    expect(document.querySelector('b')).toBeNull()
  })

  it('actor_profile vacío muestra solo el nombre sin paréntesis', async () => {
    stubFetch({
      comments: [makeComment({ actor_name: 'Sola', actor_profile: '' })],
    })

    render(<CommentList itemKind="task" itemId="t1" />)

    const section = await screen.findByRole('region', { name: 'Comentarios' })
    const item = await within(section).findByRole('listitem')
    expect(item.textContent).toContain('Sola')
    expect(item.textContent).not.toContain('(')
  })

  it('lista vacía muestra "Sin comentarios"', async () => {
    stubFetch({ comments: [] })

    render(<CommentList itemKind="task" itemId="t1" />)

    expect(await screen.findByText('Sin comentarios')).toBeDefined()
  })

  it('un error muestra ErrorState y el reintento vuelve a pedir', async () => {
    const fetchMock = stubFetch({
      comments: [makeComment()],
      failOnce: true,
    })
    const user = userEvent.setup()

    render(<CommentList itemKind="task" itemId="t1" />)

    expect(await screen.findByText('No se pudo cargar')).toBeDefined()
    expect(screen.queryByText('Sin comentarios')).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    expect(await screen.findByText('Primer comentario')).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('muestra "Cargando comentarios…" mientras carga', async () => {
    stubFetch({ hang: true })

    render(<CommentList itemKind="task" itemId="t1" />)

    expect(await screen.findByText('Cargando comentarios…')).toBeDefined()
  })

  it('con itemKind task la URL lleva item_kind=task e item_id', async () => {
    const fetchMock = stubFetch()

    render(<CommentList itemKind="task" itemId="t1" />)
    await screen.findByText('Sin comentarios')

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/comments?item_kind=task&item_id=t1',
      expect.anything(),
    )
  })

  it('con itemKind bug la URL lleva item_kind=bug e item_id', async () => {
    const fetchMock = stubFetch()

    render(<CommentList itemKind="bug" itemId="b7" />)
    await screen.findByText('Sin comentarios')

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/comments?item_kind=bug&item_id=b7',
      expect.anything(),
    )
  })
})

describe('CommentList — refresco por eventos (tarea 03)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function renderWithEvents(connection: FakeConnection) {
    return render(<CommentList itemKind="task" itemId="t1" />, {
      wrapper: ({ children }) => (
        <EventsProvider connection={connection}>{children}</EventsProvider>
      ),
    })
  }

  it('un invalidate de comment vuelve a pedir los comentarios; otro recurso no', async () => {
    const fetchMock = stubFetch({ comments: [makeComment()] })
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByText('Primer comentario')).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'task' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    expect(fetchMock).toHaveBeenCalledTimes(1)

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'comment' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(screen.getByText('Primer comentario')).toBeDefined()
  })

  it('un error de refresco muestra ErrorState sin borrar los comentarios previos', async () => {
    let calls = 0
    const fetchMock = vi.fn((url: string) => {
      if (url.startsWith('/api/comments?')) {
        calls += 1
        return Promise.resolve(
          calls === 1
            ? jsonResponse([makeComment()])
            : jsonResponse({ error: 'No se pudo cargar' }, 500),
        )
      }
      return Promise.resolve(jsonResponse({ error: 'URL no esperada' }, 404))
    })
    vi.stubGlobal('fetch', fetchMock)
    const connection = createFakeConnection()
    renderWithEvents(connection)

    await act(async () => {})
    expect(screen.getByText('Primer comentario')).toBeDefined()

    act(() => {
      connection.emit({ type: 'invalidate', resource: 'comment' })
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(screen.getByText('No se pudo cargar')).toBeDefined()
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeDefined()
    expect(screen.getByText('Primer comentario')).toBeDefined()
  })
})
