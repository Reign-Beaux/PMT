import { Link } from 'react-router'
import { PageBreadcrumbs } from '../shell/PageBreadcrumbs'
import s from '../styles/App.module.scss'

export function NotFoundView() {
  return <><PageBreadcrumbs items={[{ label: 'Página no encontrada' }]} /><main id="contenido" className={s.main}><h1>Página no encontrada</h1><Link className={s['text-link']} to="/projects">Volver a proyectos</Link></main></>
}
