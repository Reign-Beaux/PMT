import type { GetOptions } from './http'
import { del, get, post, put } from './http'
import { PROJECT_ROLES, type ProjectRole } from './projects'
import { requireArray, requireOneOf, requireRecord, requireString } from './validate'

export interface Member {
  username: string
  name: string
  role: ProjectRole
}

/** Rol que el owner puede dar: el owner es uno solo y no cambia. */
export type SharedRole = Exclude<ProjectRole, 'owner'>

function validateMember(value: unknown): Member {
  const record = requireRecord(value)
  return {
    username: requireString(record, 'username'),
    name: requireString(record, 'name'),
    role: requireOneOf(record, 'role', PROJECT_ROLES),
  }
}

const membersPath = (projectId: string) => `/projects/${encodeURIComponent(projectId)}/members`
const memberPath = (projectId: string, username: string) =>
  `${membersPath(projectId)}/${encodeURIComponent(username)}`

export async function getProjectMembers(projectId: string, options?: GetOptions): Promise<Member[]> {
  const data = await get<unknown>(membersPath(projectId), options)
  return requireArray(data).map(validateMember)
}

export async function addProjectMember(projectId: string, username: string, role: SharedRole): Promise<Member> {
  return validateMember(await post<unknown>(membersPath(projectId), { username, role }))
}

export async function updateProjectMember(projectId: string, username: string, role: SharedRole): Promise<Member> {
  return validateMember(await put<unknown>(memberPath(projectId, username), { role }))
}

export function removeProjectMember(projectId: string, username: string): Promise<void> {
  return del<void>(memberPath(projectId, username))
}
