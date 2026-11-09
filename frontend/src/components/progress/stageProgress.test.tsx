import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StageProgress } from './StageProgress'
import { progressSegments, stageProgressPercent } from './stageProgress'

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

describe('progressSegments', () => {
  it('da los status en el orden de TASK_STATUS_LABELS', () => {
    const segments = progressSegments({ done: 1, backlog: 2, in_dev: 3 })

    expect(segments.map((segment) => segment.status)).toEqual([
      'backlog',
      'in_dev',
      'done',
    ])
  })

  it('los percents suman 100', () => {
    const segments = progressSegments({ done: 2, in_dev: 1 })

    const sum = segments.reduce((total, segment) => total + segment.percent, 0)
    expect(sum).toBeCloseTo(100, 6)
    expect(segments.map((segment) => segment.count)).toEqual([1, 2])
    expect(segments.map((segment) => segment.tone)).toEqual([
      'info',
      'success',
    ])
  })

  it('descarta claves desconocidas, 0, negativos y NaN', () => {
    const segments = progressSegments({
      done: 1,
      inventado: 5,
      backlog: 0,
      ready: -2,
      in_qa: Number.NaN,
    })

    expect(segments.map((segment) => segment.status)).toEqual(['done'])
    expect(segments[0].percent).toBe(100)
  })

  it('suma 0 → []', () => {
    expect(progressSegments({})).toEqual([])
    expect(progressSegments({ done: 0, backlog: 0 })).toEqual([])
  })

  it('un solo status → un segmento al 100 %', () => {
    const segments = progressSegments({ in_review: 4 })

    expect(segments).toHaveLength(1)
    expect(segments[0].percent).toBe(100)
  })
})

describe('StageProgress con counts', () => {
  it('pinta un segmento por status con data-status y aria-hidden', () => {
    const { container } = render(
      <StageProgress done={2} total={3} cancelled={0} counts={{ done: 2, in_dev: 1 }} />,
    )

    const segments = container.querySelectorAll('[data-status]')
    expect(segments).toHaveLength(2)
    expect(segments[0].getAttribute('data-status')).toBe('in_dev')
    expect(segments[1].getAttribute('data-status')).toBe('done')
    expect(segments[0].getAttribute('aria-hidden')).toBe('true')
    expect(segments[0].getAttribute('style')).toContain('33.33')
    expect(segments[1].getAttribute('style')).toContain('66.66')
  })

  it('conserva role, aria-valuetext y texto calculados con done/total/cancelled', () => {
    render(
      <StageProgress done={2} total={3} cancelled={0} counts={{ done: 2, in_dev: 1 }} />,
    )

    const bar = screen.getByRole('progressbar', { name: 'Avance de la etapa' })
    expect(bar.getAttribute('aria-valuenow')).toBe('67')
    expect(bar.getAttribute('aria-valuetext')).toBe('67 % (2 de 3 tareas)')
    expect(screen.getByText('67 % · 2/3')).toBeDefined()
  })

  it('counts vacíos pintan solo el riel', () => {
    const { container } = render(
      <StageProgress done={0} total={0} cancelled={0} counts={{}} />,
    )

    expect(container.querySelectorAll('[data-status]')).toHaveLength(0)
    expect(screen.getByText('0 % · 0/0')).toBeDefined()
  })

  it('sin counts conserva el fill único', () => {
    const { container } = render(
      <StageProgress done={1} total={2} cancelled={0} />,
    )

    expect(container.querySelectorAll('[data-status]')).toHaveLength(0)
  })
})
