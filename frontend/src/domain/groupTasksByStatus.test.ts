import { describe, expect, it } from 'vitest'
import type { Task } from '../api/tasks'
import type { TaskStatus } from './status'
import { TASK_COLUMN_ORDER, TASK_STATUS_LABELS } from './status'
import { groupTasksByStatus } from './groupTasksByStatus'

function makeTask(id: string, status: TaskStatus, updatedAt: string): Task {
  return {
    id,
    stage_id: 's1',
    title: `Tarea ${id}`,
    description: '',
    status,
    created_at: '2026-11-01T09:00:00-06:00',
    updated_at: updatedAt,
  }
}

describe('groupTasksByStatus', () => {
  it('con [] devuelve las nueve columnas vacías en orden con su etiqueta', () => {
    const columns = groupTasksByStatus([])

    expect(columns).toHaveLength(9)
    expect(columns.map((column) => column.status)).toEqual([
      ...TASK_COLUMN_ORDER,
    ])
    expect(columns.map((column) => column.label)).toEqual(
      TASK_COLUMN_ORDER.map((status) => TASK_STATUS_LABELS[status]),
    )
    for (const column of columns) {
      expect(column.tasks).toEqual([])
    }
  })

  it('reparte una tarea de cada estado en su columna', () => {
    const tasks = TASK_COLUMN_ORDER.map((status, index) =>
      makeTask(`t${index}`, status, '2026-11-01T09:00:00-06:00'),
    )

    const columns = groupTasksByStatus(tasks)

    for (const [index, column] of columns.entries()) {
      expect(column.tasks.map((task) => task.id)).toEqual([`t${index}`])
    }
  })

  it('ordena por updated_at descendente dentro de la columna', () => {
    const tasks = [
      makeTask('vieja', 'in_dev', '2026-11-01T09:00:00-06:00'),
      makeTask('nueva', 'in_dev', '2026-11-05T09:00:00-06:00'),
      makeTask('media', 'in_dev', '2026-11-03T09:00:00-06:00'),
    ]

    const columns = groupTasksByStatus(tasks)
    const inDev = columns.find((column) => column.status === 'in_dev')

    expect(inDev?.tasks.map((task) => task.id)).toEqual([
      'nueva',
      'media',
      'vieja',
    ])
  })

  it('con updated_at iguales conserva el orden de entrada', () => {
    const tasks = [
      makeTask('a', 'done', '2026-11-01T09:00:00-06:00'),
      makeTask('b', 'done', '2026-11-01T09:00:00-06:00'),
      makeTask('c', 'done', '2026-11-01T09:00:00-06:00'),
    ]

    const columns = groupTasksByStatus(tasks)
    const done = columns.find((column) => column.status === 'done')

    expect(done?.tasks.map((task) => task.id)).toEqual(['a', 'b', 'c'])
  })

  it('las tareas canceladas aparecen en la columna "Cancelada"', () => {
    const tasks = [makeTask('x', 'cancelled', '2026-11-01T09:00:00-06:00')]

    const columns = groupTasksByStatus(tasks)
    const cancelled = columns.find(
      (column) => column.status === 'cancelled',
    )

    expect(cancelled?.label).toBe('Cancelada')
    expect(cancelled?.tasks.map((task) => task.id)).toEqual(['x'])
  })

  it('no muta la lista recibida', () => {
    const tasks = [
      makeTask('vieja', 'in_dev', '2026-11-01T09:00:00-06:00'),
      makeTask('nueva', 'in_dev', '2026-11-05T09:00:00-06:00'),
    ]
    const copia = [...tasks]

    groupTasksByStatus(tasks)

    expect(tasks).toEqual(copia)
  })
})
