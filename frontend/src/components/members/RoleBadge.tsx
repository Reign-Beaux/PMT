import type { ProjectRole } from '../../api/projects'
import { ROLE_HELP, ROLE_LABELS, ROLE_TONES } from '../../domain/role'
import { Badge } from '../card/Badge'
import s from './Members.module.scss'

/** Rol de una persona en un proyecto. `mine` lo anuncia como el rol propio. */
export function RoleBadge({ role, mine = false }: { role: ProjectRole; mine?: boolean }) {
  return <span className={s.role} title={ROLE_HELP[role]}>
    {mine && <span className={s.srOnly}>Tu rol: </span>}
    <Badge label={ROLE_LABELS[role]} tone={ROLE_TONES[role]} />
  </span>
}
