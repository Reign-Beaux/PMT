import type { TaskStatus } from '../../domain/status'
import { TASK_STATUS_LABELS } from '../../domain/status'
import { TASK_STATUS_TONES } from '../../domain/statusTone'
import type { BadgeTone } from '../card/Badge'

export interface StageProgressInput {
  done: number
  total: number
  cancelled: number
}

export function stageProgressDenominator({
  total,
  cancelled,
}: StageProgressInput): number {
  return total - cancelled
}

export function stageProgressPercent({
  done,
  total,
  cancelled,
}: StageProgressInput): number {
  const denom = total - cancelled
  if (denom <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((100 * done) / denom)))
}

export interface ProgressSegment {
  status: TaskStatus
  count: number
  percent: number
  tone: BadgeTone
}

/**
 * Segmentos por status de tarea, en el orden de TASK_STATUS_LABELS; descarta
 * claves desconocidas y conteos ≤ 0 o no finitos. Suma 0 → sin segmentos.
 */
export function progressSegments(
  counts: Record<string, number>,
): ProgressSegment[] {
  const statuses = (Object.keys(TASK_STATUS_LABELS) as TaskStatus[]).filter(
    (status) => {
      const count = counts[status]
      return typeof count === 'number' && Number.isFinite(count) && count > 0
    },
  )
  const sum = statuses.reduce((total, status) => total + counts[status], 0)
  if (sum <= 0) return []
  return statuses.map((status) => ({
    status,
    count: counts[status],
    percent: (counts[status] / sum) * 100,
    tone: TASK_STATUS_TONES[status],
  }))
}
