import type {
  BugStatus,
  ProjectStatus,
  StageStatus,
  TaskStatus,
} from '../domain/status'
import {
  BUG_STATUS_LABELS,
  PROJECT_STATUS_LABELS,
  STAGE_STATUS_LABELS,
  TASK_STATUS_LABELS,
} from '../domain/status'
import type { GetOptions } from './http'
import { get } from './http'
import {
  optionalString,
  requireArrayField,
  requireCounts,
  requireRecord,
  requireStatus,
  requireString,
  requireStringArray,
} from './validate'

export interface ProgressTask {
  id: string
  title: string
  status: TaskStatus
  assignee?: string
  updated_at: string
}

export interface ProgressStage {
  id: string
  name: string
  status: StageStatus
  lead?: string
  members: string[]
  counts: Record<string, number>
  tasks: ProgressTask[]
}

/** open_bugs solo trae los bugs abiertos; el listado completo es getProjectBugs. */
export interface Progress {
  project: {
    id: string
    name: string
    status: ProjectStatus
    last_change_at: string
  }
  stages: ProgressStage[]
  open_bugs: {
    counts: Record<string, number>
    items: {
      id: string
      title: string
      status: BugStatus
      updated_at: string
    }[]
  }
  signals: {
    code: string
    stage_id: string
    stage_name: string
  }[]
}

function validateProgressTask(value: unknown): ProgressTask {
  const record = requireRecord(value)
  const task: ProgressTask = {
    id: requireString(record, 'id'),
    title: requireString(record, 'title'),
    status: requireStatus(record, 'status', TASK_STATUS_LABELS),
    updated_at: requireString(record, 'updated_at'),
  }
  const assignee = optionalString(record, 'assignee')
  if (assignee !== undefined) task.assignee = assignee
  return task
}

function validateProgressStage(value: unknown): ProgressStage {
  const record = requireRecord(value)
  const stage: ProgressStage = {
    id: requireString(record, 'id'),
    name: requireString(record, 'name'),
    status: requireStatus(record, 'status', STAGE_STATUS_LABELS),
    members: requireStringArray(record, 'members'),
    counts: requireCounts(record, 'counts'),
    tasks: requireArrayField(record, 'tasks').map(validateProgressTask),
  }
  const lead = optionalString(record, 'lead')
  if (lead !== undefined) stage.lead = lead
  return stage
}

function validateOpenBug(value: unknown): Progress['open_bugs']['items'][number] {
  const record = requireRecord(value)
  return {
    id: requireString(record, 'id'),
    title: requireString(record, 'title'),
    status: requireStatus(record, 'status', BUG_STATUS_LABELS),
    updated_at: requireString(record, 'updated_at'),
  }
}

function validateSignal(value: unknown): Progress['signals'][number] {
  const record = requireRecord(value)
  return {
    code: requireString(record, 'code'),
    stage_id: requireString(record, 'stage_id'),
    stage_name: requireString(record, 'stage_name'),
  }
}

function validateProgress(value: unknown): Progress {
  const record = requireRecord(value)
  const projectRecord = requireRecord(record['project'])
  const openBugsRecord = requireRecord(record['open_bugs'])
  return {
    project: {
      id: requireString(projectRecord, 'id'),
      name: requireString(projectRecord, 'name'),
      status: requireStatus(projectRecord, 'status', PROJECT_STATUS_LABELS),
      last_change_at: requireString(projectRecord, 'last_change_at'),
    },
    stages: requireArrayField(record, 'stages').map(validateProgressStage),
    open_bugs: {
      counts: requireCounts(openBugsRecord, 'counts'),
      items: requireArrayField(openBugsRecord, 'items').map(validateOpenBug),
    },
    signals: requireArrayField(record, 'signals').map(validateSignal),
  }
}

export async function getProjectProgress(
  projectId: string,
  options?: GetOptions,
): Promise<Progress> {
  const data = await get<unknown>(
    `/projects/${encodeURIComponent(projectId)}/progress`,
    options,
  )
  return validateProgress(data)
}

/** Los counts ausentes por estado equivalen a 0. */
export function countOf(counts: Record<string, number>, key: string): number {
  return counts[key] ?? 0
}
