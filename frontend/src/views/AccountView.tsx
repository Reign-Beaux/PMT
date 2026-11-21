import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router'
import {
  changePassword,
  createInvitation,
  createToken,
  disableUser,
  enableUser,
  listInvitations,
  listTokens,
  listUsers,
  logout,
  revokeToken,
} from '../api/auth'
import { QueryState } from '../components/QueryState'
import { formatDate } from '../domain/formatDate'
import { invitationState, type NewApiToken, type NewInvitation } from '../domain/user'
import { useAsync } from '../hooks/useAsync'
import { useCurrentUser } from '../shell/authContext'
import { PageBreadcrumbs } from '../shell/PageBreadcrumbs'
import app from '../styles/App.module.scss'
import s from './Account.module.scss'

function message(err: unknown): string {
  return err instanceof Error ? err.message : 'Ocurrió un error inesperado'
}

function Secret({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false)
  async function copy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
    } catch {
      setCopied(false)
    }
  }
  return <div className={s.secret}>
    <code aria-label={label}>{value}</code>
    <button type="button" className={s.secondary} onClick={copy}>{copied ? 'Copiado' : 'Copiar'}</button>
  </div>
}

function Tokens() {
  const tokens = useAsync(listTokens, [])
  const [name, setName] = useState('')
  const [created, setCreated] = useState<NewApiToken | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setError('')
    if (!name.trim()) {
      setError('Escribe un nombre para el token, por ejemplo «laptop»')
      return
    }
    setBusy(true)
    try {
      setCreated(await createToken(name.trim()))
      setName('')
      tokens.retry()
    } catch (err) {
      setError(message(err))
    } finally {
      setBusy(false)
    }
  }

  async function revoke(id: string, tokenName: string) {
    if (!window.confirm(`¿Revocar el token «${tokenName}»? Los agentes que lo usan dejarán de entrar.`)) return
    setError('')
    try {
      await revokeToken(id)
      if (created?.id === id) setCreated(null)
      tokens.retry()
    } catch (err) {
      setError(message(err))
    }
  }

  return <section className={s.section} aria-labelledby="tokens-title">
    <h2 id="tokens-title">Tokens de API</h2>
    <p>Para tus agentes: mándalo como <code>Authorization: Bearer</code> al MCP. Actúan con tu username.</p>
    <form className={s.inlineForm} onSubmit={submit} noValidate>
      <label>Nombre<input className={s.input} value={name} onChange={e => setName(e.target.value)} maxLength={100} placeholder="laptop" /></label>
      <button type="submit" className={s.primary} disabled={busy}>{busy ? 'Creando…' : 'Crear token'}</button>
    </form>
    {created && <div className={s.reveal} role="status">
      <p>Copia el token «{created.name}» ahora: no se vuelve a mostrar.</p>
      <Secret value={created.token} label="Token nuevo" />
      <button type="button" className={`${s.secondary} ${s.revealDone}`} onClick={() => setCreated(null)}>Ya lo copié</button>
    </div>}
    {error && <p className={s.error} role="alert">{error}</p>}
    <QueryState result={tokens} label="Cargando tokens…" />
    {tokens.data && (tokens.data.length === 0
      ? <p className={s.empty}>No tienes tokens.</p>
      : <div className={s.tableWrap}><table className={s.table}>
        <thead><tr><th>Nombre</th><th>Creado</th><th>Último uso</th><th><span className={app['sr-only']}>Acciones</span></th></tr></thead>
        <tbody>{tokens.data.map(token => <tr key={token.id}>
          <td>{token.name}</td>
          <td className={s.muted}>{formatDate(token.created_at)}</td>
          <td className={s.muted}>{token.last_used_at ? formatDate(token.last_used_at) : 'Nunca'}</td>
          <td><button type="button" className={s.danger} onClick={() => revoke(token.id, token.name)} aria-label={`Revocar ${token.name}`}>Revocar</button></td>
        </tr>)}</tbody>
      </table></div>)}
  </section>
}

function Password() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)
  const [busy, setBusy] = useState(false)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setError('')
    setDone(false)
    if ([...next].length < 8) { setError('La contraseña nueva debe tener al menos 8 caracteres'); return }
    if (next !== confirmation) { setError('Las contraseñas no coinciden'); return }
    setBusy(true)
    try {
      await changePassword(current, next)
      setCurrent(''); setNext(''); setConfirmation('')
      setDone(true)
    } catch (err) {
      setError(message(err))
    } finally {
      setBusy(false)
    }
  }

  return <section className={s.section} aria-labelledby="password-title">
    <h2 id="password-title">Contraseña</h2>
    <p>Al cambiarla se cierran tus otras sesiones del navegador. Tus tokens siguen vigentes.</p>
    <form className={s.inlineForm} onSubmit={submit} noValidate>
      <label>Actual<input className={s.input} type="password" autoComplete="current-password" value={current} onChange={e => setCurrent(e.target.value)} /></label>
      <label>Nueva<input className={s.input} type="password" autoComplete="new-password" value={next} onChange={e => setNext(e.target.value)} /></label>
      <label>Confirmar<input className={s.input} type="password" autoComplete="new-password" value={confirmation} onChange={e => setConfirmation(e.target.value)} /></label>
      <button type="submit" className={s.primary} disabled={busy}>{busy ? 'Guardando…' : 'Cambiar contraseña'}</button>
    </form>
    {error && <p className={s.error} role="alert">{error}</p>}
    {done && <p className={s.ok} role="status">Contraseña cambiada.</p>}
  </section>
}

const STATE_LABEL = { pendiente: 'Pendiente', aceptada: 'Aceptada', vencida: 'Vencida' } as const

function Invitations() {
  const invitations = useAsync(listInvitations, [])
  const [email, setEmail] = useState('')
  const [created, setCreated] = useState<NewInvitation | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [now] = useState(() => new Date())

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setError('')
    if (!email.trim()) { setError('Escribe el correo de la persona'); return }
    setBusy(true)
    try {
      setCreated(await createInvitation(email.trim()))
      setEmail('')
      invitations.retry()
    } catch (err) {
      setError(message(err))
    } finally {
      setBusy(false)
    }
  }

  return <section className={s.section} aria-labelledby="invitations-title">
    <h2 id="invitations-title">Invitaciones</h2>
    <p>PMT no envía correos: copia el enlace y mándalo tú. Vence en 7 días.</p>
    <form className={s.inlineForm} onSubmit={submit} noValidate>
      <label>Correo<input className={s.input} type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="persona@ejemplo.com" /></label>
      <button type="submit" className={s.primary} disabled={busy}>{busy ? 'Invitando…' : 'Invitar'}</button>
    </form>
    {created && <div className={s.reveal} role="status">
      <p>Enlace para {created.email}. Cópialo ahora: no se vuelve a mostrar.</p>
      <Secret value={created.url} label="Enlace de invitación" />
    </div>}
    {error && <p className={s.error} role="alert">{error}</p>}
    <QueryState result={invitations} label="Cargando invitaciones…" />
    {invitations.data && (invitations.data.length === 0
      ? <p className={s.empty}>No hay invitaciones.</p>
      : <div className={s.tableWrap}><table className={s.table}>
        <thead><tr><th>Correo</th><th>Estado</th><th>Vence</th></tr></thead>
        <tbody>{invitations.data.map(item => <tr key={item.id}>
          <td>{item.email}</td>
          <td>{STATE_LABEL[invitationState(item, now)]}</td>
          <td className={s.muted}>{formatDate(item.expires_at)}</td>
        </tr>)}</tbody>
      </table></div>)}
  </section>
}

function Users() {
  const users = useAsync(listUsers, [])
  const [error, setError] = useState('')

  async function toggle(id: string, username: string, disabled: boolean) {
    if (!disabled && !window.confirm(`¿Desactivar a ${username}? Se cierran sus sesiones y se revocan sus tokens.`)) return
    setError('')
    try {
      await (disabled ? enableUser(id) : disableUser(id))
      users.retry()
    } catch (err) {
      setError(message(err))
    }
  }

  return <section className={s.section} aria-labelledby="users-title">
    <h2 id="users-title">Usuarios</h2>
    <p>Un usuario desactivado no entra ni con su sesión ni con sus tokens. Al reactivarlo debe crear tokens nuevos.</p>
    {error && <p className={s.error} role="alert">{error}</p>}
    <QueryState result={users} label="Cargando usuarios…" />
    {users.data && <div className={s.tableWrap}><table className={s.table}>
      <thead><tr><th>Username</th><th>Nombre</th><th>Correo</th><th>Estado</th><th><span className={app['sr-only']}>Acciones</span></th></tr></thead>
      <tbody>{users.data.map(user => <tr key={user.id}>
        <td>{user.username}</td>
        <td>{user.name}</td>
        <td className={s.muted}>{user.email}</td>
        <td>{user.is_instance_owner ? 'Dueño' : user.disabled ? 'Desactivado' : 'Activo'}</td>
        <td>{!user.is_instance_owner && <button type="button" className={user.disabled ? s.secondary : s.danger} onClick={() => toggle(user.id, user.username, user.disabled)}>{user.disabled ? 'Reactivar' : 'Desactivar'}</button>}</td>
      </tr>)}</tbody>
    </table></div>}
  </section>
}

export function AccountView() {
  const user = useCurrentUser()
  const navigate = useNavigate()
  const [error, setError] = useState('')

  async function signOut() {
    setError('')
    try {
      await logout()
      navigate('/login', { replace: true })
    } catch (err) {
      setError(message(err))
    }
  }

  return <><PageBreadcrumbs items={[{ label: 'Cuenta' }]} /><main id="contenido" className={app.main}>
    <header className={app['page-heading']}>
      <div><h1>{user.name}</h1><p className={app['heading-subtitle']}>{user.username} · {user.email}{user.is_instance_owner ? ' · dueño de la instancia' : ''}</p></div>
      <div><button type="button" className={s.secondary} onClick={signOut}>Cerrar sesión</button>{error && <p className={s.error} role="alert">{error}</p>}</div>
    </header>
    <div className={s.sections}>
      <Tokens />
      <Password />
      {user.is_instance_owner && <><Invitations /><Users /></>}
    </div>
  </main></>
}
