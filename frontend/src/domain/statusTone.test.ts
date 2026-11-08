import { describe, expect, it } from 'vitest'
import {
  BUG_STATUS_LABELS,
  PROJECT_STATUS_LABELS,
  STAGE_STATUS_LABELS,
  TASK_STATUS_LABELS,
} from './status'
import {
  BUG_STATUS_TONES,
  PROJECT_STATUS_TONES,
  STAGE_STATUS_TONES,
  TASK_STATUS_TONES,
} from './statusTone'

const VALID_TONES = ['neutral', 'info', 'success', 'warning', 'danger']

describe('statusTone', () => {
  it('todo estado de proyecto tiene un tone válido', () => {
    expect(Object.keys(PROJECT_STATUS_TONES).sort()).toEqual(
      Object.keys(PROJECT_STATUS_LABELS).sort(),
    )
    for (const tone of Object.values(PROJECT_STATUS_TONES)) {
      expect(VALID_TONES).toContain(tone)
    }
  })

  it('todo estado de etapa tiene un tone válido', () => {
    expect(Object.keys(STAGE_STATUS_TONES).sort()).toEqual(
      Object.keys(STAGE_STATUS_LABELS).sort(),
    )
    for (const tone of Object.values(STAGE_STATUS_TONES)) {
      expect(VALID_TONES).toContain(tone)
    }
  })

  it('mapeos esperados de proyecto', () => {
    expect(PROJECT_STATUS_TONES).toEqual({
      planning: 'neutral',
      stand_by: 'neutral',
      in_dev: 'info',
      working: 'success',
      stopped: 'warning',
      cancelled: 'danger',
    })
  })

  it('mapeos esperados de etapa', () => {
    expect(STAGE_STATUS_TONES).toEqual({
      planning: 'neutral',
      stand_by: 'neutral',
      in_dev: 'info',
      done: 'success',
      stopped: 'warning',
      cancelled: 'danger',
    })
  })

  it('todo estado de tarea tiene un tone válido', () => {
    expect(Object.keys(TASK_STATUS_TONES).sort()).toEqual(
      Object.keys(TASK_STATUS_LABELS).sort(),
    )
    expect(Object.keys(TASK_STATUS_TONES)).toHaveLength(9)
    for (const tone of Object.values(TASK_STATUS_TONES)) {
      expect(VALID_TONES).toContain(tone)
    }
  })

  it('mapeos esperados de tarea', () => {
    expect(TASK_STATUS_TONES).toEqual({
      backlog: 'neutral',
      ready: 'neutral',
      in_dev: 'info',
      in_review: 'info',
      changes_requested: 'warning',
      in_qa: 'info',
      pending_close: 'success',
      done: 'success',
      cancelled: 'danger',
    })
  })

  it('todo estado de bug tiene un tone válido', () => {
    expect(Object.keys(BUG_STATUS_TONES).sort()).toEqual(
      Object.keys(BUG_STATUS_LABELS).sort(),
    )
    for (const tone of Object.values(BUG_STATUS_TONES)) {
      expect(VALID_TONES).toContain(tone)
    }
  })

  it('mapeos esperados de bug', () => {
    expect(BUG_STATUS_TONES).toEqual({
      reported: 'neutral',
      confirmed: 'neutral',
      in_dev: 'info',
      in_review: 'info',
      changes_requested: 'warning',
      in_qa: 'info',
      pending_close: 'success',
      done: 'success',
      rejected: 'danger',
    })
  })
})
