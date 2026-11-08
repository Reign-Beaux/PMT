import type { Progress } from '../api/progress'
import { countOf } from '../api/progress'

export interface StageProgressNumbers {
  done: number
  total: number
  cancelled: number
}

/** Une una etapa con su entrada de progress; sin entrada devuelve ceros. */
export function stageProgressFor(
  stageId: string,
  progress: Progress,
): StageProgressNumbers {
  const stage = progress.stages.find((candidate) => candidate.id === stageId)
  if (stage === undefined) return { done: 0, total: 0, cancelled: 0 }
  return {
    done: countOf(stage.counts, 'done'),
    cancelled: countOf(stage.counts, 'cancelled'),
    total: Object.values(stage.counts).reduce((sum, count) => sum + count, 0),
  }
}

/** Copia ordenada por position ascendente, con desempate estable por nombre. */
export function sortStagesByPosition<T extends { position: number; name: string }>(
  stages: readonly T[],
): T[] {
  return [...stages].sort(
    (a, b) => a.position - b.position || a.name.localeCompare(b.name),
  )
}
