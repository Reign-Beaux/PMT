import type { Bug } from '../api/bugs'
import { formatDate } from '../domain/formatDate'
import { BUG_STATUS_LABELS } from '../domain/status'
import { BUG_STATUS_TONES } from '../domain/statusTone'
import { Badge } from '../components/card/Badge'
import { ActivityTabs } from '../components/detail/ActivityTabs'
import s from '../styles/App.module.scss'

export function BugDetail({ bug }: { bug: Bug }) {
  return <>
    <dl className={s.propertyGrid}>
      <div><dt>Estado</dt><dd><Badge label={BUG_STATUS_LABELS[bug.status]} tone={BUG_STATUS_TONES[bug.status]} /></dd></div>
      <div><dt>Versión</dt><dd>{bug.product_version || '—'}</dd></div>
      <div><dt>Reportado por</dt><dd>{bug.reported_by || '—'}</dd></div>
      <div><dt>Creado</dt><dd><time dateTime={bug.created_at}>{formatDate(bug.created_at)}</time></dd></div>
    </dl>
    <section className={s['drawer-section']} aria-label="Descripción"><h3>Descripción</h3><p className={s.preformatted}>{bug.description || 'Sin descripción'}</p></section>
    <section className={s['drawer-section']} aria-label="Pasos para reproducir"><h3>Pasos para reproducir</h3><p className={s.preformatted}>{bug.steps_to_reproduce}</p></section>
    <ActivityTabs itemKind="bug" itemId={bug.id} />
  </>
}
