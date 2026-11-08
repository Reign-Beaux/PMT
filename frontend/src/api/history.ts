import type { BugStatus, TaskStatus } from '../domain/status'
import { BUG_STATUS_LABELS, TASK_STATUS_LABELS } from '../domain/status'
import type { GetOptions } from './http'
import { get } from './http'
import { requireArray, requireRecord, requireStatus, requireString } from './validate'

export interface HistoryBase {
  id: string
  actor_name: string
  actor_profile: string
  reason: string
  changed_at: string
}

export type TaskHistory = HistoryBase & {
  task_id: string
  from_status: TaskStatus
  to_status: TaskStatus
}

export type BugHistory = HistoryBase & {
  bug_id: string
  from_status: BugStatus
  to_status: BugStatus
}

function validateHistoryBase(record: Record<string, unknown>): HistoryBase {
  return {
    id: requireString(record, 'id'),
    actor_name: requireString(record, 'actor_name'),
    actor_profile: requireString(record, 'actor_profile'),
    reason: requireString(record, 'reason'),
    changed_at: requireString(record, 'changed_at'),
  }
}

function validateTaskHistory(value: unknown): TaskHistory {
  const record = requireRecord(value)
  return {
    ...validateHistoryBase(record),
    task_id: requireString(record, 'task_id'),
    from_status: requireStatus(record, 'from_status', TASK_STATUS_LABELS),
    to_status: requireStatus(record, 'to_status', TASK_STATUS_LABELS),
  }
}

function validateBugHistory(value: unknown): BugHistory {
  const record = requireRecord(value)
  return {
    ...validateHistoryBase(record),
    bug_id: requireString(record, 'bug_id'),
    from_status: requireStatus(record, 'from_status', BUG_STATUS_LABELS),
    to_status: requireStatus(record, 'to_status', BUG_STATUS_LABELS),
  }
}

export async function getTaskHistory(
  taskId: string,
  options?: GetOptions,
): Promise<TaskHistory[]> {
  const data = await get<unknown>(
    `/tasks/${encodeURIComponent(taskId)}/status-changes`,
    options,
  )
  return requireArray(data).map(validateTaskHistory)
}

export async function getBugHistory(
  bugId: string,
  options?: GetOptions,
): Promise<BugHistory[]> {
  const data = await get<unknown>(
    `/bugs/${encodeURIComponent(bugId)}/status-changes`,
    options,
  )
  return requireArray(data).map(validateBugHistory)
}
