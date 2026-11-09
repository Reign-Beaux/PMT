import { getProjects } from '../api/projects'
import { useAsync } from '../hooks/useAsync'
import { PROJECT_RESOURCES } from '../realtime/refreshResources'
import { ProjectCardWithProgress } from '../components/ProjectCard'
import { QueryState } from '../components/QueryState'
import { Empty } from '../components/feedback'
import { PageBreadcrumbs } from '../shell/PageBreadcrumbs'
import s from '../styles/App.module.scss'

export function ProjectsView() {
  const projects = useAsync(signal => getProjects({ signal }), [], { refreshOn: PROJECT_RESOURCES })
  return <><PageBreadcrumbs items={[{ label: 'Projects' }]} /><main id="contenido" className={s.main}>
    <header className={s['page-heading']}><h1>Proyectos</h1></header>
    <QueryState result={projects} label="Cargando proyectos…" />
    {projects.data && <><div className={s['section-toolbar']}><h2>Todos los proyectos <span className={s.count}>{projects.data.length}</span></h2></div>{projects.data.length === 0 ? <Empty message="No hay proyectos" /> : <div className={s.projectGrid}>{projects.data.map(project => <ProjectCardWithProgress key={project.id} project={project} />)}</div>}</>}
  </main></>
}
