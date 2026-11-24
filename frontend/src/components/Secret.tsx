import { useState } from 'react'
import s from '../views/Account.module.scss'

export function Secret({ value, label }: { value: string; label: string }) {
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

