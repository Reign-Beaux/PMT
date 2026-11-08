import { describe, expect, it } from 'vitest'
import { parseChangeEvent } from './events'
import type { ChangeEventResource } from './events'

describe('eventos válidos', () => {
  it('acepta resync', () => {
    expect(parseChangeEvent('{"type":"resync"}')).toEqual({
      ok: true,
      event: { type: 'resync' },
    })
  })

  it.each<ChangeEventResource>([
    'project',
    'stage',
    'task',
    'bug',
    'comment',
    'status_change',
  ])('acepta invalidate con resource %s', (resource) => {
    expect(parseChangeEvent(JSON.stringify({ type: 'invalidate', resource }))).toEqual({
      ok: true,
      event: { type: 'invalidate', resource },
    })
  })

  it('devuelve el evento sin campos extra', () => {
    const result = parseChangeEvent(
      '{"type":"invalidate","resource":"task","actor":"pmt-frontend-devfront"}',
    )

    expect(result).toEqual({ ok: true, event: { type: 'invalidate', resource: 'task' } })
  })

  it('resync ignora campos extra', () => {
    expect(parseChangeEvent('{"type":"resync","extra":1}')).toEqual({
      ok: true,
      event: { type: 'resync' },
    })
  })
})

describe('errores controlados (nunca lanza)', () => {
  it.each([
    ['data que no es string', 42, /no es texto/],
    ['JSON inválido', '{no-json', /no es JSON válido/],
    ['null', 'null', /no es un objeto/],
    ['array', '[]', /no es un objeto/],
    ['número', '7', /no es un objeto/],
    ['type desconocido', '{"type":"reload"}', /desconocido/],
    ['sin type', '{}', /desconocido/],
    ['invalidate sin resource', '{"type":"invalidate"}', /sin resource/],
    [
      'invalidate con recurso no permitido',
      '{"type":"invalidate","resource":"user"}',
      /no permitido/,
    ],
  ])('%s devuelve ok:false con mensaje', (_caso, data, mensaje) => {
    let result!: ReturnType<typeof parseChangeEvent>
    expect(() => {
      result = parseChangeEvent(data)
    }).not.toThrow()
    expect(result).toMatchObject({ ok: false })
    expect(result.ok === false && result.error).toMatch(mensaje)
  })
})
