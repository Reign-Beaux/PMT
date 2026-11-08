import { describe, expect, it } from 'vitest'
import type { Bug } from '../api/bugs'
import type { BugStatus } from './status'
import { BUG_COLUMN_ORDER, BUG_STATUS_LABELS } from './status'
import { groupBugsByStatus } from './groupBugsByStatus'

function makeBug(id: string, status: BugStatus, updatedAt: string): Bug {
  return {
    id,
    project_id: 'p1',
    title: `Bug ${id}`,
    description: '',
    steps_to_reproduce: '',
    product_version: '1.0.0',
    status,
    created_at: '2026-11-01T09:00:00-06:00',
    updated_at: updatedAt,
  }
}

describe('groupBugsByStatus', () => {
  it('con [] devuelve las nueve columnas vacías en orden con su etiqueta', () => {
    const columns = groupBugsByStatus([])

    expect(columns).toHaveLength(9)
    expect(columns.map((column) => column.status)).toEqual([
      ...BUG_COLUMN_ORDER,
    ])
    expect(columns.map((column) => column.label)).toEqual(
      BUG_COLUMN_ORDER.map((status) => BUG_STATUS_LABELS[status]),
    )
    for (const column of columns) {
      expect(column.bugs).toEqual([])
    }
  })

  it('reparte un bug de cada estado en su columna, incluidos done y rejected', () => {
    const bugs = BUG_COLUMN_ORDER.map((status, index) =>
      makeBug(`b${index}`, status, '2026-11-01T09:00:00-06:00'),
    )

    const columns = groupBugsByStatus(bugs)

    for (const [index, column] of columns.entries()) {
      expect(column.bugs.map((bug) => bug.id)).toEqual([`b${index}`])
    }
    const done = columns.find((column) => column.status === 'done')
    expect(done?.label).toBe('Corregido')
    expect(done?.bugs).toHaveLength(1)
    const rejected = columns.find((column) => column.status === 'rejected')
    expect(rejected?.label).toBe('Rechazado')
    expect(rejected?.bugs).toHaveLength(1)
  })

  it('ordena por updated_at descendente dentro de la columna', () => {
    const bugs = [
      makeBug('viejo', 'reported', '2026-11-01T09:00:00-06:00'),
      makeBug('nuevo', 'reported', '2026-11-05T09:00:00-06:00'),
      makeBug('medio', 'reported', '2026-11-03T09:00:00-06:00'),
    ]

    const columns = groupBugsByStatus(bugs)
    const reported = columns.find((column) => column.status === 'reported')

    expect(reported?.bugs.map((bug) => bug.id)).toEqual([
      'nuevo',
      'medio',
      'viejo',
    ])
  })

  it('con updated_at iguales conserva el orden de entrada', () => {
    const bugs = [
      makeBug('a', 'in_dev', '2026-11-01T09:00:00-06:00'),
      makeBug('b', 'in_dev', '2026-11-01T09:00:00-06:00'),
      makeBug('c', 'in_dev', '2026-11-01T09:00:00-06:00'),
    ]

    const columns = groupBugsByStatus(bugs)
    const inDev = columns.find((column) => column.status === 'in_dev')

    expect(inDev?.bugs.map((bug) => bug.id)).toEqual(['a', 'b', 'c'])
  })

  it('no muta la lista recibida', () => {
    const bugs = [
      makeBug('viejo', 'reported', '2026-11-01T09:00:00-06:00'),
      makeBug('nuevo', 'reported', '2026-11-05T09:00:00-06:00'),
    ]
    const copia = [...bugs]

    groupBugsByStatus(bugs)

    expect(bugs).toEqual(copia)
  })
})
