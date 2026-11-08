import { describe, expect, it } from 'vitest'
import { parseSelection, toSearch, type DetailSelection } from './selection'

describe('parseSelection', () => {
  it('sin parámetros devuelve null', () => {
    expect(parseSelection(new URLSearchParams(''))).toBeNull()
  })

  it('?task= vacío devuelve null', () => {
    expect(parseSelection(new URLSearchParams('task='))).toBeNull()
  })

  it('?bug= vacío devuelve null', () => {
    expect(parseSelection(new URLSearchParams('bug='))).toBeNull()
  })

  it('task y bug a la vez devuelve null por ambiguo', () => {
    expect(parseSelection(new URLSearchParams('task=a&bug=b'))).toBeNull()
  })

  it('?task=abc devuelve selección de tarea', () => {
    expect(parseSelection(new URLSearchParams('task=abc'))).toEqual({
      kind: 'task',
      id: 'abc',
    })
  })

  it('?bug=xyz devuelve selección de bug', () => {
    expect(parseSelection(new URLSearchParams('bug=xyz'))).toEqual({
      kind: 'bug',
      id: 'xyz',
    })
  })

  it('id con espacios, acentos y / hace ida y vuelta igual', () => {
    const sel: DetailSelection = { kind: 'task', id: 'tarea 1/ñandú' }
    const search = toSearch(sel)
    expect(parseSelection(new URLSearchParams(search))).toEqual(sel)
  })
})

describe('toSearch', () => {
  it('null sin parámetros previos devuelve cadena vacía', () => {
    expect(toSearch(null)).toBe('')
  })

  it('serializa la selección de tarea', () => {
    expect(toSearch({ kind: 'task', id: 'abc' })).toBe('task=abc')
  })

  it('conserva otros parámetros al seleccionar y sustituye la selección previa', () => {
    const current = new URLSearchParams('keep=1&task=old')
    expect(toSearch({ kind: 'bug', id: 'b1' }, current)).toBe('keep=1&bug=b1')
  })

  it('conserva otros parámetros al limpiar', () => {
    const current = new URLSearchParams('keep=1&task=t1')
    expect(toSearch(null, current)).toBe('keep=1')
  })
})
