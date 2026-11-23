import { useState } from 'react'
import { Link, useSearchParams } from 'react-router'
import { authorizeCliLogin, cancelCliLogin, getCliLogin, type CliLogin } from '../api/cli'
import { HttpError } from '../api/http'
import { ErrorState, Loading } from '../components/feedback'
import { useAsync } from '../hooks/useAsync'
import { AuthCard } from '../shell/AuthCard'
import s from '../shell/Auth.module.scss'

/**
 * Autoriza el inicio de sesión del CLI pmt. Llega con ?codigo= desde la URL que
 * abre `pmt login`; sin sesión pasa por /login y vuelve aquí.
 */
export function CliLoginView() {
  const [params] = useSearchParams()
  const code = (params.get('codigo') ?? '').trim()
  const login = useAsync(signal => getCliLogin(code, signal), [code])
  const [decided, setDecided] = useState<CliLogin>()
  const [error, setError] = useState('')
  const [sending, setSending] = useState<'authorize' | 'cancel'>()

  async function decide(action: 'authorize' | 'cancel') {
    if (sending) return
    setError('')
    setSending(action)
    try {
      setDecided(await (action === 'authorize' ? authorizeCliLogin(code) : cancelCliLogin(code)))
    } catch (err) {
      if (err instanceof HttpError && err.status === 409) login.retry()
      setError(err instanceof Error ? err.message : 'Ocurrió un error inesperado')
    } finally {
      setSending(undefined)
    }
  }

  if (!code || (login.status === 'error' && login.error?.status === 404)) {
    return <AuthCard>
      <h1>Código no encontrado</h1>
      <p>Revisa la URL que mostró <code>pmt login</code> o vuelve a correrlo.</p>
      <Link className={s.link} to="/projects">Ir a PMT</Link>
    </AuthCard>
  }
  if (login.status === 'loading' && !login.data) return <AuthCard><Loading label="Cargando código…" /></AuthCard>
  if (login.status === 'error' || login.data === undefined) {
    return <AuthCard><ErrorState message={login.error?.message ?? 'No se pudo cargar el código'} onRetry={login.retry} /></AuthCard>
  }

  const current = decided ?? login.data
  const codeBox = <div className={s.cliCode} aria-label="Código">{current.code}</div>

  if (current.status === 'authorized') {
    return <AuthCard>
      <h1>Listo</h1>
      {codeBox}
      <p className={s.cliText}>Autorizaste a <strong>{current.machine}</strong>. Vuelve a la terminal: <code>pmt</code> recibirá el token «{current.token_name}». Puedes revocarlo en <Link to="/cuenta">Cuenta</Link>.</p>
    </AuthCard>
  }
  if (current.status === 'cancelled') {
    return <AuthCard>
      <h1>Inicio de sesión cancelado</h1>
      {codeBox}
      <p className={s.cliText}>No se creó ningún token. Si fue un error, vuelve a correr <code>pmt login</code>.</p>
    </AuthCard>
  }
  if (current.status === 'expired') {
    return <AuthCard>
      <h1>El código caducó</h1>
      {codeBox}
      <p className={s.cliText}>Ya se usó o pasaron más de 10 minutos. Vuelve a correr <code>pmt login</code>.</p>
    </AuthCard>
  }

  return <AuthCard>
    <h1>Autorizar el CLI pmt</h1>
    <p>Comprueba que la terminal muestre este mismo código.</p>
    {codeBox}
    <dl className={s.cliFacts}>
      <dt>Máquina</dt><dd>{current.machine}</dd>
      <dt>Token que se crea</dt><dd>{current.token_name}</dd>
    </dl>
    <p className={s.cliText}>Si no corriste <code>pmt login</code> tú, cancela: quien lo corrió podría actuar con tu cuenta.</p>
    {error && <div className={s.error} role="alert">{error}</div>}
    <div className={s.cliActions}>
      <button type="button" className={s.secondary} disabled={!!sending} onClick={() => decide('cancel')}>{sending === 'cancel' ? 'Cancelando…' : 'Cancelar'}</button>
      <button type="button" className={s.primary} disabled={!!sending} onClick={() => decide('authorize')}>{sending === 'authorize' ? 'Autorizando…' : 'Autorizar'}</button>
    </div>
  </AuthCard>
}
