import styles from './Feedback.module.scss'

export interface LoadingProps {
  label?: string
}

export function Loading({ label }: LoadingProps) {
  return (
    <p role="status" className={styles.inline}>
      {label ?? 'Cargando…'}
    </p>
  )
}
