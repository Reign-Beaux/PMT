import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConnectionStatus, Empty, ErrorState, Loading } from './index'

describe('Loading', () => {
  it('sin label muestra "Cargando…" con role status', () => {
    render(<Loading />)
    const status = screen.getByRole('status')
    expect(status.textContent).toBe('Cargando…')
  })

  it('con label muestra el label dado', () => {
    render(<Loading label="Cargando etapas…" />)
    expect(screen.getByRole('status').textContent).toBe('Cargando etapas…')
  })
})

describe('Loading — variantes con esqueleto (tarea 11)', () => {
  it.each(['list', 'detail'] as const)(
    'variante %s: el role status tiene exactamente el label y el esqueleto es aria-hidden sin texto',
    (variant) => {
      const { container } = render(
        <Loading label="Cargando contenido…" variant={variant} />,
      )
      expect(screen.getByRole('status').textContent).toBe(
        'Cargando contenido…',
      )
      const skeleton = container.querySelector('[aria-hidden="true"]')
      expect(skeleton).not.toBeNull()
      expect(skeleton?.textContent).toBe('')
      expect(skeleton?.querySelectorAll('div').length).toBeGreaterThan(0)
    },
  )

  it('la variante por defecto no renderiza esqueleto', () => {
    const { container } = render(<Loading label="Cargando contenido…" />)
    expect(screen.getByRole('status').textContent).toBe(
      'Cargando contenido…',
    )
    expect(container.querySelector('[aria-hidden="true"]')).toBeNull()
  })

  it.each(['list', 'detail'] as const)(
    'variante %s sin label muestra "Cargando…"',
    (variant) => {
      render(<Loading variant={variant} />)
      expect(screen.getByRole('status').textContent).toBe('Cargando…')
    },
  )
})

describe('Empty', () => {
  it('muestra el mensaje', () => {
    render(<Empty message="No hay tareas en esta columna" />)
    expect(screen.getByText('No hay tareas en esta columna')).toBeDefined()
  })

  it('muestra texto con HTML como literal', () => {
    render(<Empty message="<b>vacío</b>" />)
    expect(screen.getByText('<b>vacío</b>')).toBeDefined()
    expect(document.querySelector('b')).toBeNull()
  })
})

describe('ErrorState', () => {
  it('muestra el mensaje con role alert', () => {
    render(<ErrorState message="Falló la carga" onRetry={() => {}} />)
    expect(screen.getByRole('alert').textContent).toContain('Falló la carga')
  })

  it('clic en "Reintentar" llama onRetry una vez', async () => {
    const user = userEvent.setup()
    const onRetry = vi.fn()
    render(<ErrorState message="Falló la carga" onRetry={onRetry} />)
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })
})

describe('ConnectionStatus', () => {
  const horaLocal = (iso: string) =>
    new Intl.DateTimeFormat('es', {
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
    }).format(new Date(iso))

  it.each([
    ['connected', 'En vivo'],
    ['reconnecting', 'Reconectando…'],
    ['stale', 'Sin conexión'],
  ] as const)('muestra el texto de %s sin lastSyncedAt', (status, text) => {
    render(<ConnectionStatus status={status} />)
    expect(screen.getByRole('status').textContent).toBe(text)
  })

  it('stale con lastSyncedAt válido muestra la hora local', () => {
    const iso = '2026-11-07T10:30:00Z'
    render(<ConnectionStatus status="stale" lastSyncedAt={iso} />)
    expect(screen.getByRole('status').textContent).toBe(
      `Sin conexión, datos de las ${horaLocal(iso)}`,
    )
  })

  it('stale con lastSyncedAt inválido muestra solo "Sin conexión"', () => {
    render(<ConnectionStatus status="stale" lastSyncedAt="no-es-fecha" />)
    expect(screen.getByRole('status').textContent).toBe('Sin conexión')
  })

  it('al cambiar de estado conserva el mismo nodo y no mueve el foco', () => {
    const { rerender } = render(
      <div>
        <button type="button">Foco</button>
        <ConnectionStatus status="connected" />
      </div>,
    )
    const button = screen.getByRole('button')
    button.focus()
    const node = screen.getByRole('status')
    rerender(
      <div>
        <button type="button">Foco</button>
        <ConnectionStatus status="stale" lastSyncedAt="2026-11-07T10:30:00Z" />
      </div>,
    )
    expect(screen.getByRole('status')).toBe(node)
    expect(document.activeElement).toBe(button)
    expect(node.textContent).toContain('Sin conexión')
  })
})

describe('datos previos visibles junto a Loading', () => {
  it('una lista renderizada junto a Loading sigue visible', () => {
    render(
      <div>
        <ul>
          <li>Tarea 1</li>
          <li>Tarea 2</li>
        </ul>
        <Loading label="Actualizando…" />
      </div>,
    )
    expect(screen.getByText('Tarea 1')).toBeDefined()
    expect(screen.getByText('Tarea 2')).toBeDefined()
    expect(screen.getByRole('status').textContent).toBe('Actualizando…')
  })
})
