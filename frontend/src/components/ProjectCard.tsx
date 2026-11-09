import { Link } from 'react-router'
import type { Project } from '../api/projects'
import { getProjectProgress, type Progress } from '../api/progress'
import { useAsync } from '../hooks/useAsync'
import { BUG_RESOURCES, PROJECT_RESOURCES, STAGE_RESOURCES } from '../realtime/refreshResources'
import { summarize } from '../domain/summary'
import { formatDate } from '../domain/formatDate'
import { PROJECT_STATUS_LABELS } from '../domain/status'
import { PROJECT_STATUS_TONES } from '../domain/statusTone'
import { Badge } from './card/Badge'
import { QueryState } from './QueryState'
import { Icon } from './Icon'
import s from '../styles/App.module.scss'

export function ProjectCard({ project, progress }: { project: Project; progress?: Progress }) {
  const summary = progress ? summarize(progress) : undefined
  return <Link className={`${s.panel} ${s['project-card']}`} to={`/projects/${project.id}`}>
    <div className={s['project-card-top']}><span className={s['project-logo']} aria-hidden="true">{project.name.slice(0, 1).toUpperCase()}</span><div><h2>{project.name}</h2><div className={s['heading-line']}><Badge label={PROJECT_STATUS_LABELS[project.status]} tone={PROJECT_STATUS_TONES[project.status]} />{project.product_version && <span className={`${s.version} ${s.mono}`}>v{project.product_version}</span>}</div></div><Icon name="arrow" /></div>
    {project.description && <p className={s.cardDescription}>{project.description}</p>}
    {summary && <><div className={s.cardProgressLabel}><span>Avance del proyecto</span><strong>{summary.done} de {summary.denominator} tareas · {summary.percent}%</strong></div><div className={`${s.track} ${s.purple}`}><span style={{ width: `${summary.percent}%` }} /></div></>}
    <div className={s['project-card-bottom']}><span>Actualizado <time dateTime={project.updated_at}>{formatDate(project.updated_at)}</time></span><span className={s['text-link']}>Abrir proyecto <Icon name="arrow" /></span></div>
  </Link>
}

export function ProjectCardWithProgress({ project }: { project: Project }) {
  const progress = useAsync(signal => getProjectProgress(project.id, { signal }), [project.id], { refreshOn: [...PROJECT_RESOURCES, ...STAGE_RESOURCES, ...BUG_RESOURCES] })
  return <div><ProjectCard project={project} progress={progress.data} />{progress.status === 'error' && <QueryState result={progress} label="Cargando avance…" />}</div>
}
