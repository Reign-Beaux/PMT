import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Badge } from './Badge'
import badgeStyles from './Badge.module.scss'

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
