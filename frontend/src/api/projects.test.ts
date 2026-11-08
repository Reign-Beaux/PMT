import { afterEach, describe, expect, it, vi } from 'vitest'
import { getProject, getProjects, getProjectStages } from './projects'

const PROJECT_FIXTURE = {
  id: '1b6f5b3a-9d2e-4c7f-8a1b-2c3d4e5f6a7b',
  name: 'PMT',
  description: 'Gestor de proyectos',
  status: 'working',
  product_version: '1.0.0',
  created_at: '2026-11-01T09:00:00-06:00',
  updated_at: '2026-11-02T10:30:00-06:00',
}

const STAGE_FIXTURE = {
  id: 'f27b13db-6a93-4878-aae6-465827b849f9',
  project_id: PROJECT_FIXTURE.id,
  name: 'Etapa 1',
  description: 'Base del frontend',
  position: 1,
  status: 'in_dev',
  created_at: '2026-11-01T09:05:00-06:00',
  updated_at: '2026-11-03T11:00:00-06:00',
  lead: 'pmt-frontend-tl',
  members: ['pmt-frontend-devfront', 'pmt-frontend-devback'],
  add_reason: 'Refuerzo para la etapa',
  add_requested_by: 'pmt-frontend-tl',
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

describe('rutas y datos', () => {
  it('getProjects pide /api/projects y devuelve la lista', async () => {
    const fetchMock = stubFetch([PROJECT_FIXTURE])

    const projects = await getProjects()

    expect(fetchMock).toHaveBeenCalledWith('/api/projects', expect.anything())
    expect(projects).toEqual([PROJECT_FIXTURE])
  })

  it('getProject pide /api/projects/{id} codificando caracteres especiales', async () => {
    const fetchMock = stubFetch(PROJECT_FIXTURE)

    const project = await getProject('a/b c-ñ')

    expect(fetchMock).toHaveBeenCalledWith(
      `/api/projects/${encodeURIComponent('a/b c-ñ')}`,
      expect.anything(),
    )
    expect(fetchMock.mock.calls[0][0]).toBe('/api/projects/a%2Fb%20c-%C3%B1')
    expect(project).toEqual(PROJECT_FIXTURE)
  })

  it('getProjectStages pide /api/projects/{id}/stages y devuelve las etapas', async () => {
    const fetchMock = stubFetch([STAGE_FIXTURE])

    const stages = await getProjectStages(PROJECT_FIXTURE.id)

    expect(fetchMock).toHaveBeenCalledWith(
      `/api/projects/${PROJECT_FIXTURE.id}/stages`,
      expect.anything(),
    )
    expect(stages).toEqual([STAGE_FIXTURE])
  })

  it('la lista vacía es válida', async () => {
    stubFetch([])

    await expect(getProjects()).resolves.toEqual([])
    await expect(getProjectStages(PROJECT_FIXTURE.id)).resolves.toEqual([])
  })
})

describe('opcionales de Stage', () => {
  const { lead: _lead, members: _members, add_reason: _ar, add_requested_by: _arb, ...sinOpcionales } =
    STAGE_FIXTURE

  it('etapa sin lead, members, add_reason ni add_requested_by es válida', async () => {
    stubFetch([sinOpcionales])

    const stages = await getProjectStages(PROJECT_FIXTURE.id)

    expect(stages).toEqual([sinOpcionales])
    expect(stages[0]).not.toHaveProperty('lead')
    expect(stages[0]).not.toHaveProperty('members')
  })

  it('members null es válido', async () => {
    stubFetch([{ ...sinOpcionales, members: null }])

    const stages = await getProjectStages(PROJECT_FIXTURE.id)

    expect(stages[0].members).toBeNull()
  })
})

describe('respuesta mal formada', () => {
  it.each([
    ['lista que no es array', { hola: 'mundo' }, getProjects],
    [
      'proyecto sin campo name',
      [{ ...PROJECT_FIXTURE, name: undefined }],
      getProjects,
    ],
    [
      'proyecto con estado desconocido',
      [{ ...PROJECT_FIXTURE, status: 'archived' }],
      getProjects,
    ],
    [
      'etapa con position string',
      [{ ...STAGE_FIXTURE, position: '1' }],
      getProjectStages.bind(null, PROJECT_FIXTURE.id) as () => Promise<unknown>,
    ],
    [
      'etapa con members que contiene un número',
      [{ ...STAGE_FIXTURE, members: [1] }],
      getProjectStages.bind(null, PROJECT_FIXTURE.id) as () => Promise<unknown>,
    ],
  ])('%s rechaza con invalid_response', async (_caso, body, fn) => {
    stubFetch(body)

    await expect((fn as () => Promise<unknown>)()).rejects.toMatchObject({
      kind: 'invalid_response',
      message: 'Respuesta no válida del servidor',
    })
  })

  it('getProject con respuesta que no es objeto rechaza con invalid_response', async () => {
    stubFetch(null)

    await expect(getProject(PROJECT_FIXTURE.id)).rejects.toMatchObject({
      kind: 'invalid_response',
    })
  })
})

describe('propagación de errores', () => {
  it('404 se propaga sin cambio', async () => {
    stubFetch({ error: 'Proyecto no encontrado' }, 404)

    await expect(getProject('no-existe')).rejects.toMatchObject({
      kind: 'http',
      status: 404,
      message: 'Proyecto no encontrado',
    })
  })

  it('abort se propaga sin cambio', async () => {
    const fetchMock = stubFetch([PROJECT_FIXTURE])
    const controller = new AbortController()
    controller.abort()

    await expect(getProjects({ signal: controller.signal })).rejects.toMatchObject({
      kind: 'aborted',
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
