import type { ChangeEventResource } from '../api/events'

export type Resource = ChangeEventResource

/** Recursos que refrescan cada vista al recibir un invalidate (resync refresca todas). */
export const PROJECT_RESOURCES: readonly Resource[] = ['project']
export const STAGE_RESOURCES: readonly Resource[] = ['stage', 'task']
export const TASK_RESOURCES: readonly Resource[] = ['task']
export const BUG_RESOURCES: readonly Resource[] = ['bug']
export const COMMENT_RESOURCES: readonly Resource[] = ['comment']
export const STATUS_CHANGE_RESOURCES: readonly Resource[] = ['status_change']
