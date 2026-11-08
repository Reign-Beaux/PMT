import styles from './Feedback.module.scss'

export interface ErrorStateProps {
  message: string
  onRetry: () => void
}

export function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <div role="alert" className={styles.error}>
      <span>{message}</span>
      <button type="button" className={styles.retry} onClick={onRetry}>
        Reintentar
      </button>
    </div>
  )
}
