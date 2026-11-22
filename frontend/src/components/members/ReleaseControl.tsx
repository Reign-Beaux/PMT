import { useId, useState, type FormEvent } from 'react'
import s from './Members.module.scss'

function message(err: unknown): string {
  return err instanceof Error ? err.message : 'Ocurrió un error inesperado'
}

/**
 * Botón «Liberar» de una etapa o un bug tomados, con su motivo obligatorio.
 * Solo lo monta quien es owner del proyecto; los errores del backend se
 * muestran con su mensaje.
 */
export function ReleaseControl({ kind, holder, onRelease }: {
  kind: 'stage' | 'bug'
  holder: string
  onRelease: (reason: string) => Promise<unknown>
}) {
  const [open, setOpen] = useState(false)
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const reasonId = useId()
  const what = kind === 'stage' ? 'la etapa' : 'el bug'

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setError('')
    if (!reason.trim()) {
      setError('Escribe el motivo: queda en el historial de status')
      return
    }
    setBusy(true)
    try {
      await onRelease(reason.trim())
      setOpen(false)
      setReason('')
    } catch (err) {
      setError(message(err))
    } finally {
      setBusy(false)
    }
  }

  if (!open) {
    return <button type="button" className={s.release} onClick={() => { setOpen(true); setError('') }}>Liberar</button>
  }
  return <form className={s.releaseForm} onSubmit={submit} noValidate aria-label={`Liberar ${what}`}>
    <p>{kind === 'stage' ? 'La etapa' : 'El bug'} queda sin tomar y {holder} deja de tenerl{kind === 'stage' ? 'a' : 'o'}. El status no cambia.</p>
    <label htmlFor={reasonId}>Motivo</label>
    <textarea id={reasonId} className={s.textarea} value={reason} onChange={e => setReason(e.target.value)} rows={3} maxLength={2000} autoFocus />
    {error && <p className={s.error} role="alert">{error}</p>}
    <div className={s.actions}>
      <button type="submit" className={s.primary} disabled={busy}>{busy ? 'Liberando…' : 'Confirmar liberación'}</button>
      <button type="button" className={s.secondary} onClick={() => { setOpen(false); setError('') }} disabled={busy}>Cancelar</button>
    </div>
  </form>
}
