import type { ReactNode } from 'react'
import { Empty } from '../feedback'
import styles from './Kanban.module.scss'

export interface KanbanColumnProps {
  id: string
  label: string
  count: number
  children?: ReactNode
}

export function KanbanColumn({ id, label, count, children }: KanbanColumnProps) {
  const headingId = `kanban-col-${id}`
  return (
    <section className={styles.column} aria-labelledby={headingId}>
      <h3 id={headingId} className={styles.columnHeader}>
        <span className={styles.columnLabel}>{label}</span>
        <span className={styles.columnCount} aria-label={`${count} elementos`}>
          {count}
        </span>
      </h3>
      <div className={styles.columnBody}>
        {count === 0 ? <Empty message="Sin elementos" /> : children}
      </div>
    </section>
  )
}
