import { useEffect, useId, useRef } from 'react'
import type { ReactNode } from 'react'
import { Icon } from '../Icon'
import s from '../../styles/App.module.scss'

export function DetailDialog({ open, title, onClose, children }: { open: boolean; title: string; onClose: () => void; children: ReactNode }) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const titleId = useId()

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog || !open) return
    const trigger = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const root = document.documentElement
    const previousOverflow = root.style.overflow
    const previousScrollBehavior = root.style.scrollBehavior
    const x = window.scrollX
    const y = window.scrollY
    root.style.scrollBehavior = 'auto'
    root.style.overflow = 'hidden'
    if (!dialog.open) dialog.showModal()
    // showModal enfoca su primer control; algunos navegadores desplazan el
    // documento al hacerlo, incluso si el diálogo tiene position: fixed.
    window.scrollTo({ left: x, top: y, behavior: 'instant' })
    closeRef.current?.focus({ preventScroll: true })
    return () => {
      if (dialog.open) dialog.close()
      root.style.overflow = previousOverflow
      // Tanto el foco como la URL del detalle deben conservar el scroll de la lista.
      root.style.scrollBehavior = 'auto'
      window.scrollTo({ left: x, top: y, behavior: 'instant' })
      if (trigger?.isConnected) trigger.focus({ preventScroll: true })
      root.style.scrollBehavior = previousScrollBehavior
    }
  }, [open])

  return <dialog ref={dialogRef} className={`${s.drawer} ${s.detailDialog}`} aria-labelledby={titleId}
    onCancel={event => { event.preventDefault(); onClose() }}
    onClick={event => {
      // El backdrop pertenece al dialog, pero no su rectángulo visible.
      if (event.target !== event.currentTarget) return
      const bounds = event.currentTarget.getBoundingClientRect()
      if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) onClose()
    }}>
    <header className={s['drawer-top']}><span><Icon name="file" />Detalle</span><button ref={closeRef} type="button" className={s.iconButton} onClick={onClose} aria-label="Cerrar detalle"><Icon name="close" /></button></header>
    <div className={s['drawer-scroll']} data-testid="detail-scroll"><h2 id={titleId}>{title}</h2>{children}</div>
    <footer className={s['drawer-footer']}><button type="button" className={s.button} onClick={onClose}>Cerrar detalle <Icon name="close" /></button></footer>
  </dialog>
}
