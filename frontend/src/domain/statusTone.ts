import type { BadgeTone } from '../components/card/Badge'
import type { BugStatus, ProjectStatus, StageStatus, TaskStatus } from './status'

export const PROJECT_STATUS_TONES: Record<ProjectStatus, BadgeTone> = {
  planning: 'neutral',
  stand_by: 'neutral',
  in_dev: 'info',
  working: 'success',
  stopped: 'warning',
  cancelled: 'danger',
}

export const STAGE_STATUS_TONES: Record<StageStatus, BadgeTone> = {
  planning: 'neutral',
  stand_by: 'neutral',
  in_dev: 'info',
  done: 'success',
  stopped: 'warning',
  cancelled: 'danger',
}

export const TASK_STATUS_TONES: Record<TaskStatus, BadgeTone> = {
  backlog: 'neutral',
  ready: 'neutral',
  in_dev: 'info',
  in_review: 'info',
  changes_requested: 'warning',
  in_qa: 'info',
  pending_close: 'success',
  done: 'success',
  cancelled: 'danger',
}

export const BUG_STATUS_TONES: Record<BugStatus, BadgeTone> = {
  reported: 'neutral',
  confirmed: 'neutral',
  in_dev: 'info',
  in_review: 'info',
  changes_requested: 'warning',
  in_qa: 'info',
  pending_close: 'success',
  done: 'success',
  rejected: 'danger',
}
