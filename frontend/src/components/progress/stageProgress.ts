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
