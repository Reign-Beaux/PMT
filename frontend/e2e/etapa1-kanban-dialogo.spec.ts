import { test, expect } from '@playwright/test'

// Composición de fondos semitransparentes hasta el primer fondo opaco
const COMPOSITE_AND_CONTRAST = (el: Element) => {
  function parse(c: string): [number, number, number, number] {
    const p = c.match(/[\d.]+/g)?.map(Number) ?? []
    return [p[0], p[1], p[2], p.length > 3 ? p[3] : 1]
  }
  function blend(
    fg: [number, number, number, number],
    bg: [number, number, number],
  ): [number, number, number] {
    return [
      fg[0] * fg[3] + bg[0] * (1 - fg[3]),
      fg[1] * fg[3] + bg[1] * (1 - fg[3]),
      fg[2] * fg[3] + bg[2] * (1 - fg[3]),
    ]
  }
  function luminance([r, g, b]: [number, number, number]): number {
    const ch = (c: number) => {
      const s = c / 255
      return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
    }
    return 0.2126 * ch(r) + 0.7152 * ch(g) + 0.0722 * ch(b)
  }

  const layers: [number, number, number, number][] = []
  let node: Element | null = el
  let opaque: [number, number, number] = [255, 255, 255]
  while (node) {
    const bg = parse(getComputedStyle(node).backgroundColor)
    if (bg[3] >= 1) {
      opaque = [bg[0], bg[1], bg[2]]
      break
    }
    if (bg[3] > 0) layers.push(bg)
    node = node.parentElement
  }
  let effective = opaque
  for (let i = layers.length - 1; i >= 0; i -= 1) {
    effective = blend(layers[i], effective)
  }

  let fg: [number, number, number, number] = [0, 0, 0, 1]
  let tnode: Element | null = el
  while (tnode) {
    const c = parse(getComputedStyle(tnode).color)
    if (c[3] > 0) {
      fg = c
      break
    }
    tnode = tnode.parentElement
  }
  const fgRgb: [number, number, number] =
    fg[3] >= 1 ? [fg[0], fg[1], fg[2]] : blend(fg, effective)

  const l1 = luminance(fgRgb)
  const l2 = luminance(effective)
  return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05)
}

const HORIZONTAL_OVERFLOW_CHECK = () => {
  const doc = document.documentElement
  const overflow = doc.scrollWidth - doc.clientWidth
  const vw = doc.clientWidth
  // Un elemento solo cuenta como desbordado si ningún ancestro con scroll/clip
  // horizontal (p. ej. el tablero kanban con overflow-x: auto) lo recorta
  // dentro del viewport.
  const outside = Array.from(document.querySelectorAll('body *')).filter(
    (el) => {
      if (el.getBoundingClientRect().right <= vw + 1) return false
      let node = el.parentElement
      while (node && node !== document.body) {
        const ox = getComputedStyle(node).overflowX
        if (ox !== 'visible' && node.getBoundingClientRect().right <= vw + 1) {
          return false
        }
        node = node.parentElement
      }
      return true
    },
  ).length
  return { overflow, outside }
}

const KANBAN_LABELS = [
  'Backlog',
  'Ready',
  'En desarrollo',
  'En revisión',
  'En QA',
  'Cambios pedidos',
  'Pending close',
  'Done',
  'Cancelled',
]

const EMPTY_COLUMNS = ['En revisión', 'Cambios pedidos', 'Cancelled']

test.describe('Etapa 1 — Kanban (criterio 3)', () => {
  test('foco visible, desplazamiento interno y columnas alcanzables con flechas', async ({
    page,
  }) => {
    await page.goto('/dev/componentes')
    const board = page.getByRole('region', { name: 'Tablero de ejemplo' })
    await expect(board).toBeVisible()

    // El tablero es un contenedor de scroll horizontal enfocable
    const boardInfo = await board.evaluate((el) => ({
      overflowX: getComputedStyle(el).overflowX,
      tabIndex: (el as HTMLElement).tabIndex,
      scrollWidth: el.scrollWidth,
      clientWidth: el.clientWidth,
    }))
    expect(boardInfo.overflowX).toBe('auto')
    expect(boardInfo.tabIndex).toBe(0)
    expect(boardInfo.scrollWidth).toBeGreaterThan(boardInfo.clientWidth)

    // Alcanzable con Tab desde el botón anterior y con foco visible
    await page.getByRole('progressbar').last().focus()
    let reached = false
    for (let i = 0; i < 30 && !reached; i += 1) {
      await page.keyboard.press('Tab')
      reached = await board.evaluate((el) => document.activeElement === el)
    }
    expect(reached).toBe(true)
    const focusVisible = await board.evaluate((el) => {
      const cs = getComputedStyle(el)
      return cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2
    })
    expect(focusVisible).toBe(true)

    // Con el tablero enfocado, las flechas lo desplazan hasta que todas las
    // columnas (incluidas las vacías) quedan visibles en el viewport
    const seen = new Set<string>()
    const collectVisible = async () => {
      const vw = page.viewportSize()!.width
      for (const label of KANBAN_LABELS) {
        if (seen.has(label)) continue
        const box = await page
          .locator('h3', { hasText: label })
          .first()
          .boundingBox()
        if (box && box.x >= -1 && box.x + box.width <= vw + 1) {
          seen.add(label)
        }
      }
    }
    await collectVisible()
    for (let i = 0; i < 60 && seen.size < KANBAN_LABELS.length; i += 1) {
      await page.keyboard.press('ArrowRight')
      await collectVisible()
    }
    expect([...seen].sort()).toEqual([...KANBAN_LABELS].sort())

    // Las columnas vacías muestran "Sin elementos"
    for (const label of EMPTY_COLUMNS) {
      const column = page
        .locator('section[aria-labelledby]')
        .filter({ has: page.locator('h3', { hasText: label }) })
        .first()
      await expect(column.getByText('Sin elementos')).toBeVisible()
    }

    // Contraste AA del contador de columna
    const counter = page.locator('h3 span[aria-label$="elementos"]').first()
    const ratio = await counter.evaluate(COMPOSITE_AND_CONTRAST)
    expect(ratio).toBeGreaterThanOrEqual(4.5)
  })

  test('a 360 px la página no desborda y el tablero se desplaza internamente', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto('/dev/componentes')
    const board = page.getByRole('region', { name: 'Tablero de ejemplo' })
    await expect(board).toBeVisible()

    const result = await page.evaluate(HORIZONTAL_OVERFLOW_CHECK)
    expect(result.overflow).toBeLessThanOrEqual(1)
    expect(result.outside).toBe(0)

    // Las columnas conservan su ancho y el scroll vive dentro del tablero
    const info = await board.evaluate((el) => ({
      scrollWidth: el.scrollWidth,
      clientWidth: el.clientWidth,
      columnWidth: el.firstElementChild?.getBoundingClientRect().width ?? 0,
    }))
    expect(info.scrollWidth).toBeGreaterThan(info.clientWidth)
    expect(info.clientWidth).toBeLessThanOrEqual(360)
    expect(info.columnWidth).toBeGreaterThanOrEqual(200) // ~16rem
  })
})

test.describe('Etapa 1 — Diálogo de detalle (criterio 6)', () => {
  test('abre con teclado, foco dentro, Escape y Cerrar devuelven el foco', async ({
    page,
  }) => {
    await page.goto('/dev/componentes')
    const trigger = page.getByRole('button', { name: 'Abrir detalle' })
    const dialog = page.locator('dialog[open]')
    const close = page.getByRole('button', { name: 'Cerrar' })

    // Se abre con teclado (Enter sobre el botón enfocado)
    await trigger.focus()
    await page.keyboard.press('Enter')
    await expect(dialog).toBeVisible()

    // El foco queda dentro del diálogo, en el botón Cerrar
    await expect(close).toBeFocused()

    // El foco no escapa del diálogo al pulsar Tab
    await page.keyboard.press('Tab')
    await page.keyboard.press('Tab')
    const focusInside = await dialog.evaluate(
      (el) => el.contains(document.activeElement) || document.activeElement === el,
    )
    expect(focusInside).toBe(true)

    // Escape cierra y el foco vuelve a "Abrir detalle"
    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    await expect(trigger).toBeFocused()

    // Se abre de nuevo con teclado (Space) y el clic en Cerrar cierra
    await page.keyboard.press('Space')
    await expect(dialog).toBeVisible()
    await close.click()
    await expect(dialog).toHaveCount(0)
    await expect(trigger).toBeFocused()
  })

  test('a 360 px el panel es usable, sin overflow global y el título largo se parte', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto('/dev/componentes')
    const trigger = page.getByRole('button', { name: 'Abrir detalle' })
    await trigger.focus()
    await page.keyboard.press('Enter')
    const dialog = page.locator('dialog[open]')
    await expect(dialog).toBeVisible()

    // Panel casi a ancho completo y dentro del viewport
    const box = await dialog.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.width).toBeLessThanOrEqual(360)
    expect(box!.width).toBeGreaterThanOrEqual(300)

    // Botón Cerrar visible y usable
    const close = page.getByRole('button', { name: 'Cerrar' })
    await expect(close).toBeVisible()
    const closeBox = await close.boundingBox()
    expect(closeBox!.height).toBeGreaterThanOrEqual(32)

    // Sin overflow global con el diálogo abierto
    const result = await page.evaluate(HORIZONTAL_OVERFLOW_CHECK)
    expect(result.overflow).toBeLessThanOrEqual(1)
    expect(result.outside).toBe(0)

    // El título largo se parte en varias líneas sin desbordar el panel
    const title = dialog.locator('h2')
    const titleInfo = await title.evaluate((el) => {
      const cs = getComputedStyle(el)
      const lineHeight =
        parseFloat(cs.lineHeight) || parseFloat(cs.fontSize) * 1.2
      return {
        lines: Math.round(el.getBoundingClientRect().height / lineHeight),
        scrollWidth: el.scrollWidth,
        clientWidth: el.clientWidth,
      }
    })
    expect(titleInfo.lines).toBeGreaterThanOrEqual(2)
    expect(titleInfo.scrollWidth).toBeLessThanOrEqual(titleInfo.clientWidth + 1)
  })
})
