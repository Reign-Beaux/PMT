import type { Task } from '../api/tasks'
import type { TaskStatus } from './status'
import { TASK_COLUMN_ORDER, TASK_STATUS_LABELS } from './status'

export interface TaskColumn {
  status: TaskStatus
  label: string
  tasks: Task[]
}

/**
 * Devuelve las nueve columnas en el orden de TASK_COLUMN_ORDER, siempre todas,
 * con las tareas repartidas por status y ordenadas por updated_at descendente
 * (empate → orden de entrada). No muta la lista recibida.
 */
export function groupTasksByStatus(tasks: Task[]): TaskColumn[] {
  return TASK_COLUMN_ORDER.map((status) => ({
    status,
    label: TASK_STATUS_LABELS[status],
    tasks: tasks
      .filter((task) => task.status === status)
      .sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at)),
  }))
}
