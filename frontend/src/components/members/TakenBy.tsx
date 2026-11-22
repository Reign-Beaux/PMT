import type { Hold } from '../../api/hold'
import { formatDate } from '../../domain/formatDate'
import { Avatar } from '../avatar/Avatar'
import s from './Members.module.scss'

/**
 * Quién tiene tomada una etapa o un bug y desde cuándo. Sin taken_by no
 * muestra nada: nadie lo tiene.
 */
export function TakenBy({ hold, kind, compact = false, bare = false }: { hold: Hold; kind: 'stage' | 'bug'; compact?: boolean; bare?: boolean }) {
  if (!hold.taken_by) return null
  // bare: sin el verbo, cuando una etiqueta al lado ya dice «Tomado por».
  const verb = bare ? '' : kind === 'stage' ? 'Tomada por ' : 'Tomado por '
  const since = hold.taken_at ? <> · desde <time dateTime={hold.taken_at}>{formatDate(hold.taken_at)}</time></> : null
  if (compact) {
    return <small className={s.takenCompact}>{verb}<span className={s.who}>{hold.taken_by}</span>{since}</small>
  }
  return <span className={s.taken}><Avatar name={hold.taken_by} decorative /><span>{verb}<span className={s.who}>{hold.taken_by}</span>{since}</span></span>
}
