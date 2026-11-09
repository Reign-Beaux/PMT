import type { ReactNode } from 'react'
import type { Progress } from '../api/progress'
import { summarize } from '../domain/summary'
import { Icon } from './Icon'
import s from '../styles/App.module.scss'

export function Metric({ label, value, detail, icon = 'layers' }: { label: string; value: ReactNode; detail?: ReactNode; icon?: 'layers' | 'check' | 'bug' | 'grid' | 'activity' }) {
  return <article className={`${s.panel} ${s.metric}`}><div className={s['metric-label']}>{label}<Icon name={icon} /></div><div className={s['metric-value']}>{value}</div>{detail !== undefined && <p className={s['metric-bottom']}>{detail}</p>}</article>
}

export function ProjectMetrics({ progress }: { progress: Progress }) {
  const summary = summarize(progress)
  return <section className={s.metrics} aria-label="Resumen del proyecto">
    <article className={`${s.panel} ${s.metric} ${s['progress-metric']}`}>
      <div><div className={s['metric-label']}>Avance del proyecto</div><div className={s['metric-value']}>{summary.percent}<small>%</small></div><p className={s['metric-bottom']}>{summary.done} de {summary.denominator} tareas terminadas</p></div>
      <div className={s.ring} style={{ background: `conic-gradient(var(--accent) ${summary.percent}%, #383347 0)` }} aria-hidden="true"><Icon name="check" /></div>
    </article>
    <Metric label="Etapas completadas" value={<>{summary.completedStages} <small>/ {progress.stages.length}</small></>} detail={`${summary.activeStages} en desarrollo`} />
    <Metric label="Tareas terminadas" icon="check" value={<>{summary.done} <small>/ {summary.denominator}</small></>} detail={`${summary.counts.in_review ?? 0} en revisión`} />
    <Metric label="Bugs abiertos" icon="bug" value={summary.openBugs} />
  </section>
}
