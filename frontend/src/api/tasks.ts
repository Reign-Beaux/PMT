import type { TaskStatus } from '../domain/status'
import { TASK_STATUS_LABELS } from '../domain/status'
import type { GetOptions } from './http'
import { get } from './http'
import {
  optionalString,
  requireArray,
  requireRecord,
  requireStatus,
  requireString,
} from './validate'

export interface Task {
  id: string
  stage_id: string
  title: string
  description: string
  status: TaskStatus
  created_at: string
  updated_at: string
  assignee?: string
}

function validateTask(value: unknown): Task {
  const record = requireRecord(value)
  const task: Task = {
    id: requireString(record, 'id'),
    stage_id: requireString(record, 'stage_id'),
    title: requireString(record, 'title'),
    description: requireString(record, 'description'),
    status: requireStatus(record, 'status', TASK_STATUS_LABELS),
    created_at: requireString(record, 'created_at'),
    updated_at: requireString(record, 'updated_at'),
  }
  const assignee = optionalString(record, 'assignee')
  if (assignee !== undefined) task.assignee = assignee
  return task
}

export async function getStageTasks(
  stageId: string,
  options?: GetOptions,
): Promise<Task[]> {
  const data = await get<unknown>(
    `/stages/${encodeURIComponent(stageId)}/tasks`,
    options,
  )
  return requireArray(data).map(validateTask)
}

export async function getTask(id: string, options?: GetOptions): Promise<Task> {
  const data = await get<unknown>(`/tasks/${encodeURIComponent(id)}`, options)
  return validateTask(data)
}
