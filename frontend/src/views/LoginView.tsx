import { useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { login } from '../api/auth'
import { HttpError } from '../api/http'
import { AuthCard } from '../shell/AuthCard'
import s from '../shell/Auth.module.scss'

export function LoginView() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (sending) return
    setError('')
    if (!email.trim() || !password) {
      setError('Escribe tu correo y tu contraseña')
      return
    }
    setSending(true)
    try {
      await login(email.trim(), password)
      navigate(from && from !== '/login' ? from : '/projects', { replace: true })
    } catch (err) {
      setSending(false)
      if (err instanceof HttpError && err.status === 401) {
        setPassword('')
        setError('Correo o contraseña incorrectos')
      } else if (err instanceof HttpError && err.status === 403) {
        setError('Tu usuario está desactivado. Pide al dueño de la instancia que lo reactive')
      } else {
        setError(err instanceof Error ? err.message : 'Ocurrió un error inesperado')
      }
    }
  }

  return <AuthCard>
    <h1>Entra a PMT</h1>
    <p>Inicia sesión para seguir tus proyectos.</p>
    <form noValidate onSubmit={submit} className={s.form}>
      <label htmlFor="login-email">Correo</label>
      <input id="login-email" className={s.input} type="email" autoComplete="email" value={email} readOnly={sending} onChange={e => setEmail(e.target.value)} />
      <label htmlFor="login-password">Contraseña</label>
      <input id="login-password" className={s.input} type="password" autoComplete="current-password" value={password} readOnly={sending} onChange={e => setPassword(e.target.value)} />
      {error && <div className={s.error} role="alert">{error}</div>}
      <button type="submit" disabled={sending}>{sending ? 'Entrando…' : 'Entrar'}</button>
    </form>
  </AuthCard>
}
