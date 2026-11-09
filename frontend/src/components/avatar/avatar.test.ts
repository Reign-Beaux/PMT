import { describe, expect, it } from 'vitest'
import { avatarTone, initials } from './avatar'

const TONES = [
  'neutral',
  'progress',
  'success',
  'warning',
  'danger',
  'discovery',
]

describe('initials', () => {
  it('nombre con guiones toma las dos primeras partes', () => {
    expect(initials('pmt-frontend-devfront')).toBe('PF')
  })

  it('nombre con espacios toma las dos primeras partes', () => {
    expect(initials('Ana María López')).toBe('AM')
  })

  it('nombre con guion bajo', () => {
    expect(initials('élodie_rué')).toBe('ÉR')
  })

  it('una sola parte da una letra', () => {
    expect(initials('admin')).toBe('A')
  })

  it('vacío, solo espacios o solo separadores dan cadena vacía', () => {
    expect(initials('')).toBe('')
    expect(initials('  ')).toBe('')
    expect(initials('--_')).toBe('')
  })

  it('los caracteres fuera de BMP no se parten', () => {
    expect(initials('😀x')).toBe('😀')
    expect(initials('😀x y')).toBe('😀Y')
  })
})

describe('avatarTone', () => {
  it('es estable: dos llamadas con el mismo nombre dan el mismo tono', () => {
    expect(avatarTone('pmt-frontend-devfront')).toBe(
      avatarTone('pmt-frontend-devfront'),
    )
    expect(avatarTone('Ana María López')).toBe(avatarTone('Ana María López'))
  })

  it('siempre devuelve un tono válido, incluso con cadena vacía', () => {
    const nombres = [
      'pmt-frontend-devfront',
      'pmt-frontend-devback',
      'Ana María López',
      'admin',
      '',
      'élodie_rué',
      'qa',
      'tl',
      'pp',
      'pc',
    ]
    for (const nombre of nombres) {
      expect(TONES).toContain(avatarTone(nombre))
    }
  })
})
