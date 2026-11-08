import type { Task } from '../api/tasks'
import { getTask } from '../api/tasks'
import { Badge } from '../components/card/Badge'
import { CommentList } from '../components/detail/CommentList'
import { StatusHistory } from '../components/detail/StatusHistory'
import { DetailDialog } from '../components/dialog/DetailDialog'
import { ErrorState, Loading } from '../components/feedback'
import { formatDate } from '../domain/formatDate'
import { TASK_STATUS_LABELS } from '../domain/status'
import { TASK_STATUS_TONES } from '../domain/statusTone'
import { useAsync } from '../hooks/useAsync'
import { TASK_RESOURCES } from '../realtime/refreshResources'
import styles from './BugsView.module.scss'

export function TaskDetailDialog({
  taskId,
  onClose,
}: {
  taskId: string
  onClose: () => void
}) {
  const { status, data, error, retry } = useAsync(
    (signal) => getTask(taskId, { signal }),
    [taskId],
    { refreshOn: TASK_RESOURCES },
  )

  const notFound =
    status === 'error' &&
    error !== undefined &&
    error.kind === 'http' &&
    error.status === 404

  if (notFound) {
    // 404 con datos previos: la tarea desapareció tras un refresco.
    if (data !== undefined) {
      return (
        <DetailDialog open title={data.title} onClose={onClose}>
          <p>Ya no existe</p>
        </DetailDialog>
      )
    }
    return (
      <DetailDialog open title="Tarea no encontrada" onClose={onClose}>
        <p>Tarea no encontrada</p>
      </DetailDialog>
    )
  }
  if (status === 'error' && error !== undefined) {
    return (
      <DetailDialog open title="Tarea" onClose={onClose}>
        <ErrorState message={error.message} onRetry={retry} />
        {data !== undefined && <TaskDetail task={data} />}
      </DetailDialog>
    )
  }
  if (data !== undefined) {
    return (
      <DetailDialog open title={data.title} onClose={onClose}>
        <TaskDetail task={data} />
      </DetailDialog>
    )
  }
  return (
    <DetailDialog open title="Tarea" onClose={onClose}>
      <Loading label="Cargando tarea…" />
    </DetailDialog>
  )
}

export function TaskDetail({ task }: { task: Task }) {
  return (
    <div className={styles.detail}>
      <dl className={styles.facts}>
        <dt>Estado</dt>
        <dd>
          <Badge
            label={TASK_STATUS_LABELS[task.status]}
            tone={TASK_STATUS_TONES[task.status]}
          />
        </dd>
        <dt>Asignado a</dt>
        <dd>
          {task.assignee !== undefined && task.assignee !== ''
            ? task.assignee
            : 'Sin asignar'}
        </dd>
        <dt>Creado</dt>
        <dd>{formatDate(task.created_at)}</dd>
        <dt>Actualizado</dt>
        <dd>{formatDate(task.updated_at)}</dd>
      </dl>
      <section aria-label="Descripción">
        <h3 className={styles.sectionTitle}>Descripción</h3>
        <p className={styles.preformatted}>{task.description}</p>
      </section>
      <CommentList itemKind="task" itemId={task.id} />
      <StatusHistory itemKind="task" itemId={task.id} />
    </div>
  )
}
