import { useState } from 'react'
import { useProjectContext } from '../shell/useProjectContext'
import { useDetailSelection } from '../routes/useDetailSelection'
import { BUG_STATUS_LABELS } from '../domain/status'
import { BUG_STATUS_TONES } from '../domain/statusTone'
import { formatShortDate } from '../domain/formatDate'
import { Badge } from '../components/card/Badge'
import { Icon } from '../components/Icon'
import { Filters } from '../components/Filters'
import { QueryState } from '../components/QueryState'
import { Empty } from '../components/feedback'
import { DetailDialog } from '../components/dialog/DetailDialog'
import { BugDetail } from './BugDetail'
import s from '../styles/App.module.scss'

export function BugsView() {
  const { bugs } = useProjectContext()
  const { selection, select, clear } = useDetailSelection()
  const [filter, setFilter] = useState<'all' | 'open' | 'done'>('all')
  const [seenIds, setSeenIds] = useState<string[]>([])
  const selected = selection?.kind === 'bug' ? bugs.data?.find(bug => bug.id === selection.id) : undefined
  if (selected && !seenIds.includes(selected.id)) setSeenIds([...seenIds, selected.id])
  const items = bugs.data?.filter(bug => filter === 'all' || (filter === 'done' ? bug.status === 'done' : !['done', 'rejected'].includes(bug.status)))
  return <>
    <section aria-label="Bugs del proyecto">
      <div className={s['section-toolbar']}><h2>Bugs {bugs.data && <span className={s.count}>{bugs.data.length}</span>}</h2><Filters label="Filtrar bugs" value={filter} onChange={setFilter} options={[{ value: 'all', label: 'Todos' }, { value: 'open', label: 'Abiertos' }, { value: 'done', label: 'Corregidos' }]} /></div>
      <QueryState result={bugs} label="Cargando bugs…" />
      {bugs.data && (items?.length ? <div className={`${s.panel} ${s['table-wrap']}`}><table className={s.bugTable}>
        <caption className={s['sr-only']}>Bugs del proyecto</caption><thead><tr><th scope="col">Bug</th><th scope="col">Estado</th><th scope="col">Versión</th><th scope="col">Reportado por</th><th scope="col">Creado</th></tr></thead>
        <tbody>{items.map(bug => <tr key={bug.id}><td><div className={s.taskNameCell}><Icon name="bug" /><button type="button" className={s.taskTitle} onClick={() => select({ kind: 'bug', id: bug.id })}>{bug.title}</button></div></td><td><Badge label={BUG_STATUS_LABELS[bug.status]} tone={BUG_STATUS_TONES[bug.status]} /></td><td>{bug.product_version || '—'}</td><td>{bug.reported_by || '—'}</td><td><time dateTime={bug.created_at}>{formatShortDate(bug.created_at)}</time></td></tr>)}</tbody>
      </table></div> : <Empty message={bugs.data.length === 0 ? 'Este proyecto no tiene bugs' : 'No hay bugs para este filtro'} />)}
    </section>
    {selection?.kind === 'bug' && bugs.data && <DetailDialog key={selection.id} open title={selected?.title ?? (seenIds.includes(selection.id) ? 'Ya no existe' : 'Bug no encontrado')} onClose={clear}>{selected ? <BugDetail bug={selected} /> : <p>{seenIds.includes(selection.id) ? 'Ya no existe' : 'Bug no encontrado'}</p>}</DetailDialog>}
  </>
}
