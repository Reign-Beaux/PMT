import type { ReactNode } from 'react'
import { Link, useLocation } from 'react-router'
import { ConnectionStatus } from '../components/feedback'
import { useConnectionStatus } from '../realtime/eventsContext'
import styles from './Shell.module.scss'

interface ShellProps {
  projectId?: string
  projectName?: string
  stageName?: string
  children: ReactNode
}

export function Shell({
  projectId,
  projectName,
  stageName,
  children,
}: ShellProps) {
  const { pathname } = useLocation()
  const connection = useConnectionStatus()
  const base = projectId !== undefined ? `/projects/${projectId}` : null
  const etapasActive =
    base !== null && (pathname === base || pathname.startsWith(`${base}/stages/`))
  const bugsActive = base !== null && pathname.startsWith(`${base}/bugs`)

  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <Link to="/" className={styles.brand}>
          PMT
        </Link>
        {base !== null && (
          <div className={styles.context}>
            <span>{projectName ?? projectId}</span>
            {stageName !== undefined && <span>{stageName}</span>}
          </div>
        )}
        {base !== null && (
          <nav aria-label="Secciones del proyecto" className={styles.nav}>
            <Link
              to={base}
              className={etapasActive ? `${styles.tab} ${styles.tabActive}` : styles.tab}
              aria-current={etapasActive ? 'page' : undefined}
            >
              Etapas
            </Link>
            <Link
              to={`${base}/bugs`}
              className={bugsActive ? `${styles.tab} ${styles.tabActive}` : styles.tab}
              aria-current={bugsActive ? 'page' : undefined}
            >
              Bugs
            </Link>
          </nav>
        )}
        {connection !== null && (
          <div className={styles.connection}>
            <ConnectionStatus {...connection} />
          </div>
        )}
      </header>
      <main className={styles.main}>{children}</main>
    </div>
  )
}
