import { afterEach, describe, expect, it, vi } from 'vitest'
import { releaseBug } from './bugs'
import { addProjectMember, getProjectMembers, removeProjectMember, updateProjectMember } from './members'
import { getProject, getProjectStages, releaseStage } from './projects'

const PID = '1b6f5b3a-9d2e-4c7f-8a1b-2c3d4e5f6a7b'
const PROJECT = { id: PID, name: 'PMT', description: '', status: 'working', product_version: '1.0.0', created_at: '2026-11-01T09:00:00Z', updated_at: '2026-11-02T10:00:00Z' }
const STAGE = { id: 's1', project_id: PID, name: 'E1', description: '', position: 1, status: 'in_dev', created_at: '2026-11-01T09:00:00Z', updated_at: '2026-11-01T09:00:00Z' }
const BUG = { id: 'b1', project_id: PID, title: 'Falla', description: '', steps_to_reproduce: '1', product_version: '1.0.0', status: 'in_dev', created_at: '2026-11-01T09:00:00Z', updated_at: '2026-11-01T09:00:00Z' }

function stubFetch(body: unknown, status = 200) {
  const fetchMock = vi.fn((_url: string, _init?: RequestInit) => Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(body === undefined ? '' : JSON.stringify(body)),
  } as unknown as Response))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const sent = (mock: ReturnType<typeof stubFetch>) => {
  const [url, init] = mock.mock.calls[0]
  return { url, method: init?.method ?? 'GET', body: init?.body ? JSON.parse(String(init.body)) : undefined }
}

afterEach(() => vi.unstubAllGlobals())

describe('rol del proyecto', () => {
  it('lee role cuando viene y lo omite cuando no', async () => {
    stubFetch({ ...PROJECT, role: 'viewer' })
    expect((await getProject(PID)).role).toBe('viewer')
    stubFetch(PROJECT)
    expect('role' in (await getProject(PID))).toBe(false)
  })

  it('rechaza un rol desconocido', async () => {
    stubFetch({ ...PROJECT, role: 'admin' })
    await expect(getProject(PID)).rejects.toMatchObject({ kind: 'invalid_response' })
  })
})

describe('tomado por', () => {
  it('las etapas traen taken_by y taken_at si alguien las tiene', async () => {
    stubFetch([{ ...STAGE, taken_by: 'ana', taken_at: '2026-11-22T10:00:00Z' }, STAGE])
    const [taken, free] = await getProjectStages(PID)
    expect(taken).toMatchObject({ taken_by: 'ana', taken_at: '2026-11-22T10:00:00Z' })
    expect(free.taken_by).toBeUndefined()
  })
})

describe('miembros', () => {
  it('lista, agrega, cambia el rol y quita', async () => {
    let mock = stubFetch([{ username: 'saul', name: 'Saúl', role: 'owner' }])
    expect(await getProjectMembers(PID)).toEqual([{ username: 'saul', name: 'Saúl', role: 'owner' }])
    expect(sent(mock).url).toBe(`/api/projects/${PID}/members`)

    mock = stubFetch({ username: 'ana', name: 'Ana', role: 'viewer' }, 201)
    await addProjectMember(PID, 'ana', 'viewer')
    expect(sent(mock)).toEqual({ url: `/api/projects/${PID}/members`, method: 'POST', body: { username: 'ana', role: 'viewer' } })

    mock = stubFetch({ username: 'ana', name: 'Ana', role: 'editor' })
    await updateProjectMember(PID, 'ana', 'editor')
    expect(sent(mock)).toEqual({ url: `/api/projects/${PID}/members/ana`, method: 'PUT', body: { role: 'editor' } })

    mock = stubFetch(undefined, 204)
    await removeProjectMember(PID, 'ana')
    expect(sent(mock)).toMatchObject({ url: `/api/projects/${PID}/members/ana`, method: 'DELETE' })
  })

  it('propaga el mensaje del backend', async () => {
    stubFetch({ error: 'el usuario ya es miembro del proyecto' }, 409)
    await expect(addProjectMember(PID, 'ana', 'editor')).rejects.toMatchObject({ status: 409, message: 'el usuario ya es miembro del proyecto' })
  })
})

describe('liberar', () => {
  it('etapa y bug con su motivo', async () => {
    let mock = stubFetch(STAGE)
    await releaseStage('s1', 'vacaciones')
    expect(sent(mock)).toEqual({ url: '/api/stages/s1/release', method: 'POST', body: { reason: 'vacaciones' } })
    mock = stubFetch(BUG)
    await releaseBug('b1', 'vacaciones')
    expect(sent(mock)).toEqual({ url: '/api/bugs/b1/release', method: 'POST', body: { reason: 'vacaciones' } })
  })
})
