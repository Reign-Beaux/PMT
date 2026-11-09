import { Link, NavLink, Outlet, useParams } from 'react-router'
import { getProject, getProjectStages } from '../api/projects'
import { getProjectProgress } from '../api/progress'
import { getProjectBugs } from '../api/bugs'
import { useAsync } from '../hooks/useAsync'
import { BUG_RESOURCES, PROJECT_RESOURCES, STAGE_RESOURCES } from '../realtime/refreshResources'
import { Badge } from '../components/card/Badge'
import { Avatar } from '../components/avatar/Avatar'
import { PROJECT_STATUS_LABELS, TASK_STATUS_LABELS } from '../domain/status'
import { PROJECT_STATUS_TONES, TASK_STATUS_TONES } from '../domain/statusTone'
import { formatDate } from '../domain/formatDate'
import { summarize } from '../domain/summary'
import { QueryState } from '../components/QueryState'
import { ProjectMetrics } from '../components/Metrics'
import { Icon } from '../components/Icon'
import { PageBreadcrumbs } from './PageBreadcrumbs'
import type { ProjectContext } from './useProjectContext'
import s from '../styles/App.module.scss'

export function ProjectLayout() {
  const { projectId, stageId } = useParams()
  const base = `/projects/${projectId}`
  const project = useAsync(signal => getProject(projectId!, { signal }), [projectId], { refreshOn: PROJECT_RESOURCES })
  const stages = useAsync(async signal => {
    const [items, progress] = await Promise.all([getProjectStages(projectId!, { signal }), getProjectProgress(projectId!, { signal })])
    return { stages: items, progress }
  }, [projectId], { refreshOn: [...STAGE_RESOURCES, ...BUG_RESOURCES, ...PROJECT_RESOURCES] })
  const bugs = useAsync(signal => getProjectBugs(projectId!, { signal }), [projectId], { refreshOn: BUG_RESOURCES })
  const currentStage = stages.data?.stages.find(stage => stage.id === stageId)
  const breadcrumbs = [{ label: 'Projects', to: '/projects' }, { label: project.data?.name ?? 'Proyecto', to: stageId ? base : undefined }]
  if (stageId) breadcrumbs.push({ label: currentStage?.name ?? 'Etapa', to: undefined })
  const notFound = project.error?.status === 404
  const context: ProjectContext | undefined = project.data ? { project: project.data, stages, bugs } : undefined
  const progress = stages.data?.progress
  const summary = progress ? summarize(progress) : undefined
  const active = stages.data?.stages.find(stage => stage.status === 'in_dev')

  return <>
    <PageBreadcrumbs items={breadcrumbs} />
    <main id="contenido" className={s.main}>
      {notFound ? <section className={s.notFound}><h1>Proyecto no encontrado</h1><Link to="/projects">Volver a proyectos</Link></section> : <QueryState result={project} label="Cargando proyecto…" />}
      {context && !notFound && (stageId ? <Outlet context={context} /> : <>
        <header className={s['page-heading']}><div>
          <div className={s['heading-line']}><h1>{context.project.name}</h1><Badge label={PROJECT_STATUS_LABELS[context.project.status]} tone={PROJECT_STATUS_TONES[context.project.status]} />{context.project.product_version && <span className={`${s.version} ${s.mono}`}>v{context.project.product_version}</span>}</div>
          {context.project.description && <p className={`${s['heading-subtitle']} ${s['project-heading-description']}`}>{context.project.description}</p>}
        </div></header>
        <nav className={s['project-tabs']} aria-label="Secciones del proyecto">
          <NavLink to={base} end><Icon name="layers" />Etapas{stages.data && <span className={s.count}>{stages.data.stages.length}</span>}</NavLink>
          <NavLink to={`${base}/bugs`}><Icon name="bug" />Bugs{bugs.data && <span className={s.count}>{bugs.data.length}</span>}</NavLink>
        </nav>
        <QueryState result={stages} label="Cargando avance…" />
        {progress && <ProjectMetrics progress={progress} />}
        <div className={s.columns}><div className={s.listColumn}><Outlet context={context} /></div>
          <aside className={s['right-column']} aria-label="Información del proyecto">
            {summary && <section className={`${s.panel} ${s['aside-panel']}`}><h2>Estado de las tareas</h2><p className={s.muted}>{summary.total} tareas</p>
              <div className={s.statusDistribution}>{Object.entries(TASK_STATUS_LABELS).filter(([key]) => summary.counts[key] > 0).map(([key, label]) => <div className={s.legend} key={key}><Badge label={label} tone={TASK_STATUS_TONES[key as keyof typeof TASK_STATUS_TONES]} /><strong>{summary.counts[key]}</strong></div>)}</div>
              <div className={s['aside-divider']} /><div className={s['detail-pair']}><span>Versión</span><strong>{context.project.product_version || '—'}</strong></div><div className={s['detail-pair']}><span>Actualizado</span><strong><time dateTime={context.project.updated_at}>{formatDate(context.project.updated_at)}</time></strong></div>
            </section>}
            {active && <section className={`${s.panel} ${s['aside-panel']}`}><h2>Responsable de la etapa activa</h2><Link className={s['text-link']} to={`${base}/stages/${active.id}`}>{active.name}</Link><div className={s['team-member']}><Avatar name={active.lead} /><div><strong>{active.lead || 'Sin asignar'}</strong></div></div></section>}
          </aside>
        </div>
      </>)}
    </main>
  </>
}
