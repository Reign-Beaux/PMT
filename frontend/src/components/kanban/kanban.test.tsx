import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { KanbanBoard } from './KanbanBoard'
import { KanbanColumn } from './KanbanColumn'

describe('KanbanColumn', () => {
  it('muestra label y count', () => {
    render(
      <KanbanColumn id="in_dev" label="En desarrollo" count={3}>
        <p>Tarea A</p>
      </KanbanColumn>,
    )
    expect(screen.getByText('En desarrollo')).toBeDefined()
    expect(screen.getByLabelText('3 elementos')).toBeDefined()
    expect(screen.getByLabelText('3 elementos').textContent).toBe('3')
  })

  it('con count 0 muestra "Sin elementos" y la columna está presente', () => {
    render(<KanbanColumn id="done" label="Done" count={0} />)
    const column = screen.getByRole('region', { hidden: true })
    expect(column).toBeDefined()
    expect(screen.getByText('Sin elementos')).toBeDefined()
  })

  it('con count > 0 muestra los children', () => {
    render(
      <KanbanColumn id="ready" label="Ready" count={2}>
        <p>Tarea 1</p>
        <p>Tarea 2</p>
      </KanbanColumn>,
    )
    expect(screen.getByText('Tarea 1')).toBeDefined()
    expect(screen.getByText('Tarea 2')).toBeDefined()
    expect(screen.queryByText('Sin elementos')).toBeNull()
  })

  it('muestra el label como texto literal', () => {
    render(<KanbanColumn id="x" label="<b>x</b>" count={1} />)
    expect(screen.getByText('<b>x</b>')).toBeDefined()
    expect(document.querySelector('b')).toBeNull()
  })

  it('el id con acentos no rompe el aria-labelledby', () => {
    render(<KanbanColumn id="diseño" label="Diseño" count={1} />)
    const column = screen.getByRole('region', { name: /Diseño/ })
    expect(column.getAttribute('aria-labelledby')).toBe('kanban-col-diseño')
    expect(screen.getByLabelText('1 elementos')).toBeDefined()
  })
})

describe('KanbanBoard', () => {
  it('tiene role region con aria-label y tabIndex 0', () => {
    render(
      <KanbanBoard label="Tablero del proyecto">
        <KanbanColumn id="a" label="A" count={0} />
      </KanbanBoard>,
    )
    const board = screen.getByRole('region', { name: 'Tablero del proyecto' })
    expect(board.getAttribute('tabindex')).toBe('0')
  })

  it('renderiza todas las columnas recibidas, incluidas las vacías', () => {
    render(
      <KanbanBoard label="Tablero">
        <KanbanColumn id="backlog" label="Backlog" count={2}>
          <p>T1</p>
          <p>T2</p>
        </KanbanColumn>
        <KanbanColumn id="ready" label="Ready" count={0} />
        <KanbanColumn id="done" label="Done" count={0} />
      </KanbanBoard>,
    )
    const board = screen.getByRole('region', { name: 'Tablero' })
    const columns = within(board).getAllByRole('region', { hidden: true })
    expect(columns).toHaveLength(3)
    expect(within(board).getAllByText('Sin elementos')).toHaveLength(2)
  })
})
