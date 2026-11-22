import { releaseBug, type Bug } from '../api/bugs'
import { formatDate } from '../domain/formatDate'
import { BUG_STATUS_LABELS } from '../domain/status'
import { BUG_STATUS_TONES } from '../domain/statusTone'
import { Badge } from '../components/card/Badge'
import { ActivityTabs } from '../components/detail/ActivityTabs'
import { TakenBy } from '../components/members/TakenBy'
import { ReleaseControl } from '../components/members/ReleaseControl'
import { useProjectContext } from '../shell/useProjectContext'
import m from '../components/members/Members.module.scss'
import s from '../styles/App.module.scss'

export function BugDetail({ bug }: { bug: Bug }) {
  const { project, bugs } = useProjectContext()
  return <>
    <dl className={s.propertyGrid}>
      <div><dt>Estado</dt><dd><Badge label={BUG_STATUS_LABELS[bug.status]} tone={BUG_STATUS_TONES[bug.status]} /></dd></div>
      <div><dt>Versión</dt><dd>{bug.product_version || '—'}</dd></div>
      <div><dt>Reportado por</dt><dd>{bug.reported_by || '—'}</dd></div>
      <div><dt>Creado</dt><dd><time dateTime={bug.created_at}>{formatDate(bug.created_at)}</time></dd></div>
      <div><dt>Tomado por</dt><dd>{bug.taken_by ? <TakenBy hold={bug} kind="bug" bare /> : 'Nadie'}</dd></div>
    </dl>
    {bug.taken_by && project.role === 'owner' && <div className={m.releaseBlock}><ReleaseControl key={`${bug.id}:${bug.taken_by}`} kind="bug" holder={bug.taken_by} onRelease={async reason => { await releaseBug(bug.id, reason); bugs.retry() }} /></div>}
    <section className={s['drawer-section']} aria-label="Descripción"><h3>Descripción</h3><p className={s.preformatted}>{bug.description || 'Sin descripción'}</p></section>
    <section className={s['drawer-section']} aria-label="Pasos para reproducir"><h3>Pasos para reproducir</h3><p className={s.preformatted}>{bug.steps_to_reproduce}</p></section>
    <ActivityTabs itemKind="bug" itemId={bug.id} />
  </>
}
