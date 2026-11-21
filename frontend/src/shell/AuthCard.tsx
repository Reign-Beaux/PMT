import type { ReactNode } from 'react'
import s from './Auth.module.scss'

/** Marco de las pantallas sin sesión. */
export function AuthCard({ children }: { children: ReactNode }) {
  return <main id="contenido" className={s.page}>
    <div className={s.card}>
      <div className={s.brand}><span className={s.brandMark} aria-hidden="true"><i /><i /><i /></span>PMT</div>
      {children}
    </div>
  </main>
}
