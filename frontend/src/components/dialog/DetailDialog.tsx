import { useEffect, useId, useRef } from 'react'
import type { ReactNode } from 'react'
import styles from './DetailDialog.module.scss'

export interface DetailDialogProps {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
}

export function DetailDialog({ open, title, onClose, children }: DetailDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const closeButtonRef = useRef<HTMLButtonElement>(null)
  const triggerRef = useRef<HTMLElement | null>(null)
  const titleId = useId()

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return

    if (open) {
      triggerRef.current =
        document.activeElement instanceof HTMLElement
          ? document.activeElement
          : null
      if (!dialog.open) dialog.showModal()
      closeButtonRef.current?.focus()
    } else if (dialog.open) {
      dialog.close()
      if (triggerRef.current?.isConnected) triggerRef.current.focus()
    }
  }, [open])

  useEffect(() => {
    const dialog = dialogRef.current
    return () => {
      if (dialog?.open) {
        dialog.close()
        if (triggerRef.current?.isConnected) triggerRef.current.focus()
      }
    }
  }, [])

  return (
    <dialog
      ref={dialogRef}
      className={styles.dialog}
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault()
        onClose()
      }}
    >
      <div className={styles.panel}>
        <h2 id={titleId} className={styles.title}>
          {title}
        </h2>
        <div className={styles.content}>{children}</div>
        <button
          ref={closeButtonRef}
          type="button"
          className={styles.closeButton}
          onClick={onClose}
        >
          Cerrar
        </button>
      </div>
    </dialog>
  )
}
