import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StageProgress } from './StageProgress'
import { stageProgressPercent } from './stageProgress'

describe('stageProgressPercent', () => {
  it('0 tareas → 0', () => {
    expect(stageProgressPercent({ done: 0, total: 0, cancelled: 0 })).toBe(0)
  })

  it('todas canceladas → 0', () => {
    expect(stageProgressPercent({ done: 0, total: 3, cancelled: 3 })).toBe(0)
  })

  it('1/2 → 50', () => {
    expect(stageProgressPercent({ done: 1, total: 2, cancelled: 0 })).toBe(50)
  })

  it('2/3 → 67', () => {
    expect(stageProgressPercent({ done: 2, total: 3, cancelled: 0 })).toBe(67)
  })

  it('3/3 → 100', () => {
    expect(stageProgressPercent({ done: 3, total: 3, cancelled: 0 })).toBe(100)
  })

  it('done mayor que el denominador → 100', () => {
    expect(stageProgressPercent({ done: 5, total: 2, cancelled: 0 })).toBe(100)
  })

  it('done negativo → 0', () => {
    expect(stageProgressPercent({ done: -3, total: 2, cancelled: 0 })).toBe(0)
  })

  it('las canceladas no cuentan en el denominador', () => {
    expect(stageProgressPercent({ done: 1, total: 4, cancelled: 2 })).toBe(50)
  })
})

describe('StageProgress', () => {
  it('expone role progressbar con aria-valuenow y aria-valuetext', () => {
    render(<StageProgress done={1} total={2} cancelled={0} />)
    const bar = screen.getByRole('progressbar', { name: 'Avance de la etapa' })
    expect(bar.getAttribute('aria-valuenow')).toBe('50')
    expect(bar.getAttribute('aria-valuemin')).toBe('0')
    expect(bar.getAttribute('aria-valuemax')).toBe('100')
    expect(bar.getAttribute('aria-valuetext')).toBe('50 % (1 de 2 tareas)')
  })

  it('muestra el porcentaje y el cociente visibles', () => {
    render(<StageProgress done={1} total={2} cancelled={0} />)
    expect(screen.getByText('50 % · 1/2')).toBeDefined()
  })

  it('0 tareas muestra "0 %" y "0/0"', () => {
    render(<StageProgress done={0} total={0} cancelled={0} />)
    expect(screen.getByText('0 % · 0/0')).toBeDefined()
  })

  it('todas canceladas muestra "0 %" con denominador 0', () => {
    render(<StageProgress done={0} total={2} cancelled={2} />)
    expect(screen.getByText('0 % · 0/0')).toBeDefined()
  })
})
