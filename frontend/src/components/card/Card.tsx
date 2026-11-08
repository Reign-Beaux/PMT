import type { ReactNode } from 'react'
import styles from './Card.module.scss'

export interface CardProps {
  title: string
  id: string
  subtitle?: string
  meta?: ReactNode
  onOpen?: () => void
}

function CardContent({ title, id, subtitle, meta }: CardProps) {
  return (
    <>
      <span className={styles.title}>{title}</span>
      <span className={styles.id}>{id}</span>
      {subtitle !== undefined ? (
        <span className={styles.subtitle}>{subtitle}</span>
      ) : null}
      {meta !== undefined ? <span className={styles.meta}>{meta}</span> : null}
    </>
  )
}

export function Card(props: CardProps) {
  const { onOpen } = props
  if (onOpen) {
    return (
      <button type="button" className={styles.card} onClick={onOpen}>
        <CardContent {...props} />
      </button>
    )
  }
  return (
    <article className={styles.card}>
      <CardContent {...props} />
    </article>
  )
}
