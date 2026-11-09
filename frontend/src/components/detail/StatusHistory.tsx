import { getBugHistory, getTaskHistory } from '../../api/history'
import { formatDate } from '../../domain/formatDate'
import { BUG_STATUS_LABELS, TASK_STATUS_LABELS } from '../../domain/status'
import { BUG_STATUS_TONES, TASK_STATUS_TONES } from '../../domain/statusTone'
import { useAsync } from '../../hooks/useAsync'
import { STATUS_CHANGE_RESOURCES } from '../../realtime/refreshResources'
import type { BadgeTone } from '../card/Badge'
import { Badge } from '../card/Badge'
import { Empty, ErrorState, Loading } from '../feedback'
import styles from './Detail.module.scss'

interface HistoryRow {
  id: string
  from: string
  fromTone: BadgeTone
  to: string
  toTone: BadgeTone
  actor_name: string
  actor_profile: string
  reason: string
  changed_at: string
}

export interface StatusHistoryProps {
  itemKind: 'task' | 'bug'
  itemId: string
}

export function StatusHistory({ itemKind, itemId }: StatusHistoryProps) {
  const { status, data, error, retry } = useAsync(
    async (signal): Promise<HistoryRow[]> => {
      if (itemKind === 'task') {
        const changes = await getTaskHistory(itemId, { signal })
        return changes.map((change) => ({
          id: change.id,
          from: TASK_STATUS_LABELS[change.from_status],
          fromTone: TASK_STATUS_TONES[change.from_status],
          to: TASK_STATUS_LABELS[change.to_status],
          toTone: TASK_STATUS_TONES[change.to_status],
          actor_name: change.actor_name,
          actor_profile: change.actor_profile,
          reason: change.reason,
          changed_at: change.changed_at,
        }))
      }
      const changes = await getBugHistory(itemId, { signal })
      return changes.map((change) => ({
        id: change.id,
        from: BUG_STATUS_LABELS[change.from_status],
        fromTone: BUG_STATUS_TONES[change.from_status],
        to: BUG_STATUS_LABELS[change.to_status],
        toTone: BUG_STATUS_TONES[change.to_status],
        actor_name: change.actor_name,
        actor_profile: change.actor_profile,
        reason: change.reason,
        changed_at: change.changed_at,
      }))
    },
    [itemKind, itemId],
    { refreshOn: STATUS_CHANGE_RESOURCES },
  )

  const rows =
    data !== undefined
      ? [...data].sort(
          (a, b) => Date.parse(a.changed_at) - Date.parse(b.changed_at),
        )
      : []

  return (
    <section aria-label="Historial de estados" className={styles.section}>
      <h3 className={styles.sectionTitle}>Historial de estados</h3>
      {status === 'loading' && <Loading label="Cargando historial…" />}
      {status === 'error' && error !== undefined && (
        <ErrorState message={error.message} onRetry={retry} />
      )}
      {rows.length === 0 && data !== undefined && (
        <Empty message="Sin cambios de estado" />
      )}
      {rows.length > 0 && (
        <ol className={`${styles.list} ${styles.historyList}`}>
          {rows.map((row) => (
            <li key={row.id} className={`${styles.item} ${styles.historyItem}`}>
              <p className={styles.meta}>
                <span>
                  <Badge label={row.from} tone={row.fromTone} />
                  {' → '}
                  <Badge label={row.to} tone={row.toTone} />
                </span>
                <span>
                  {row.actor_profile !== ''
                    ? `${row.actor_name} (${row.actor_profile})`
                    : row.actor_name}
                </span>
                <time dateTime={row.changed_at}>{formatDate(row.changed_at)}</time>
              </p>
              {row.reason.trim() !== '' && (
                <p className={styles.reason}>{row.reason}</p>
              )}
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
