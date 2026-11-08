import { useNavigate } from 'react-router'
import { getProjectStages } from '../api/projects'
import { getProjectProgress } from '../api/progress'
import { Badge } from '../components/card/Badge'
import { Card } from '../components/card/Card'
import { Empty, ErrorState, Loading } from '../components/feedback'
import { StageProgress } from '../components/progress/StageProgress'
import { STAGE_STATUS_LABELS } from '../domain/status'
import { STAGE_STATUS_TONES } from '../domain/statusTone'
import { useAsync } from '../hooks/useAsync'
import { STAGE_RESOURCES } from '../realtime/refreshResources'
import { sortStagesByPosition, stageProgressFor } from './stageListProgress'
import styles from './StageList.module.scss'

export interface StageListProps {
  projectId: string
}

export function StageList({ projectId }: StageListProps) {
  const navigate = useNavigate()
  const { status, data, error, retry } = useAsync(
    async (signal) => {
      const [stages, progress] = await Promise.all([
        getProjectStages(projectId, { signal }),
        getProjectProgress(projectId, { signal }),
      ])
      return { stages, progress }
    },
    [projectId],
    { refreshOn: STAGE_RESOURCES },
  )

  return (
    <section className={styles.view}>
      <h2>Etapas</h2>
      {status === 'loading' && <Loading label="Cargando etapas…" />}
      {status === 'error' && error !== undefined && (
        <ErrorState message={error.message} onRetry={retry} />
      )}
      {data !== undefined &&
        (data.stages.length === 0 ? (
          <Empty message="Este proyecto no tiene etapas" />
        ) : (
          <ul className={styles.list}>
            {sortStagesByPosition(data.stages).map((stage) => {
              const { done, total, cancelled } = stageProgressFor(
                stage.id,
                data.progress,
              )
              return (
                <li key={stage.id}>
                  <Card
                    title={stage.name}
                    id={stage.id}
                    subtitle={
                      stage.lead !== undefined && stage.lead !== ''
                        ? `TL: ${stage.lead}`
                        : undefined
                    }
                    meta={
                      <>
                        <Badge
                          label={STAGE_STATUS_LABELS[stage.status]}
                          tone={STAGE_STATUS_TONES[stage.status]}
                        />
                        <StageProgress
                          done={done}
                          total={total}
                          cancelled={cancelled}
                        />
                      </>
                    }
                    onOpen={() =>
                      navigate(`/projects/${projectId}/stages/${stage.id}`)
                    }
                  />
                </li>
              )
            })}
          </ul>
        ))}
    </section>
  )
}
