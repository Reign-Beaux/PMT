import type { Bug } from '../api/bugs'
import { Badge } from '../components/card/Badge'
import { CommentList } from '../components/detail/CommentList'
import { StatusHistory } from '../components/detail/StatusHistory'
import { formatDate } from '../domain/formatDate'
import { BUG_STATUS_LABELS } from '../domain/status'
import { BUG_STATUS_TONES } from '../domain/statusTone'
import styles from './BugsView.module.scss'

export function BugDetail({ bug }: { bug: Bug }) {
  return (
    <div className={styles.detail}>
      <dl className={styles.facts}>
        <dt>Estado</dt>
        <dd>
          <Badge
            label={BUG_STATUS_LABELS[bug.status]}
            tone={BUG_STATUS_TONES[bug.status]}
          />
        </dd>
        <dt>Versión</dt>
        <dd>{bug.product_version}</dd>
        {bug.reported_by !== undefined && bug.reported_by !== '' && (
          <>
            <dt>Reportado por</dt>
            <dd>{bug.reported_by}</dd>
          </>
        )}
        <dt>Creado</dt>
        <dd>{formatDate(bug.created_at)}</dd>
        <dt>Actualizado</dt>
        <dd>{formatDate(bug.updated_at)}</dd>
      </dl>
      <section aria-label="Descripción">
        <h3 className={styles.sectionTitle}>Descripción</h3>
        <p className={styles.preformatted}>{bug.description}</p>
      </section>
      <section aria-label="Pasos para reproducir">
        <h3 className={styles.sectionTitle}>Pasos para reproducir</h3>
        <p className={styles.preformatted}>{bug.steps_to_reproduce}</p>
      </section>
      <CommentList itemKind="bug" itemId={bug.id} />
      <StatusHistory itemKind="bug" itemId={bug.id} />
    </div>
  )
}
