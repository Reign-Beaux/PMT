import { afterEach, describe, expect, it, vi } from 'vitest'
import { getComments } from './comments'

const TASK_ID = '70aece42-3e49-4723-87a1-035a17c5af79'
const BUG_ID = '7c9e6679-7425-40de-a82f-38cd8fdc1b08'

// Fixtures con la forma real del backend; QA los usa como referencia (criterio 6).
const COMMENT = {
  id: '2e1e5f6a-9c0b-4d3e-8f1a-2b3c4d5e6f70',
  item_kind: 'task',
  item_id: TASK_ID,
  body: 'Evidencia: pnpm test:run pasó sin errores.',
  actor_name: 'pmt-frontend-devfront',
  actor_profile: 'DevFrontend',
  created_at: '2026-11-07T10:30:00-06:00',
}

function stubFetch(body: unknown, status = 200) {
  const fetchMock = vi.fn((_url: string, _init?: RequestInit) =>
    Promise.resolve({
      ok: status >= 200 && status < 300,
      status,
      text: () =>
        Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
    } as unknown as Response),
  )
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ruta y datos', () => {
  it("getComments('task', id) pide /api/comments?item_kind=task&item_id=… y devuelve la lista", async () => {
    const fetchMock = stubFetch([COMMENT])

    const comments = await getComments('task', TASK_ID)

    expect(fetchMock.mock.calls[0][0]).toBe(
      `/api/comments?item_kind=task&item_id=${TASK_ID}`,
    )
    expect(comments).toEqual([COMMENT])
  })

  it("getComments('bug', id) pide item_kind=bug", async () => {
    const fetchMock = stubFetch([{ ...COMMENT, item_kind: 'bug', item_id: BUG_ID }])

    await getComments('bug', BUG_ID)

    expect(fetchMock.mock.calls[0][0]).toBe(
      `/api/comments?item_kind=bug&item_id=${BUG_ID}`,
    )
  })

  it('codifica el id con & y espacio en la query', async () => {
    const fetchMock = stubFetch([])

    await getComments('task', 'id raro&x')

    expect(fetchMock.mock.calls[0][0]).toBe(
      `/api/comments?${new URLSearchParams({ item_kind: 'task', item_id: 'id raro&x' })}`,
    )
    expect(fetchMock.mock.calls[0][0]).toContain('item_id=id+raro%26x')
  })

  it('la lista vacía es válida', async () => {
    stubFetch([])

    await expect(getComments('task', TASK_ID)).resolves.toEqual([])
  })

  it('el body con HTML se devuelve literal', async () => {
    const html = '<script>alert("x")</script>\n<b>negrita</b>'
    stubFetch([{ ...COMMENT, body: html }])

    const comments = await getComments('task', TASK_ID)

    expect(comments[0].body).toBe(html)
  })
})

describe('datos inválidos', () => {
  it.each([
    ['respuesta que no es array', { comments: [COMMENT] }],
    [
      'item_kind desconocido',
      [{ ...COMMENT, item_kind: 'user' }],
    ],
    ['body que no es string', [{ ...COMMENT, body: 42 }]],
    ['respuesta null', null],
  ])('%s rechaza con invalid_response', async (_caso, body) => {
    stubFetch(body)

    await expect(getComments('task', TASK_ID)).rejects.toMatchObject({
      kind: 'invalid_response',
      message: 'Respuesta no válida del servidor',
    })
  })
})

describe('propagación de errores', () => {
  it('400 se propaga sin cambio', async () => {
    stubFetch({ error: 'item_id debe ser un UUID' }, 400)

    await expect(getComments('task', 'no-uuid')).rejects.toMatchObject({
      kind: 'http',
      status: 400,
      message: 'item_id debe ser un UUID',
    })
  })

  it('abort se propaga sin cambio', async () => {
    const fetchMock = stubFetch([COMMENT])
    const controller = new AbortController()
    controller.abort()

    await expect(
      getComments('task', TASK_ID, { signal: controller.signal }),
    ).rejects.toMatchObject({ kind: 'aborted' })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
