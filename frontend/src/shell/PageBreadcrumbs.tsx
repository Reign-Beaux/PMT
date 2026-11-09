import { Link } from 'react-router'
import { Icon } from '../components/Icon'
import { ConnectionStatus } from '../components/feedback'
import { useConnectionStatus } from '../realtime/eventsContext'
import s from '../styles/App.module.scss'

export function PageBreadcrumbs({ items }: { items: readonly { label: string; to?: string }[] }) {
  const connection = useConnectionStatus()
  return <header className={s.topbar}><nav className={s.breadcrumbs} aria-label="Ruta de navegación">
    <Link to="/dashboard" aria-label="Dashboard"><Icon name="home" /></Link>
    {items.map((item, index) => <span className={s.breadcrumbItem} key={`${index}-${item.label}`}><span className={s.sep}>/</span>{item.to ? <Link to={item.to}>{item.label}</Link> : <strong aria-current="page">{item.label}</strong>}</span>)}
  </nav>{connection !== null && <div className={s.topbarConnection}><ConnectionStatus {...connection} /></div>}</header>
}
