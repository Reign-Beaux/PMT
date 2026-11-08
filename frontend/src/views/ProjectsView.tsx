import { useNavigate } from 'react-router'
import { getProjects } from '../api/projects'
import { Badge } from '../components/card/Badge'
import { Card } from '../components/card/Card'
import { Empty, ErrorState, Loading } from '../components/feedback'
import { PROJECT_STATUS_LABELS } from '../domain/status'
import { PROJECT_STATUS_TONES } from '../domain/statusTone'
import { useAsync } from '../hooks/useAsync'
import { PROJECT_RESOURCES } from '../realtime/refreshResources'
import styles from './ProjectsView.module.scss'

export function ProjectsView() {
  const navigate = useNavigate()
  const { status, data, error, retry } = useAsync(
    (signal) => getProjects({ signal }),
    [],
    { refreshOn: PROJECT_RESOURCES },
  )

  return (
    <section className={styles.view}>
      <h1>Proyectos</h1>
      {status === 'loading' && <Loading label="Cargando proyectos…" />}
      {status === 'error' && error !== undefined && (
        <ErrorState message={error.message} onRetry={retry} />
      )}
      {data !== undefined &&
        (data.length === 0 ? (
          <Empty message="No hay proyectos" />
        ) : (
          <ul className={styles.list}>
            {data.map((project) => (
              <li key={project.id}>
                <Card
                  title={project.name}
                  id={project.id}
                  subtitle={
                    project.product_version !== ''
                      ? `Versión ${project.product_version}`
                      : undefined
                  }
                  meta={
                    <Badge
                      label={PROJECT_STATUS_LABELS[project.status]}
                      tone={PROJECT_STATUS_TONES[project.status]}
                    />
                  }
                  onOpen={() => navigate(`/projects/${project.id}`)}
                />
              </li>
            ))}
          </ul>
        )
      )}
    </section>
  )
}
