import { describe, expect, it } from 'vitest'
import {
  BUG_COLUMN_ORDER,
  BUG_STATUS_LABELS,
  PROJECT_STATUS_LABELS,
  STAGE_STATUS_LABELS,
  TASK_COLUMN_ORDER,
  TASK_STATUS_LABELS,
} from './status'

const PROJECT_STATUSES = [
  'planning',
  'stand_by',
  'in_dev',
  'working',
  'stopped',
  'cancelled',
]
const STAGE_STATUSES = [
  'planning',
  'stand_by',
  'in_dev',
  'done',
  'stopped',
  'cancelled',
]
const TASK_STATUSES = [
  'backlog',
  'ready',
  'in_dev',
  'in_review',
  'changes_requested',
  'in_qa',
  'pending_close',
  'done',
  'cancelled',
]
const BUG_STATUSES = [
  'reported',
  'confirmed',
  'in_dev',
  'in_review',
  'changes_requested',
  'in_qa',
  'pending_close',
  'done',
  'rejected',
]

describe('mapas de etiquetas', () => {
  it('PROJECT_STATUS_LABELS tiene exactamente los estados de proyecto', () => {
    expect(Object.keys(PROJECT_STATUS_LABELS).sort()).toEqual(
      [...PROJECT_STATUSES].sort(),
    )
    expect(Object.values(PROJECT_STATUS_LABELS).every((v) => v.length > 0)).toBe(
      true,
    )
  })

  it('STAGE_STATUS_LABELS tiene exactamente los estados de etapa', () => {
    expect(Object.keys(STAGE_STATUS_LABELS).sort()).toEqual(
      [...STAGE_STATUSES].sort(),
    )
    expect(Object.values(STAGE_STATUS_LABELS).every((v) => v.length > 0)).toBe(
      true,
    )
  })

  it('TASK_STATUS_LABELS tiene exactamente los estados de tarea', () => {
    expect(Object.keys(TASK_STATUS_LABELS).sort()).toEqual(
      [...TASK_STATUSES].sort(),
    )
    expect(Object.values(TASK_STATUS_LABELS).every((v) => v.length > 0)).toBe(
      true,
    )
  })

  it('BUG_STATUS_LABELS tiene exactamente los estados de bug', () => {
    expect(Object.keys(BUG_STATUS_LABELS).sort()).toEqual(
      [...BUG_STATUSES].sort(),
    )
    expect(Object.values(BUG_STATUS_LABELS).every((v) => v.length > 0)).toBe(
      true,
    )
  })
})

describe('orden de columnas', () => {
  it('TASK_COLUMN_ORDER contiene todos los estados de tarea, sin repetir y en el orden del plan', () => {
    expect(TASK_COLUMN_ORDER).toEqual(TASK_STATUSES)
    expect(new Set(TASK_COLUMN_ORDER).size).toBe(TASK_STATUSES.length)
  })

  it('BUG_COLUMN_ORDER contiene todos los estados de bug, sin repetir y en el orden del plan', () => {
    expect(BUG_COLUMN_ORDER).toEqual(BUG_STATUSES)
    expect(new Set(BUG_COLUMN_ORDER).size).toBe(BUG_STATUSES.length)
  })
})
