export type ProjectStatus =
  | 'planning'
  | 'stand_by'
  | 'in_dev'
  | 'working'
  | 'stopped'
  | 'cancelled'

export type StageStatus =
  | 'planning'
  | 'stand_by'
  | 'in_dev'
  | 'done'
  | 'stopped'
  | 'cancelled'

export type TaskStatus =
  | 'backlog'
  | 'ready'
  | 'in_dev'
  | 'in_review'
  | 'changes_requested'
  | 'in_qa'
  | 'pending_close'
  | 'done'
  | 'cancelled'

export type BugStatus =
  | 'reported'
  | 'confirmed'
  | 'in_dev'
  | 'in_review'
  | 'changes_requested'
  | 'in_qa'
  | 'pending_close'
  | 'done'
  | 'rejected'

export const PROJECT_STATUS_LABELS: Record<ProjectStatus, string> = {
  planning: 'Planeación',
  stand_by: 'En espera',
  in_dev: 'En desarrollo',
  working: 'En operación',
  stopped: 'Detenido',
  cancelled: 'Cancelado',
}

export const STAGE_STATUS_LABELS: Record<StageStatus, string> = {
  planning: 'Planeación',
  stand_by: 'En espera',
  in_dev: 'En desarrollo',
  done: 'Terminada',
  stopped: 'Detenida',
  cancelled: 'Cancelada',
}

export const TASK_STATUS_LABELS: Record<TaskStatus, string> = {
  backlog: 'Backlog',
  ready: 'Lista',
  in_dev: 'En desarrollo',
  in_review: 'En revisión',
  changes_requested: 'Cambios solicitados',
  in_qa: 'En QA',
  pending_close: 'Pendiente de cierre',
  done: 'Terminada',
  cancelled: 'Cancelada',
}

export const BUG_STATUS_LABELS: Record<BugStatus, string> = {
  reported: 'Reportado',
  confirmed: 'Confirmado',
  in_dev: 'En desarrollo',
  in_review: 'En revisión',
  changes_requested: 'Cambios solicitados',
  in_qa: 'En QA',
  pending_close: 'Pendiente de cierre',
  done: 'Corregido',
  rejected: 'Rechazado',
}

export const TASK_COLUMN_ORDER = [
  'backlog',
  'ready',
  'in_dev',
  'in_review',
  'changes_requested',
  'in_qa',
  'pending_close',
  'done',
  'cancelled',
] as const satisfies readonly TaskStatus[]

export const BUG_COLUMN_ORDER = [
  'reported',
  'confirmed',
  'in_dev',
  'in_review',
  'changes_requested',
  'in_qa',
  'pending_close',
  'done',
  'rejected',
] as const satisfies readonly BugStatus[]

// Si falta un estado en el orden, Exclude deja de ser never y `true` ya no asigna: falla tsc.
type TaskOrderComplete = Exclude<
  TaskStatus,
  (typeof TASK_COLUMN_ORDER)[number]
> extends never
  ? true
  : never
type BugOrderComplete = Exclude<
  BugStatus,
  (typeof BUG_COLUMN_ORDER)[number]
> extends never
  ? true
  : never

const taskOrderComplete: TaskOrderComplete = true
const bugOrderComplete: BugOrderComplete = true
void taskOrderComplete
void bugOrderComplete
