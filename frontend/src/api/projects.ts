import type { ProjectStatus, StageStatus } from '../domain/status'
import { PROJECT_STATUS_LABELS, STAGE_STATUS_LABELS } from '../domain/status'
import type { GetOptions } from './http'
import { get } from './http'
import {
  optionalString,
  optionalStringArray,
  requireArray,
  requireInt,
  requireRecord,
  requireStatus,
  requireString,
} from './validate'

export interface Project {
  id: string
  name: string
  description: string
  status: ProjectStatus
  product_version: string
  created_at: string
  updated_at: string
}

export interface Stage {
  id: string
  project_id: string
  name: string
  description: string
  position: number
  status: StageStatus
  created_at: string
  updated_at: string
  lead?: string
  members?: string[] | null
  add_reason?: string
  add_requested_by?: string
}

function validateProject(value: unknown): Project {
  const record = requireRecord(value)
  return {
    id: requireString(record, 'id'),
    name: requireString(record, 'name'),
    description: requireString(record, 'description'),
    status: requireStatus(record, 'status', PROJECT_STATUS_LABELS),
    product_version: requireString(record, 'product_version'),
    created_at: requireString(record, 'created_at'),
    updated_at: requireString(record, 'updated_at'),
  }
}

function validateStage(value: unknown): Stage {
  const record = requireRecord(value)
  const stage: Stage = {
    id: requireString(record, 'id'),
    project_id: requireString(record, 'project_id'),
    name: requireString(record, 'name'),
    description: requireString(record, 'description'),
    position: requireInt(record, 'position'),
    status: requireStatus(record, 'status', STAGE_STATUS_LABELS),
    created_at: requireString(record, 'created_at'),
    updated_at: requireString(record, 'updated_at'),
  }
  const lead = optionalString(record, 'lead')
  if (lead !== undefined) stage.lead = lead
  const members = optionalStringArray(record, 'members')
  if (members !== undefined) stage.members = members
  const addReason = optionalString(record, 'add_reason')
  if (addReason !== undefined) stage.add_reason = addReason
  const addRequestedBy = optionalString(record, 'add_requested_by')
  if (addRequestedBy !== undefined) stage.add_requested_by = addRequestedBy
  return stage
}

export async function getProjects(options?: GetOptions): Promise<Project[]> {
  const data = await get<unknown>('/projects', options)
  return requireArray(data).map(validateProject)
}

export async function getProject(id: string, options?: GetOptions): Promise<Project> {
  const data = await get<unknown>(`/projects/${encodeURIComponent(id)}`, options)
  return validateProject(data)
}

export async function getProjectStages(
  projectId: string,
  options?: GetOptions,
): Promise<Stage[]> {
  const data = await get<unknown>(
    `/projects/${encodeURIComponent(projectId)}/stages`,
    options,
  )
  return requireArray(data).map(validateStage)
}
