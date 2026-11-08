import styles from './Feedback.module.scss'

export type ConnectionState = 'connected' | 'reconnecting' | 'stale'

export interface ConnectionStatusProps {
  status: ConnectionState
  lastSyncedAt?: string
}

const TIME_FORMAT = new Intl.DateTimeFormat('es', {
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})

function connectionText(status: ConnectionState, lastSyncedAt?: string): string {
  if (status === 'connected') return 'En vivo'
  if (status === 'reconnecting') return 'Reconectando…'
  const syncedDate =
    lastSyncedAt !== undefined && !Number.isNaN(Date.parse(lastSyncedAt))
      ? new Date(lastSyncedAt)
      : null
  return syncedDate === null
    ? 'Sin conexión'
    : `Sin conexión, datos de las ${TIME_FORMAT.format(syncedDate)}`
}

export function ConnectionStatus({ status, lastSyncedAt }: ConnectionStatusProps) {
  return (
    <p role="status" className={styles.inline}>
      {connectionText(status, lastSyncedAt)}
    </p>
  )
}
