import type { ReactNode } from 'react'
import styles from './Kanban.module.scss'

export interface KanbanBoardProps {
  label: string
  children: ReactNode
}

export function KanbanBoard({ label, children }: KanbanBoardProps) {
  return (
    <div
      role="region"
      aria-label={label}
      tabIndex={0}
      className={styles.board}
    >
      {children}
    </div>
  )
}
