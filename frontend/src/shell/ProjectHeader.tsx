import { Link } from 'react-router'
import type { HttpError } from '../api/http'
import type { Project } from '../api/projects'
import { Badge } from '../components/card/Badge'
import { ErrorState, Loading } from '../components/feedback'
import { PROJECT_STATUS_LABELS } from '../domain/status'
import { PROJECT_STATUS_TONES } from '../domain/statusTone'
import type { AsyncStatus } from '../hooks/useAsync'
import styles from './ProjectHeader.module.scss'

export interface ProjectHeaderProps {
  status: AsyncStatus
  project?: Project
  error?: HttpError
  onRetry: () => void
}

export function ProjectHeader({
  status,
  project,
  error,
  onRetry,
}: ProjectHeaderProps) {
  if (
    status === 'error' &&
    error !== undefined &&
    error.kind === 'http' &&
    error.status === 404
  ) {
    return (
      <div className={styles.notFound}>
        <p>Proyecto no encontrado</p>
        <Link to="/">Volver a proyectos</Link>
      </div>
    )
  }

  if (project === undefined) {
    if (status === 'error') {
      return (
        <ErrorState
          message={error?.message ?? 'No se pudo cargar el proyecto'}
          onRetry={onRetry}
        />
      )
    }
    if (status === 'loading') {
      return <Loading label="Cargando proyecto…" />
    }
    return null
  }

  return (
    <>
      {/* Error de refresco: los datos previos siguen visibles bajo el error. */}
      {status === 'error' && (
        <ErrorState
          message={error?.message ?? 'No se pudo cargar el proyecto'}
          onRetry={onRetry}
        />
      )}
      <header className={styles.header}>
        <div className={styles.titleRow}>
          <h1 className={styles.name}>{project.name}</h1>
          <Badge
            label={PROJECT_STATUS_LABELS[project.status]}
            tone={PROJECT_STATUS_TONES[project.status]}
          />
          {project.product_version !== '' && (
            <span className={styles.version}>
              Versión {project.product_version}
            </span>
          )}
        </div>
        {project.description !== '' && (
          <p className={styles.description}>{project.description}</p>
        )}
      </header>
    </>
  )
}
