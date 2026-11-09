import type { Task } from '../api/tasks'
import { formatShortDate } from '../domain/formatDate'
import { TASK_STATUS_LABELS } from '../domain/status'
import { TASK_STATUS_TONES } from '../domain/statusTone'
import { Badge } from '../components/card/Badge'
import { Avatar } from '../components/avatar/Avatar'
import { Icon } from '../components/Icon'
import s from '../styles/App.module.scss'

export function TaskTable({ tasks, onOpen }: { tasks: readonly Task[]; onOpen: (id: string) => void }) {
  return <div className={`${s.panel} ${s['table-wrap']}`}><table className={s['task-table']}>
    <caption className={s['sr-only']}>Tareas de la etapa</caption>
    <thead><tr><th scope="col">Tarea</th><th scope="col">Estado</th><th scope="col">Responsable</th><th scope="col">Creada</th><th scope="col">Terminada</th></tr></thead>
    <tbody>{tasks.map(task => <tr key={task.id}>
      <td><div className={s.taskNameCell}><Icon name={task.status === 'done' ? 'check' : 'circle'} /><button type="button" className={s.taskTitle} onClick={() => onOpen(task.id)}>{task.title}</button></div></td>
      <td><Badge label={TASK_STATUS_LABELS[task.status]} tone={TASK_STATUS_TONES[task.status]} /></td>
      <td><span className={s['person-cell']}><Avatar name={task.assignee} decorative /><span>{task.assignee || 'Sin asignar'}</span></span></td>
      <td><time dateTime={task.created_at}>{formatShortDate(task.created_at)}</time></td>
      <td>{task.done_at ? <time dateTime={task.done_at}>{formatShortDate(task.done_at)}</time> : '—'}</td>
    </tr>)}</tbody>
  </table></div>
}
