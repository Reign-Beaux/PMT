import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { EventsProvider } from '../realtime/EventsProvider'
import { createFakeConnection } from '../test/fakeConnection'
import { Shell } from './Shell'

interface RenderShellOptions {
  projectId?: string
  projectName?: string
  stageName?: string
}

function renderShell(initialEntry: string, props: RenderShellOptions = {}) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Shell {...props}>
        <p>contenido de prueba</p>
      </Shell>
    </MemoryRouter>,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Shell', () => {
  it('con projectId muestra el contexto y las pestañas con los href correctos', () => {
    renderShell('/projects/p1', { projectId: 'p1', projectName: 'Mi proyecto' })
    expect(screen.getByText('Mi proyecto')).toBeDefined()
    const etapas = screen.getByRole('link', { name: 'Etapas' })
    const bugs = screen.getByRole('link', { name: 'Bugs' })
    expect(etapas.getAttribute('href')).toBe('/projects/p1')
    expect(bugs.getAttribute('href')).toBe('/projects/p1/bugs')
  })

  it('muestra stageName en el contexto cuando existe', () => {
    renderShell('/projects/p1/stages/s2', {
      projectId: 'p1',
      projectName: 'Mi proyecto',
      stageName: 'Etapa dos',
    })
    expect(screen.getByText('Etapa dos')).toBeDefined()
  })

  it('sin projectName muestra el projectId', () => {
    renderShell('/projects/p1', { projectId: 'p1' })
    expect(screen.getByText('p1')).toBeDefined()
  })

  it('marca aria-current="page" en Etapas en /projects/p1', () => {
    renderShell('/projects/p1', { projectId: 'p1' })
    expect(screen.getByRole('link', { name: 'Etapas' }).getAttribute('aria-current')).toBe('page')
    expect(screen.getByRole('link', { name: 'Bugs' }).hasAttribute('aria-current')).toBe(false)
  })

  it('marca aria-current="page" en Etapas en /projects/p1/stages/s2', () => {
    renderShell('/projects/p1/stages/s2', { projectId: 'p1' })
    expect(screen.getByRole('link', { name: 'Etapas' }).getAttribute('aria-current')).toBe('page')
    expect(screen.getByRole('link', { name: 'Bugs' }).hasAttribute('aria-current')).toBe(false)
  })

  it('marca aria-current="page" en Bugs en /projects/p1/bugs', () => {
    renderShell('/projects/p1/bugs', { projectId: 'p1' })
    expect(screen.getByRole('link', { name: 'Bugs' }).getAttribute('aria-current')).toBe('page')
    expect(screen.getByRole('link', { name: 'Etapas' }).hasAttribute('aria-current')).toBe(false)
  })

  it('sin projectId no hay pestañas ni contexto', () => {
    renderShell('/', {})
    expect(screen.queryByRole('navigation')).toBeNull()
    expect(screen.queryByRole('link', { name: 'Etapas' })).toBeNull()
    expect(screen.queryByRole('link', { name: 'Bugs' })).toBeNull()
  })

  it('envuelve el contenido en <main>', () => {
    renderShell('/projects/p1', { projectId: 'p1' })
    const main = screen.getByRole('main')
    expect(main.textContent).toContain('contenido de prueba')
  })

  it('no llama a fetch', () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
    renderShell('/projects/p1', { projectId: 'p1', projectName: 'Mi proyecto' })
    expect(fetchSpy).not.toHaveBeenCalled()
  })
})

describe('Shell — indicador de conexión', () => {
  function renderShellWithConnection(initialEntry: string, props: RenderShellOptions = {}) {
    const connection = createFakeConnection()
    const view = render(
      <MemoryRouter initialEntries={[initialEntry]}>
        <EventsProvider connection={connection}>
          <Shell {...props}>
            <p>contenido de prueba</p>
          </Shell>
        </EventsProvider>
      </MemoryRouter>,
    )
    return { connection, ...view }
  }

  it('sin proveedor no renderiza el indicador', () => {
    renderShell('/', {})
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('muestra el estado de la conexión en el header', () => {
    renderShellWithConnection('/')
    const status = screen.getByRole('status')
    expect(status.textContent).toBe('Reconectando…')
    expect(status.closest('header')).not.toBeNull()
  })

  it('muestra el indicador también con proyecto', () => {
    renderShellWithConnection('/projects/p1', { projectId: 'p1' })
    expect(screen.getByRole('status').textContent).toBe('Reconectando…')
  })

  it('actualiza el indicador cuando la conexión cambia de estado', () => {
    const { connection } = renderShellWithConnection('/')
    expect(screen.getByRole('status').textContent).toBe('Reconectando…')
    act(() => {
      connection.setStatus({
        status: 'connected',
        lastSyncedAt: '2026-11-07T12:00:00.000Z',
      })
    })
    expect(screen.getByRole('status').textContent).toBe('En vivo')
    act(() => {
      connection.setStatus({
        status: 'stale',
        lastSyncedAt: '2026-11-07T12:00:00.000Z',
      })
    })
    expect(screen.getByRole('status').textContent).toContain(
      'Sin conexión, datos de las',
    )
  })
})
