import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { acceptInvitation, getInvitation } from '../api/auth'
import { HttpError } from '../api/http'
import { ErrorState, Loading } from '../components/feedback'
import { USERNAME_PATTERN } from '../domain/user'
import { useAsync } from '../hooks/useAsync'
import { AuthCard } from '../shell/AuthCard'
import s from '../shell/Auth.module.scss'

type Field = 'name' | 'username' | 'password' | 'confirmation' | 'form'

export function InvitationView() {
  const { token = '' } = useParams()
  const invitation = useAsync(signal => getInvitation(token, signal), [token])
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [errors, setErrors] = useState<Partial<Record<Field, string>>>({})
  const [invalid, setInvalid] = useState(false)
  const [sending, setSending] = useState(false)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (sending) return
    const next: Partial<Record<Field, string>> = {}
    if (!name.trim()) next.name = 'Escribe tu nombre'
    if (!USERNAME_PATTERN.test(username)) next.username = 'Usa de 1 a 40 minúsculas, números o guiones, sin guion al inicio ni al final'
    if ([...password].length < 8) next.password = 'La contraseña debe tener al menos 8 caracteres'
    if (confirmation !== password) next.confirmation = 'Las contraseñas no coinciden'
    setErrors(next)
    if (Object.keys(next).length > 0) return
    setSending(true)
    try {
      await acceptInvitation(token, { name: name.trim(), username, password })
      navigate('/projects', { replace: true })
    } catch (err) {
      setSending(false)
      if (err instanceof HttpError && err.status === 404) setInvalid(true)
      else if (err instanceof HttpError && err.status === 409) setErrors({ username: err.message })
      else if (err instanceof HttpError && err.message === 'la invitación ya no es válida') setInvalid(true)
      else setErrors({ form: err instanceof Error ? err.message : 'Ocurrió un error inesperado' })
    }
  }

  const bad = invalid
    || (invitation.status === 'error' && invitation.error?.status === 404)
    || (invitation.status === 'success' && invitation.data?.valid === false)

  if (bad) {
    return <AuthCard>
      <h1>Invitación no válida</h1>
      <p>La invitación venció, ya se usó o no existe. Pide otra al dueño de la instancia.</p>
      <Link className={s.link} to="/login">Iniciar sesión</Link>
    </AuthCard>
  }
  if (invitation.status === 'loading') return <AuthCard><Loading label="Cargando invitación…" /></AuthCard>
  if (invitation.status === 'error' || invitation.data === undefined) {
    return <AuthCard><ErrorState message={invitation.error?.message ?? 'No se pudo cargar la invitación'} onRetry={invitation.retry} /></AuthCard>
  }

  const described = (field: Field) => errors[field] ? `invite-${field}-error` : undefined
  return <AuthCard>
    <h1>Acepta tu invitación</h1>
    <p>Crea tu cuenta para {invitation.data.email}.</p>
    <form noValidate onSubmit={submit} className={s.form}>
      <label htmlFor="invite-email">Correo</label>
      <input id="invite-email" className={s.input} value={invitation.data.email} readOnly />
      <label htmlFor="invite-name">Nombre</label>
      <input id="invite-name" className={s.input} autoComplete="name" value={name} onChange={e => setName(e.target.value)} aria-invalid={!!errors.name} aria-describedby={described('name')} />
      {errors.name && <p id="invite-name-error" className={s.fieldError} role="alert">{errors.name}</p>}
      <label htmlFor="invite-username">Username</label>
      <input id="invite-username" className={s.input} autoComplete="username" autoCapitalize="none" spellCheck={false} value={username} onChange={e => setUsername(e.target.value)} aria-invalid={!!errors.username} aria-describedby={described('username') ?? 'invite-username-hint'} />
      <span id="invite-username-hint" className={s.hint}>Tu nombre en historiales y comentarios. No se puede cambiar.</span>
      {errors.username && <p id="invite-username-error" className={s.fieldError} role="alert">{errors.username}</p>}
      <label htmlFor="invite-password">Contraseña</label>
      <input id="invite-password" className={s.input} type="password" autoComplete="new-password" value={password} onChange={e => setPassword(e.target.value)} aria-invalid={!!errors.password} aria-describedby={described('password')} />
      {errors.password && <p id="invite-password-error" className={s.fieldError} role="alert">{errors.password}</p>}
      <label htmlFor="invite-confirmation">Confirmar contraseña</label>
      <input id="invite-confirmation" className={s.input} type="password" autoComplete="new-password" value={confirmation} onChange={e => setConfirmation(e.target.value)} aria-invalid={!!errors.confirmation} aria-describedby={described('confirmation')} />
      {errors.confirmation && <p id="invite-confirmation-error" className={s.fieldError} role="alert">{errors.confirmation}</p>}
      {errors.form && <div className={s.error} role="alert">{errors.form}</div>}
      <button type="submit" disabled={sending}>{sending ? 'Creando…' : 'Crear cuenta'}</button>
    </form>
  </AuthCard>
}
