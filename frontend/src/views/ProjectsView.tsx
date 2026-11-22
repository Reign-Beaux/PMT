import { useLocation, useNavigate } from 'react-router'
import { getProjects } from '../api/projects'
import { useAsync } from '../hooks/useAsync'
import { PROJECT_RESOURCES } from '../realtime/refreshResources'
import { ProjectCardWithProgress } from '../components/ProjectCard'
import { QueryState } from '../components/QueryState'
import { Empty } from '../components/feedback'
import { PageBreadcrumbs } from '../shell/PageBreadcrumbs'
import s from '../styles/App.module.scss'
import m from '../components/members/Members.module.scss'

function noticeOf(state: unknown): string | undefined {
  if (typeof state !== 'object' || state === null || !('notice' in state)) return undefined
  const notice = (state as { notice: unknown }).notice
  return typeof notice === 'string' && notice !== '' ? notice : undefined
}

export function ProjectsView() {
  const projects = useAsync(signal => getProjects({ signal }), [], { refreshOn: PROJECT_RESOURCES })
  const location = useLocation()
  const navigate = useNavigate()
  const notice = noticeOf(location.state)
  return <><PageBreadcrumbs items={[{ label: 'Projects' }]} /><main id="contenido" className={s.main}>
    <header className={s['page-heading']}><h1>Proyectos</h1></header>
    {notice && <div className={m.notice} role="status"><span>{notice}</span><button type="button" className={m.secondary} onClick={() => navigate('.', { replace: true, state: null })}>Entendido</button></div>}
    <QueryState result={projects} label="Cargando proyectos…" />
    {projects.data && <><div className={s['section-toolbar']}><h2>Todos los proyectos <span className={s.count}>{projects.data.length}</span></h2></div>{projects.data.length === 0 ? <Empty message="No hay proyectos" /> : <div className={s.projectGrid}>{projects.data.map(project => <ProjectCardWithProgress key={project.id} project={project} />)}</div>}</>}
  </main></>
}
