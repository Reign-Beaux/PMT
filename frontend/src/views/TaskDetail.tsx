import { getTask, type Task } from '../api/tasks'
import { useAsync } from '../hooks/useAsync'
import { TASK_RESOURCES } from '../realtime/refreshResources'
import { formatDate } from '../domain/formatDate'
import { TASK_STATUS_LABELS } from '../domain/status'
import { TASK_STATUS_TONES } from '../domain/statusTone'
import { Badge } from '../components/card/Badge'
import { Avatar } from '../components/avatar/Avatar'
import { ActivityTabs } from '../components/detail/ActivityTabs'
import { DetailDialog } from '../components/dialog/DetailDialog'
import { QueryState } from '../components/QueryState'
import s from '../styles/App.module.scss'

function TaskDetail({ task }: { task: Task }) {
  return <>
    <dl className={s.propertyGrid}>
      <div><dt>Estado</dt><dd><Badge label={TASK_STATUS_LABELS[task.status]} tone={TASK_STATUS_TONES[task.status]} /></dd></div>
      <div><dt>Responsable</dt><dd className={s['person-cell']}><Avatar name={task.assignee} decorative />{task.assignee || 'Sin asignar'}</dd></div>
      <div><dt>Creada</dt><dd><time dateTime={task.created_at}>{formatDate(task.created_at)}</time></dd></div>
      <div><dt>Terminada</dt><dd>{task.done_at ? <time dateTime={task.done_at}>{formatDate(task.done_at)}</time> : '—'}</dd></div>
    </dl>
    <section className={s['drawer-section']} aria-label="Descripción"><h3>Descripción</h3><p className={s.preformatted}>{task.description || 'Sin descripción'}</p></section>
    <ActivityTabs itemKind="task" itemId={task.id} />
  </>
}

export function TaskDetailDialog({ taskId, onClose }: { taskId: string; onClose: () => void }) {
  const result = useAsync(signal => getTask(taskId, { signal }), [taskId], { refreshOn: TASK_RESOURCES })
  const missing = result.error?.status === 404
  return <DetailDialog open title={missing ? (result.data ? result.data.title : 'Tarea no encontrada') : result.data?.title ?? 'Tarea'} onClose={onClose}>
    {missing ? <p>{result.data ? 'Ya no existe' : 'Tarea no encontrada'}</p> : <><QueryState result={result} label="Cargando tarea…" />{result.data && <TaskDetail task={result.data} />}</>}
  </DetailDialog>
}
