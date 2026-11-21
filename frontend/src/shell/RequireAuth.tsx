import { useEffect } from 'react'
import { Navigate, Outlet, useLocation, useNavigate } from 'react-router'
import { getMe } from '../api/auth'
import { UNAUTHORIZED_EVENT } from '../api/http'
import { ErrorState, Loading } from '../components/feedback'
import { useAsync } from '../hooks/useAsync'
import { EventsProvider } from '../realtime/EventsProvider'
import { AuthContext } from './authContext'
import s from './Auth.module.scss'

/**
 * Exige sesión: sin ella, o si la API responde 401 después, lleva a /login.
 * El WebSocket de eventos solo se abre con sesión y usa la cookie.
 */
export function RequireAuth() {
  const me = useAsync(getMe, [])
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    const onUnauthorized = () => navigate('/login', { replace: true })
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized)
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized)
  }, [navigate])

  if (me.status === 'loading') {
    return <div className={s.page}><Loading label="Comprobando sesión…" /></div>
  }
  if (me.status === 'error' || me.data === undefined) {
    if (me.error?.status === 401) {
      return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
    }
    return <div className={s.page}><ErrorState message={me.error?.message ?? 'No se pudo comprobar la sesión'} onRetry={me.retry} /></div>
  }
  return (
    <AuthContext.Provider value={me.data}>
      <EventsProvider><Outlet /></EventsProvider>
    </AuthContext.Provider>
  )
}
