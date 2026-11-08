import styles from './Feedback.module.scss'

export interface EmptyProps {
  message: string
}

export function Empty({ message }: EmptyProps) {
  return <p className={styles.inline}>{message}</p>
}
