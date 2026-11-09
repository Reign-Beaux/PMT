import { Link } from 'react-router'
import { getProjects } from '../api/projects'
import { getProjectProgress } from '../api/progress'
import { useAsync } from '../hooks/useAsync'
import { PROJECT_RESOURCES, STAGE_RESOURCES, BUG_RESOURCES } from '../realtime/refreshResources'
import { summarize } from '../domain/summary'
import { QueryState } from '../components/QueryState'
import { ProjectCard } from '../components/ProjectCard'
import { Metric } from '../components/Metrics'
import { Empty } from '../components/feedback'
import { Icon } from '../components/Icon'
import { PageBreadcrumbs } from '../shell/PageBreadcrumbs'
import s from '../styles/App.module.scss'

export function DashboardView() {
  const result = useAsync(async signal => {
    const projects = await getProjects({ signal })
    return Promise.all(projects.map(async project => ({ project, progress: await getProjectProgress(project.id, { signal }) })))
  }, [], { refreshOn: [...PROJECT_RESOURCES, ...STAGE_RESOURCES, ...BUG_RESOURCES] })
  const entries = result.data
  const summaries = entries?.map(entry => summarize(entry.progress))
  return <><PageBreadcrumbs items={[{ label: 'Dashboard' }]} /><main id="contenido" className={s.main}>
    <header className={s['page-heading']}><h1>Dashboard</h1><Link className={s.button} to="/projects">Ver proyectos <Icon name="arrow" /></Link></header>
    <QueryState result={result} label="Cargando dashboard…" />
    {entries && summaries && <>
      <section className={s.metrics} aria-label="Resumen general">
        <Metric label="Proyectos en operación" icon="grid" value={entries.filter(entry => entry.project.status === 'working').length} />
        <Metric label="Etapas en curso" value={summaries.reduce((sum, summary) => sum + summary.activeStages, 0)} />
        <Metric label="Tareas en revisión" icon="check" value={summaries.reduce((sum, summary) => sum + (summary.counts.in_review ?? 0), 0)} />
        <Metric label="Bugs abiertos" icon="bug" value={summaries.reduce((sum, summary) => sum + summary.openBugs, 0)} />
      </section>
      <div className={s['section-toolbar']}><h2>Proyectos <span className={s.count}>{entries.length}</span></h2><Link className={s['text-link']} to="/projects">Ver todos <Icon name="arrow" /></Link></div>
      {entries.length === 0 ? <Empty message="No hay proyectos" /> : <div className={s.projectGrid}>{entries.map(({ project, progress }) => <ProjectCard key={project.id} project={project} progress={progress} />)}</div>}
    </>}
  </main></>
}
