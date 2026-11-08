import { useEffect } from 'react'
import { Link, useParams } from 'react-router'
import { HttpError } from '../api/http'
import { getProjectStages } from '../api/projects'
import { getStageTasks } from '../api/tasks'
import { Badge } from '../components/card/Badge'
import { Card } from '../components/card/Card'
import { ErrorState, Loading } from '../components/feedback'
import { KanbanBoard, KanbanColumn } from '../components/kanban'
import { StageProgress } from '../components/progress/StageProgress'
import { groupTasksByStatus } from '../domain/groupTasksByStatus'
import { STAGE_STATUS_LABELS } from '../domain/status'
import { STAGE_STATUS_TONES } from '../domain/statusTone'
import { useAsync } from '../hooks/useAsync'
import { STAGE_RESOURCES } from '../realtime/refreshResources'
import { useDetailSelection } from '../routes/useDetailSelection'
import { useStageName } from '../shell/stageNameContext'
import styles from './StageView.module.scss'
import { TaskDetailDialog } from './TaskDetail'

export function StageView() {
  const { projectId, stageId } = useParams()
  if (projectId === undefined || stageId === undefined) return null
  return <StageBoard projectId={projectId} stageId={stageId} />
}

function StageBoard({
  projectId,
  stageId,
}: {
  projectId: string
  stageId: string
}) {
  const { selection, select, clear } = useDetailSelection()
  const setStageName = useStageName()
  const { status, data, error, retry } = useAsync(
    async (signal) => {
      const [tasks, stages] = await Promise.all([
        // 404 = la etapa no existe: lo maneja la rama "Etapa no encontrada".
        getStageTasks(stageId, { signal }).catch((err: unknown) => {
          if (err instanceof HttpError && err.kind === 'http' && err.status === 404)
            return null
          throw err
        }),
        getProjectStages(projectId, { signal }),
      ])
      return { tasks: tasks ?? [], stages, missing: tasks === null }
    },
    [projectId, stageId],
    { refreshOn: STAGE_RESOURCES },
  )

  const stage =
    data !== undefined && !data.missing
      ? data.stages.find((candidate) => candidate.id === stageId)
      : undefined
  const stageName = stage?.name

  useEffect(() => {
    setStageName(stageName)
    return () => setStageName(undefined)
  }, [stageName, setStageName])

  return (
    <section className={styles.view}>
      {status === 'loading' && <Loading label="Cargando etapa…" />}
      {status === 'error' && error !== undefined && (
        <ErrorState message={error.message} onRetry={retry} />
      )}
      {data !== undefined && stage === undefined && (
        <div className={styles.notFound}>
          <p>Etapa no encontrada</p>
          <Link to={`/projects/${projectId}`}>Volver al proyecto</Link>
        </div>
      )}
      {data !== undefined && stage !== undefined && (
        <>
          <header className={styles.header}>
            <h2 className={styles.name}>{stage.name}</h2>
            <Badge
              label={STAGE_STATUS_LABELS[stage.status]}
              tone={STAGE_STATUS_TONES[stage.status]}
            />
            <StageProgress
              done={data.tasks.filter((task) => task.status === 'done').length}
              cancelled={
                data.tasks.filter((task) => task.status === 'cancelled').length
              }
              total={data.tasks.length}
            />
          </header>
          <KanbanBoard label="Tablero de la etapa">
            {groupTasksByStatus(data.tasks).map((column) => (
              <KanbanColumn
                key={column.status}
                id={column.status}
                label={column.label}
                count={column.tasks.length}
              >
                <ul className={styles.cards}>
                  {column.tasks.map((task) => (
                    <li key={task.id}>
                      <Card
                        title={task.title}
                        id={task.id}
                        subtitle={
                          task.assignee !== undefined && task.assignee !== ''
                            ? task.assignee
                            : 'Sin asignar'
                        }
                        onOpen={() => select({ kind: 'task', id: task.id })}
                      />
                    </li>
                  ))}
                </ul>
              </KanbanColumn>
            ))}
          </KanbanBoard>
        </>
      )}
      {/* Hermano del tablero y fuera de sus estados: recargar con ?task= abre
          el detalle aunque la etapa aún cargue o falle. */}
      {selection?.kind === 'task' && (
        <TaskDetailDialog taskId={selection.id} onClose={clear} />
      )}
    </section>
  )
}
