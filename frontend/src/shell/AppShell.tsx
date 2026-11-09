import { Link, NavLink, Outlet } from 'react-router'
import { Icon } from '../components/Icon'
import s from '../styles/App.module.scss'

export function AppShell() {
  return <>
    <a className={s.skip} href="#contenido">Saltar al contenido</a>
    <aside className={s.sidebar}>
      <Link className={s.brand} to="/dashboard" aria-label="PMT, Dashboard"><span className={s['brand-mark']} aria-hidden="true"><i /><i /><i /></span>PMT</Link>
      <nav aria-label="Navegación principal">
        <NavLink to="/dashboard" className={({ isActive }) => `${s['nav-link']} ${isActive ? s.active : ''}`}><Icon name="activity" />Dashboard</NavLink>
        <NavLink to="/projects" className={({ isActive }) => `${s['nav-link']} ${isActive ? s.active : ''}`}><Icon name="grid" />Projects</NavLink>
      </nav>
    </aside>
    <div className={s.app}><Outlet /></div>
  </>
}
