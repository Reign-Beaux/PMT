import styles from './Feedback.module.scss'

export type LoadingVariant = 'text' | 'list' | 'detail'

export interface LoadingProps {
  label?: string
  variant?: LoadingVariant
}

export function Loading({ label, variant = 'text' }: LoadingProps) {
  const text = label ?? 'Cargando…'
  if (variant === 'text') {
    return (
      <p role="status" className={styles.inline}>
        {text}
      </p>
    )
  }
  return (
    <div className={styles.block}>
      <p role="status" className={styles.inline}>
        {text}
      </p>
      {variant === 'list' && (
        <div aria-hidden="true" className={styles.skeletonList}>
          {Array.from({ length: 4 }, (_, i) => (
            <div key={i} className={`${styles.skeleton} ${styles.row}`} />
          ))}
        </div>
      )}
      {variant === 'detail' && (
        <div aria-hidden="true" className={styles.skeletonDetail}>
          <div className={styles.skeletonDetailMain}>
            <div className={`${styles.skeleton} ${styles.title}`} />
            {Array.from({ length: 3 }, (_, i) => (
              <div key={i} className={`${styles.skeleton} ${styles.row}`} />
            ))}
          </div>
          <div className={`${styles.skeleton} ${styles.side}`} />
        </div>
      )}
    </div>
  )
}
