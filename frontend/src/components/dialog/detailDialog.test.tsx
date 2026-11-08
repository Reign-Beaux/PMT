import { beforeAll, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { DetailDialog } from './DetailDialog'

beforeAll(() => {
  const proto = HTMLDialogElement.prototype
  if (typeof proto.showModal !== 'function') {
    Object.defineProperty(proto, 'showModal', {
      value() {
        this.setAttribute('open', '')
      },
      configurable: true,
    })
  }
  if (typeof proto.close !== 'function') {
    Object.defineProperty(proto, 'close', {
      value() {
        this.removeAttribute('open')
      },
      configurable: true,
    })
  }
})

describe('DetailDialog', () => {
  const dialogEl = () => document.querySelector('dialog')!

  it('con open true muestra el diálogo con el título y el foco queda en Cerrar', () => {
    render(
      <DetailDialog open title="Detalle de la tarea" onClose={() => {}}>
        <p>Contenido</p>
      </DetailDialog>,
    )
    const dialog = screen.getByRole('dialog', { name: 'Detalle de la tarea' })
    expect(dialog.hasAttribute('open')).toBe(true)
    expect(screen.getByText('Contenido')).toBeDefined()
    const close = screen.getByRole('button', { name: 'Cerrar' })
    expect(document.activeElement).toBe(close)
    expect(dialog.contains(document.activeElement)).toBe(true)
  })

  it('clic en Cerrar llama onClose una vez', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(
      <DetailDialog open title="T" onClose={onClose}>
        <p>x</p>
      </DetailDialog>,
    )
    await user.click(screen.getByRole('button', { name: 'Cerrar' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('Escape llama onClose', () => {
    const onClose = vi.fn()
    render(
      <DetailDialog open title="T" onClose={onClose}>
        <p>x</p>
      </DetailDialog>,
    )
    fireEvent(
      screen.getByRole('dialog'),
      new Event('cancel', { bubbles: true, cancelable: true }),
    )
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('al pasar open de true a false el foco vuelve al botón que lo abrió', async () => {
    function Harness() {
      const [open, setOpen] = useState(false)
      return (
        <>
          <button type="button" onClick={() => setOpen(true)}>
            Abrir detalle
          </button>
          <DetailDialog open={open} title="T" onClose={() => setOpen(false)}>
            <p>x</p>
          </DetailDialog>
        </>
      )
    }
    const user = userEvent.setup()
    render(<Harness />)
    const trigger = screen.getByRole('button', { name: 'Abrir detalle' })
    await user.click(trigger)
    expect(document.activeElement).toBe(
      screen.getByRole('button', { name: 'Cerrar' }),
    )
    await user.click(screen.getByRole('button', { name: 'Cerrar' }))
    expect(dialogEl().hasAttribute('open')).toBe(false)
    expect(document.activeElement).toBe(trigger)
  })

  it('si el activador ya no existe no lanza error', async () => {
    function Harness() {
      const [open, setOpen] = useState(false)
      return (
        <>
          {!open ? (
            <button type="button" onClick={() => setOpen(true)}>
              Activador
            </button>
          ) : null}
          <DetailDialog open={open} title="T" onClose={() => setOpen(false)}>
            <p>x</p>
          </DetailDialog>
        </>
      )
    }
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByRole('button', { name: 'Activador' }))
    expect(screen.getByRole('dialog')).toBeDefined()
    await user.click(screen.getByRole('button', { name: 'Cerrar' }))
    expect(dialogEl().hasAttribute('open')).toBe(false)
  })

  it('abierto al montar sin activador previo no falla', () => {
    render(
      <DetailDialog open title="T" onClose={() => {}}>
        <p>x</p>
      </DetailDialog>,
    )
    expect(screen.getByRole('dialog').hasAttribute('open')).toBe(true)
  })

  it('al desmontar abierto cierra el diálogo', () => {
    const { unmount } = render(
      <DetailDialog open title="T" onClose={() => {}}>
        <p>x</p>
      </DetailDialog>,
    )
    const dialog = screen.getByRole('dialog')
    unmount()
    expect(dialog.hasAttribute('open')).toBe(false)
  })

  it('muestra el title como texto literal', () => {
    render(
      <DetailDialog open title="<b>x</b>" onClose={() => {}}>
        <p>x</p>
      </DetailDialog>,
    )
    expect(screen.getByText('<b>x</b>')).toBeDefined()
    expect(document.querySelector('b')).toBeNull()
  })

  it('cerrado no abre el diálogo', () => {
    render(
      <DetailDialog open={false} title="T" onClose={() => {}}>
        <p>x</p>
      </DetailDialog>,
    )
    expect(dialogEl().hasAttribute('open')).toBe(false)
  })

  it('cerrar con Escape controlado por el padre cierra el diálogo', async () => {
    function Harness() {
      const [open, setOpen] = useState(true)
      return (
        <DetailDialog open={open} title="T" onClose={() => setOpen(false)}>
          <p>x</p>
        </DetailDialog>
      )
    }
    render(<Harness />)
    fireEvent(
      screen.getByRole('dialog'),
      new Event('cancel', { bubbles: true, cancelable: true }),
    )
    await act(async () => {})
    expect(dialogEl().hasAttribute('open')).toBe(false)
  })
})
