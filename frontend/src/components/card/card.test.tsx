import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Badge } from './Badge'
import badgeStyles from './Badge.module.scss'
import { Card } from './Card'

describe('Card', () => {
  it('con onOpen se activa con click', async () => {
    const user = userEvent.setup()
    const onOpen = vi.fn()
    render(<Card title="Tarea" id="t-1" onOpen={onOpen} />)
    await user.click(screen.getByRole('button', { name: /Tarea/ }))
    expect(onOpen).toHaveBeenCalledTimes(1)
  })

  it('con onOpen se activa con Enter', async () => {
    const user = userEvent.setup()
    const onOpen = vi.fn()
    render(<Card title="Tarea" id="t-1" onOpen={onOpen} />)
    screen.getByRole('button', { name: /Tarea/ }).focus()
    await user.keyboard('{Enter}')
    expect(onOpen).toHaveBeenCalledTimes(1)
  })

  it('con onOpen se activa con Espacio', async () => {
    const user = userEvent.setup()
    const onOpen = vi.fn()
    render(<Card title="Tarea" id="t-1" onOpen={onOpen} />)
    screen.getByRole('button', { name: /Tarea/ }).focus()
    await user.keyboard(' ')
    expect(onOpen).toHaveBeenCalledTimes(1)
  })

  it('sin onOpen no hay botón y no es interactiva', () => {
    const { container } = render(<Card title="Tarea" id="t-1" />)
    expect(screen.queryByRole('button')).toBeNull()
    expect(container.querySelector('article')).not.toBeNull()
  })

  it('no hay elementos interactivos anidados', () => {
    const { container } = render(
      <Card title="Tarea" id="t-1" onOpen={() => {}} />,
    )
    expect(
      container.querySelector('button button, button a, button input, button select, button textarea'),
    ).toBeNull()
    expect(container.querySelectorAll('button')).toHaveLength(1)
  })

  it('muestra title, id y subtitle como texto literal', () => {
    render(<Card title="<b>x</b>" id="t-42" subtitle="Etapa 1 · asignada a dev" />)
    expect(screen.getByText('<b>x</b>')).toBeDefined()
    expect(screen.getByText('t-42')).toBeDefined()
    expect(screen.getByText('Etapa 1 · asignada a dev')).toBeDefined()
    expect(document.querySelector('b')).toBeNull()
  })

  it('sin subtitle no deja hueco', () => {
    const { container } = render(<Card title="Tarea" id="t-1" />)
    expect(container.querySelectorAll('span')).toHaveLength(2)
  })

  it('meta se pinta dentro de la tarjeta (dentro del botón si hay onOpen)', () => {
    render(
      <Card
        title="Tarea"
        id="t-1"
        meta={<Badge label="En desarrollo" />}
        onOpen={() => {}}
      />,
    )
    const button = screen.getByRole('button', { name: /Tarea/ })
    expect(button.textContent).toContain('En desarrollo')
    expect(screen.getByText('En desarrollo')).toBeDefined()
  })
})

describe('Badge', () => {
  it('muestra el label', () => {
    render(<Badge label="En desarrollo" />)
    expect(screen.getByText('En desarrollo')).toBeDefined()
  })

  it.each(['neutral', 'info', 'success', 'warning', 'danger'] as const)(
    'aplica la clase del tone %s',
    (tone) => {
      const { container } = render(<Badge label="x" tone={tone} />)
      const badge = container.querySelector('span')!
      expect(badge.classList.contains(badgeStyles[tone])).toBe(true)
    },
  )

  it('por defecto usa neutral', () => {
    const { container } = render(<Badge label="x" />)
    const badge = container.querySelector('span')!
    expect(badge.classList.contains(badgeStyles.neutral)).toBe(true)
  })

  it('muestra el label como texto literal', () => {
    render(<Badge label="<script>x</script>" />)
    expect(screen.getByText('<script>x</script>')).toBeDefined()
    expect(document.querySelector('script[data-x]')).toBeNull()
  })
})
