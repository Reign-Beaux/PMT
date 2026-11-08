import { describe, expect, it } from 'vitest'
import { formatDate } from './formatDate'

describe('formatDate', () => {
  it('formatea una fecha válida en español', () => {
    const result = formatDate('2026-11-01T09:30:00-06:00')

    expect(result).toContain('2026')
    expect(result).toContain('nov')
    expect(result).not.toBe('2026-11-01T09:30:00-06:00')
    expect(result).toBe(
      new Intl.DateTimeFormat('es', {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(new Date('2026-11-01T09:30:00-06:00')),
    )
  })

  it('con una fecha inválida devuelve el texto original', () => {
    expect(formatDate('no-es-una-fecha')).toBe('no-es-una-fecha')
    expect(formatDate('')).toBe('')
  })
})
