import { describe, expect, it } from 'vitest'
import type { Progress, ProgressStage } from '../api/progress'
import { stageProgressPercent } from '../components/progress/stageProgress'
import { sortStagesByPosition, stageProgressFor } from './stageListProgress'

function makeStage(id: string, counts: Record<string, number>): ProgressStage {
  return {
    id,
    name: `Etapa ${id}`,
    status: 'in_dev',
    members: [],
    counts,
    tasks: [],
  }
}

function makeProgress(stages: ProgressStage[]): Progress {
  return {
    project: {
      id: 'p1',
      name: 'Proyecto Uno',
      status: 'working',
      last_change_at: '2026-11-01T09:00:00-06:00',
    },
    stages,
    open_bugs: { counts: {}, items: [] },
    signals: [],
  }
}

describe('stageProgressFor', () => {
  it('etapa sin entrada en progress → ceros y 0 %', () => {
    const progress = makeProgress([])

    const numbers = stageProgressFor('s1', progress)

    expect(numbers).toEqual({ done: 0, total: 0, cancelled: 0 })
    expect(stageProgressPercent(numbers)).toBe(0)
  })

  it('counts vacío (0 tareas) → 0 %', () => {
    const progress = makeProgress([makeStage('s1', {})])

    const numbers = stageProgressFor('s1', progress)

    expect(numbers).toEqual({ done: 0, total: 0, cancelled: 0 })
    expect(stageProgressPercent(numbers)).toBe(0)
  })

  it('todas canceladas → 0 %', () => {
    const progress = makeProgress([makeStage('s1', { cancelled: 3 })])

    const numbers = stageProgressFor('s1', progress)

    expect(numbers).toEqual({ done: 0, total: 3, cancelled: 3 })
    expect(stageProgressPercent(numbers)).toBe(0)
  })

  it('done 2 de 3 → 67 %', () => {
    const progress = makeProgress([makeStage('s1', { done: 2, in_dev: 1 })])

    const numbers = stageProgressFor('s1', progress)

    expect(numbers).toEqual({ done: 2, total: 3, cancelled: 0 })
    expect(stageProgressPercent(numbers)).toBe(67)
  })

  it('counts sin done ni cancelled los suma al total como 0', () => {
    const progress = makeProgress([makeStage('s1', { ready: 4, backlog: 1 })])

    const numbers = stageProgressFor('s1', progress)

    expect(numbers).toEqual({ done: 0, total: 5, cancelled: 0 })
  })

  it('las canceladas no cuentan para el porcentaje: 2 done, 1 cancelada → 100 %', () => {
    const progress = makeProgress([
      makeStage('s1', { done: 2, cancelled: 1 }),
    ])

    const numbers = stageProgressFor('s1', progress)

    expect(numbers).toEqual({ done: 2, total: 3, cancelled: 1 })
    expect(stageProgressPercent(numbers)).toBe(100)
  })
})

describe('sortStagesByPosition', () => {
  it('ordena por position ascendente sin mutar la entrada', () => {
    const stages = [
      { id: 'c', name: 'C', position: 3 },
      { id: 'a', name: 'A', position: 1 },
      { id: 'b', name: 'B', position: 2 },
    ]

    const sorted = sortStagesByPosition(stages)

    expect(sorted.map((stage) => stage.id)).toEqual(['a', 'b', 'c'])
    expect(stages.map((stage) => stage.id)).toEqual(['c', 'a', 'b'])
  })

  it('positions repetidas → desempate estable por nombre', () => {
    const stages = [
      { id: 'x', name: 'Beta', position: 1 },
      { id: 'y', name: 'Alfa', position: 1 },
    ]

    const sorted = sortStagesByPosition(stages)

    expect(sorted.map((stage) => stage.id)).toEqual(['y', 'x'])
  })
})
