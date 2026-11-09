import { useState } from 'react'
import { Link, useParams } from 'react-router'
import { getStageTasks } from '../api/tasks'
import { useAsync } from '../hooks/useAsync'
import { TASK_RESOURCES } from '../realtime/refreshResources'
import { useDetailSelection } from '../routes/useDetailSelection'
import { useProjectContext } from '../shell/useProjectContext'
import { stageProgressDenominator, stageProgressPercent } from '../components/progress/stageProgress'
import { STAGE_STATUS_LABELS } from '../domain/status'
import { STAGE_STATUS_TONES } from '../domain/statusTone'
import { QueryState } from '../components/QueryState'
import { Empty } from '../components/feedback'
import { Badge } from '../components/card/Badge'
import { Filters } from '../components/Filters'
import { Icon } from '../components/Icon'
import { TaskTable } from './TaskTable'
import { TaskDetailDialog } from './TaskDetail'
import s from '../styles/App.module.scss'

export function StageView() {
  const { stageId } = useParams()
  const { project, stages } = useProjectContext()
  const tasks = useAsync(signal => getStageTasks(stageId!, { signal }), [stageId], { refreshOn: TASK_RESOURCES })
  const { selection, select, clear } = useDetailSelection()
  const [filter, setFilter] = useState<'all' | 'active' | 'done'>('all')
  const stage = stages.data?.stages.find(candidate => candidate.id === stageId)
  const numbers = { total: tasks.data?.length ?? 0, done: tasks.data?.filter(task => task.status === 'done').length ?? 0, cancelled: tasks.data?.filter(task => task.status === 'cancelled').length ?? 0 }
  const percent = stageProgressPercent(numbers)
  const items = tasks.data?.filter(task => filter === 'all' || (filter === 'done' ? task.status === 'done' : !['done', 'cancelled'].includes(task.status)))
  const missing = tasks.error?.status === 404 || (stages.data !== undefined && !stage)

  return <>
    <QueryState result={stages} label="Cargando etapa…" />
    {missing ? <section><h1>Etapa no encontrada</h1><Link to={`/projects/${project.id}`}>Volver al proyecto</Link></section> : <>
      {stage && <>
        <header className={s['page-heading']}><div><p className={s.eyebrow}>Etapa {String(stage.position).padStart(2, '0')} / {project.name}</p><div className={s['heading-line']}><h1>{stage.name}</h1><Badge label={STAGE_STATUS_LABELS[stage.status]} tone={STAGE_STATUS_TONES[stage.status]} /></div>
          {stage.description && <p className={`${s['heading-subtitle']} ${s['project-heading-description']}`}>{stage.description}</p>}
          <p className={s['heading-subtitle']}>Responsable: {stage.lead || 'Sin asignar'}</p>
        </div></header>
        <div className={`${s.panel} ${s['stage-summary']}`}><div className={s['stage-summary-left']}><Icon name="layers" /><span><strong>{numbers.done} de {stageProgressDenominator(numbers)} tareas</strong> terminadas</span></div><div className={s['stage-summary-right']}><div className={`${s.track} ${s.purple}`} role="progressbar" aria-label="Avance de la etapa" aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent}><span style={{ width: `${percent}%` }} /></div><strong>{percent}%</strong><span>de avance</span></div></div>
      </>}
      <QueryState result={tasks} label="Cargando tareas…" />
      {stage && tasks.data && <section aria-label="Tareas de la etapa"><div className={s['section-toolbar']}><h2>Tareas <span className={s.count}>{tasks.data.length}</span></h2><Filters label="Filtrar tareas" value={filter} onChange={setFilter} options={[{ value: 'all', label: 'Todas' }, { value: 'active', label: 'En curso' }, { value: 'done', label: 'Terminadas' }]} /></div>{items?.length ? <TaskTable tasks={items} onOpen={id => select({ kind: 'task', id })} /> : <Empty message={tasks.data.length === 0 ? 'Esta etapa no tiene tareas' : 'No hay tareas para este filtro'} />}</section>}
    </>}
    {selection?.kind === 'task' && <TaskDetailDialog key={selection.id} taskId={selection.id} onClose={clear} />}
  </>
}
