import { Link, NavLink, Outlet } from 'react-router'
import { Icon } from '../components/Icon'
import { useCurrentUser } from './authContext'
import s from '../styles/App.module.scss'

export function AppShell() {
  const user = useCurrentUser()
  return <>
    <a className={s.skip} href="#contenido">Saltar al contenido</a>
    <aside className={s.sidebar}>
      <Link className={s.brand} to="/dashboard" aria-label="PMT, Dashboard"><span className={s['brand-mark']} aria-hidden="true"><i /><i /><i /></span>PMT</Link>
      <nav aria-label="Navegación principal">
        <NavLink to="/dashboard" className={({ isActive }) => `${s['nav-link']} ${isActive ? s.active : ''}`}><Icon name="activity" />Dashboard</NavLink>
        <NavLink to="/projects" className={({ isActive }) => `${s['nav-link']} ${isActive ? s.active : ''}`}><Icon name="grid" />Projects</NavLink>
        <NavLink to="/cuenta" className={({ isActive }) => `${s['nav-link']} ${s['nav-account']} ${isActive ? s.active : ''}`}><Icon name="user" />Cuenta</NavLink>
      </nav>
      <div className={s['sidebar-bottom']}><Link className={s['sidebar-user']} to="/cuenta"><strong>{user.name}</strong><small>{user.username}</small></Link></div>
    </aside>
    <div className={s.app}><Outlet /></div>
  </>
}
