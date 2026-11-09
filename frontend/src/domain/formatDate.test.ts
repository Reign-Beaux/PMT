import { describe, expect, it } from 'vitest'
import { formatDate, formatShortDate } from './formatDate'

/** Cadena ISO con el offset local de la máquina: representa la fecha local dada. */
function localIso(year: number, month: number, day: number, hour = 12): string {
  const offset = -new Date(year, month, day, hour).getTimezoneOffset()
  const sign = offset >= 0 ? '+' : '-'
  const abs = Math.abs(offset)
  const off = `${String(Math.floor(abs / 60)).padStart(2, '0')}:${String(abs % 60).padStart(2, '0')}`
  return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}T${String(hour).padStart(2, '0')}:00:00${sign}${off}`
}

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

describe('formatShortDate', () => {
  it('formatea una fecha válida como dd/mm/aaaa con relleno de ceros', () => {
    expect(formatShortDate(localIso(2026, 0, 5))).toBe('05/01/2026')
    expect(formatShortDate(localIso(2026, 10, 23))).toBe('23/11/2026')
  })

  it('cerca de medianoche usa la hora local, no la UTC', () => {
    // 23:00 local: recortar en UTC cambiaría el día en zonas al este de UTC.
    expect(formatShortDate(localIso(2026, 6, 9, 23))).toBe('09/07/2026')
    expect(formatShortDate(localIso(2026, 6, 9, 0))).toBe('09/07/2026')
  })

  it('con una fecha inválida devuelve el texto original', () => {
    expect(formatShortDate('no-es-una-fecha')).toBe('no-es-una-fecha')
    expect(formatShortDate('')).toBe('')
  })
})
