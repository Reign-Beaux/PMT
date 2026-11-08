import type { Bug } from '../api/bugs'
import type { BugStatus } from './status'
import { BUG_COLUMN_ORDER, BUG_STATUS_LABELS } from './status'

export interface BugColumn {
  status: BugStatus
  label: string
  bugs: Bug[]
}

/**
 * Devuelve las nueve columnas en el orden de BUG_COLUMN_ORDER, siempre todas,
 * con los bugs repartidos por status y ordenados por updated_at descendente
 * (empate → orden de entrada). No muta la lista recibida.
 */
export function groupBugsByStatus(bugs: Bug[]): BugColumn[] {
  return BUG_COLUMN_ORDER.map((status) => ({
    status,
    label: BUG_STATUS_LABELS[status],
    bugs: bugs
      .filter((bug) => bug.status === status)
      .sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at)),
  }))
}
