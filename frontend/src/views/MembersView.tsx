import { useState, type FormEvent } from 'react'
import { addProjectMember, getProjectMembers, removeProjectMember, updateProjectMember, type SharedRole } from '../api/members'
import { QueryState } from '../components/QueryState'
import { RoleBadge } from '../components/members/RoleBadge'
import { ROLE_HELP } from '../domain/role'
import { useAsync } from '../hooks/useAsync'
import { PROJECT_RESOURCES } from '../realtime/refreshResources'
import { useCurrentUser } from '../shell/authContext'
import { useProjectContext } from '../shell/useProjectContext'
import s from '../components/members/Members.module.scss'
import app from '../styles/App.module.scss'

function message(err: unknown): string {
  return err instanceof Error ? err.message : 'Ocurrió un error inesperado'
}

const SHARED_ROLES: { value: SharedRole; label: string }[] = [
  { value: 'editor', label: 'Editor' },
  { value: 'viewer', label: 'Viewer' },
]

/**
 * Miembros del proyecto. Todos los ven; solo el owner agrega, cambia el rol y
 * quita. Las escrituras van con la cookie, sin perfil declarado.
 */
export function MembersView() {
  const { project } = useProjectContext()
  const me = useCurrentUser()
  const isOwner = project.role === 'owner'
  const members = useAsync(signal => getProjectMembers(project.id, { signal }), [project.id], { refreshOn: PROJECT_RESOURCES })
  const [username, setUsername] = useState('')
  const [role, setRole] = useState<SharedRole>('editor')
  const [error, setError] = useState('')
  const [done, setDone] = useState('')
  const [busy, setBusy] = useState<string | null>(null)

  async function run(key: string, action: () => Promise<unknown>, success: string) {
    setError('')
    setDone('')
    setBusy(key)
    try {
      await action()
      setDone(success)
      members.retry()
      return true
    } catch (err) {
      setError(message(err))
      return false
    } finally {
      setBusy(null)
    }
  }

  async function add(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    const name = username.trim().toLowerCase()
    if (!name) {
      setDone('')
      setError('Escribe el username de la persona')
      return
    }
    if (await run('add', () => addProjectMember(project.id, name, role), `${name} ahora es ${role} del proyecto`)) setUsername('')
  }

  function change(member: string, next: SharedRole) {
    void run(`role:${member}`, () => updateProjectMember(project.id, member, next), `${member} ahora es ${next}`)
  }

  function remove(member: string) {
    if (!window.confirm(`¿Quitar a ${member} del proyecto? Dejará de verlo, y sus agentes de trabajar en él.`)) return
    void run(`remove:${member}`, () => removeProjectMember(project.id, member), `${member} ya no es miembro`)
  }

  return <section className={s.sections} aria-label="Miembros del proyecto">
    <div className={s.section}>
      <div className={app['section-toolbar']}><h2>Miembros {members.data && <span className={app.count}>{members.data.length}</span>}</h2></div>
      <p>{isOwner
        ? 'Comparte el proyecto por username. Un editor trabaja en él con sus agentes; un viewer solo lo consulta aquí.'
        : `Tu rol: ${project.role ?? '—'}. ${project.role ? ROLE_HELP[project.role] : ''}. Solo el owner administra los miembros.`}</p>
      {isOwner && <form className={s.inlineForm} onSubmit={add} noValidate aria-label="Agregar miembro">
        <label>Username<input className={s.input} value={username} onChange={e => setUsername(e.target.value)} maxLength={40} placeholder="ana" autoComplete="off" spellCheck={false} /></label>
        <label className={s.narrow}>Rol<select className={s.select} value={role} onChange={e => setRole(e.target.value as SharedRole)}>
          {SHARED_ROLES.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select></label>
        <button type="submit" className={s.primary} disabled={busy !== null}>{busy === 'add' ? 'Agregando…' : 'Agregar'}</button>
      </form>}
      {error && <p className={s.error} role="alert">{error}</p>}
      {done && <p className={s.ok} role="status">{done}</p>}
      <QueryState result={members} label="Cargando miembros…" />
      {members.data && <div className={s.tableWrap}><table className={s.table}>
        <caption className={app['sr-only']}>Miembros del proyecto</caption>
        <thead><tr><th scope="col">Username</th><th scope="col">Nombre</th><th scope="col">Rol</th>{isOwner && <th scope="col"><span className={app['sr-only']}>Acciones</span></th>}</tr></thead>
        <tbody>{members.data.map(member => <tr key={member.username}>
          <td>{member.username}{member.username === me.username && <span className={s.you}>(tú)</span>}</td>
          <td>{member.name}</td>
          <td>{isOwner && member.role !== 'owner'
            ? <select className={s.select} aria-label={`Rol de ${member.username}`} value={member.role} disabled={busy !== null} onChange={e => change(member.username, e.target.value as SharedRole)}>
              {SHARED_ROLES.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
            : <RoleBadge role={member.role} />}</td>
          {isOwner && <td><div className={s.rowActions}>{member.role !== 'owner' && <button type="button" className={s.danger} disabled={busy !== null} onClick={() => remove(member.username)} aria-label={`Quitar a ${member.username}`}>{busy === `remove:${member.username}` ? 'Quitando…' : 'Quitar'}</button>}</div></td>}
        </tr>)}</tbody>
      </table></div>}
    </div>
  </section>
}
