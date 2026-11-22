import type { BugStatus } from '../domain/status'
import { BUG_STATUS_LABELS } from '../domain/status'
import type { GetOptions } from './http'
import { get, post } from './http'
import type { Hold } from './hold'
import { readHold } from './hold'
import {
  optionalString,
  requireArray,
  requireRecord,
  requireStatus,
  requireString,
} from './validate'

export interface Bug extends Hold {
  id: string
  project_id: string
  title: string
  description: string
  steps_to_reproduce: string
  product_version: string
  status: BugStatus
  reported_by?: string
  created_at: string
  updated_at: string
}

function validateBug(value: unknown): Bug {
  const record = requireRecord(value)
  const bug: Bug = {
    id: requireString(record, 'id'),
    project_id: requireString(record, 'project_id'),
    title: requireString(record, 'title'),
    description: requireString(record, 'description'),
    steps_to_reproduce: requireString(record, 'steps_to_reproduce'),
    product_version: requireString(record, 'product_version'),
    status: requireStatus(record, 'status', BUG_STATUS_LABELS),
    created_at: requireString(record, 'created_at'),
    updated_at: requireString(record, 'updated_at'),
  }
  const reportedBy = optionalString(record, 'reported_by')
  if (reportedBy !== undefined) bug.reported_by = reportedBy
  return { ...bug, ...readHold(record) }
}

export async function getProjectBugs(
  projectId: string,
  options?: GetOptions,
): Promise<Bug[]> {
  const data = await get<unknown>(
    `/projects/${encodeURIComponent(projectId)}/bugs`,
    options,
  )
  return requireArray(data).map(validateBug)
}

/** Libera un bug tomado. Solo el owner del proyecto; el motivo es obligatorio. */
export async function releaseBug(id: string, reason: string): Promise<Bug> {
  const data = await post<unknown>(`/bugs/${encodeURIComponent(id)}/release`, { reason })
  return validateBug(data)
}
