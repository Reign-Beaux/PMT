import { afterEach, describe, expect, it, vi } from 'vitest'
import { getProjectBugs } from './bugs'

const PROJECT_ID = '1b6f5b3a-9d2e-4c7f-8a1b-2c3d4e5f6a7b'

// Fixtures con la forma real del backend; QA los usa como referencia (criterio 12).
const BUG_DONE = {
  id: 'aaaaaaaa-1111-4222-8333-bbbbbbbbbbbb',
  project_id: PROJECT_ID,
  title: 'El kanban no se refresca',
  description: 'Al mover una tarjeta, la columna no se actualiza.',
  steps_to_reproduce: '1. Abrir etapa. 2. Mover tarjeta. 3. Ver columna.',
  product_version: '1.0.0',
  status: 'done',
  reported_by: 'pmt-frontend-qa',
  created_at: '2026-11-04T08:00:00-06:00',
  updated_at: '2026-11-05T09:30:00-06:00',
}

const BUG_REJECTED = {
  id: 'cccccccc-1111-4222-8333-dddddddddddd',
  project_id: PROJECT_ID,
  title: 'Tema claro ilegible',
  description: 'Se pidió solo tema oscuro.',
  steps_to_reproduce: '',
  product_version: '',
  status: 'rejected',
  reported_by: 'pmt-frontend-pp',
  created_at: '2026-11-04T10:00:00-06:00',
  updated_at: '2026-11-04T12:00:00-06:00',
}

const BUG_MINIMAL = {
  id: 'eeeeeeee-1111-4222-8333-ffffffffffff',
  project_id: PROJECT_ID,
  title: 'Sin autor registrado',
  description: 'Bug sin reported_by.',
  steps_to_reproduce: 'No aplica.',
  product_version: '1.0.0',
  status: 'reported',
  created_at: '2026-11-06T08:00:00-06:00',
  updated_at: '2026-11-06T08:00:00-06:00',
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
  it('getProjectBugs pide /api/projects/{id}/bugs sin query', async () => {
    const fetchMock = stubFetch([BUG_DONE, BUG_REJECTED, BUG_MINIMAL])

    const bugs = await getProjectBugs(PROJECT_ID)

    expect(fetchMock.mock.calls[0][0]).toBe(`/api/projects/${PROJECT_ID}/bugs`)
    expect(fetchMock.mock.calls[0][0]).not.toContain('?')
    expect(bugs).toHaveLength(3)
    expect(bugs.map((b) => b.status)).toEqual(['done', 'rejected', 'reported'])
  })

  it('la lista con bugs done y rejected se devuelve completa', async () => {
    stubFetch([BUG_DONE, BUG_REJECTED])

    const bugs = await getProjectBugs(PROJECT_ID)

    expect(bugs).toEqual([BUG_DONE, BUG_REJECTED])
  })

  it('la lista vacía es válida', async () => {
    stubFetch([])

    await expect(getProjectBugs(PROJECT_ID)).resolves.toEqual([])
  })

  it('reported_by ausente es válido', async () => {
    stubFetch([BUG_MINIMAL])

    const bugs = await getProjectBugs(PROJECT_ID)

    expect(bugs[0]).not.toHaveProperty('reported_by')
  })

  it('description, steps_to_reproduce y product_version vacíos son válidos', async () => {
    stubFetch([{ ...BUG_REJECTED, description: '', steps_to_reproduce: '', product_version: '' }])

    const bugs = await getProjectBugs(PROJECT_ID)

    expect(bugs[0].description).toBe('')
  })
})

describe('respuesta inválida', () => {
  it.each([
    ['objeto en lugar de array', { bugs: [BUG_DONE] }],
    ['bug con estado desconocido', [{ ...BUG_DONE, status: 'closed' }]],
    ['bug sin title', [{ ...BUG_DONE, title: undefined }]],
    ['reported_by que no es string', [{ ...BUG_DONE, reported_by: 42 }]],
  ])('%s rechaza con invalid_response', async (_caso, body) => {
    stubFetch(body)

    await expect(getProjectBugs(PROJECT_ID)).rejects.toMatchObject({
      kind: 'invalid_response',
      message: 'Respuesta no válida del servidor',
    })
  })
})

describe('propagación de errores', () => {
  it('error HTTP se propaga sin cambio', async () => {
    stubFetch({ error: 'Proyecto no encontrado' }, 404)

    await expect(getProjectBugs('no-existe')).rejects.toMatchObject({
      kind: 'http',
      status: 404,
      message: 'Proyecto no encontrado',
    })
  })

  it('abort se propaga sin cambio', async () => {
    const fetchMock = stubFetch([BUG_DONE])
    const controller = new AbortController()
    controller.abort()

    await expect(getProjectBugs(PROJECT_ID, { signal: controller.signal })).rejects.toMatchObject({
      kind: 'aborted',
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
