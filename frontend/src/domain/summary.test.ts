import { describe, expect, it } from 'vitest'
import type { Progress } from '../api/progress'
import { summarize } from './summary'

function progress(counts: Record<string, number>): Progress {
  return { project: { id: 'p', name: 'PMT', status: 'working', last_change_at: '' }, stages: [{ id: 's', name: 'Etapa', status: 'in_dev', counts, tasks: [], members: [] }], open_bugs: { counts: {}, items: [] }, signals: [] }
}

describe('indicadores derivados del avance', () => {
  it('excluye las tareas canceladas del total usado para el porcentaje', () => {
    expect(summarize(progress({ done: 3, in_review: 1, cancelled: 6 }))).toMatchObject({ total: 10, done: 3, denominator: 4, percent: 75, activeStages: 1 })
  })
  it('un conjunto completamente cancelado no produce NaN ni 100%', () => {
    expect(summarize(progress({ cancelled: 4 }))).toMatchObject({ denominator: 0, percent: 0 })
  })
  it('suma conteos de todas las etapas sin promediar sus porcentajes', () => {
    const input = progress({ done: 1 })
    input.stages.push({ id: 's2', name: 'Etapa 2', status: 'done', counts: { done: 8, in_dev: 1 }, tasks: [], members: [] })
    expect(summarize(input)).toMatchObject({ done: 9, denominator: 10, percent: 90, completedStages: 1 })
  })
})
