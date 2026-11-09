import type { Progress } from '../api/progress'
import { stageProgressDenominator, stageProgressPercent } from '../components/progress/stageProgress'

/** Todos los indicadores se derivan de los datos actuales; no hay campos nuevos. */
export function summarize(progress: Progress) {
  const counts: Record<string, number> = {}
  for (const stage of progress.stages) {
    for (const [status, count] of Object.entries(stage.counts)) counts[status] = (counts[status] ?? 0) + count
  }
  const total = Object.values(counts).reduce((sum, value) => sum + value, 0)
  const done = counts.done ?? 0
  const cancelled = counts.cancelled ?? 0
  return {
    counts, done, total, cancelled,
    denominator: stageProgressDenominator({ total, done, cancelled }),
    percent: stageProgressPercent({ total, done, cancelled }),
    completedStages: progress.stages.filter(stage => stage.status === 'done').length,
    activeStages: progress.stages.filter(stage => stage.status === 'in_dev').length,
    openBugs: progress.open_bugs.items.length,
  }
}
