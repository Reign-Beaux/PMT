import { test, expect, chromium } from '@playwright/test'
import { execSync, spawn } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// Composición de fondos semitransparentes (badges con color-mix + transparent)
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

  // Fondo efectivo: acumula backgroundColors semitransparentes hasta uno opaco
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

  // Color de texto efectivo: primer color no transparente hacia arriba
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

test.describe('Etapa 1 — Componentes de carga, vacío y error (criterio 5)', () => {
  test('textos en español, foco visible en Reintentar y contraste AA', async ({
    page,
  }) => {
    const consoleErrors: string[] = []
    page.on('console', (msg) => {
      if (msg.type() === 'error') consoleErrors.push(msg.text())
    })
    page.on('pageerror', (err) => consoleErrors.push(String(err)))

    await page.goto('/dev/componentes')
    await expect(
      page.getByRole('heading', { name: 'Componentes' }),
    ).toBeVisible()

    // Textos en español
    await expect(page.getByRole('status').filter({ hasText: 'Cargando…' }).first()).toBeVisible()
    await expect(page.getByText('Cargando etapas…')).toBeVisible()
    await expect(page.getByText('No hay tareas en esta columna')).toBeVisible()
    await expect(
      page.getByRole('alert').filter({ hasText: 'Falló la conexión con el servidor' }),
    ).toBeVisible()
    await expect(page.getByText('En vivo')).toBeVisible()
    await expect(page.getByText('Reconectando…')).toBeVisible()
    await expect(page.getByText(/Sin conexión, datos de las \d{2}:\d{2}/)).toBeVisible()

    // Foco visible en Reintentar al navegar con Tab
    const retry = page.getByRole('button', { name: 'Reintentar' })
    await retry.focus()
    await page.keyboard.press('Tab')
    await page.keyboard.press('Shift+Tab')
    const focusInfo = await retry.evaluate((el) => {
      const cs = getComputedStyle(el)
      return {
        isFocused: document.activeElement === el,
        outlineVisible:
          cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2,
        height: el.getBoundingClientRect().height,
      }
    })
    expect(focusInfo.isFocused).toBe(true)
    expect(focusInfo.outlineVisible).toBe(true)
    expect(focusInfo.height).toBeGreaterThanOrEqual(44)

    // Contraste AA del mensaje de error y del botón
    const errorContrast = await page
      .getByRole('alert')
      .locator('span')
      .first()
      .evaluate(COMPOSITE_AND_CONTRAST)
    expect(errorContrast).toBeGreaterThanOrEqual(4.5)
    const retryContrast = await retry.evaluate(COMPOSITE_AND_CONTRAST)
    expect(retryContrast).toBeGreaterThanOrEqual(4.5)

    expect(consoleErrors).toEqual([])
  })

  test('sin overflow horizontal a 360 px', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto('/dev/componentes')
    const result = await page.evaluate(HORIZONTAL_OVERFLOW_CHECK)
    expect(result.overflow).toBeLessThanOrEqual(1)
    expect(result.outside).toBe(0)
  })
})

test.describe('Etapa 1 — Card y Badge (criterio 10)', () => {
  test('Card navegable con Tab y foco visible', async ({ page }) => {
    await page.goto('/dev/componentes')
    const card = page.getByRole('button', { name: /Tarea interactiva/ })

    await card.focus()
    const focusInfo = await card.evaluate((el) => {
      const cs = getComputedStyle(el)
      return {
        isFocused: document.activeElement === el,
        outlineVisible:
          cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2,
      }
    })
    expect(focusInfo.isFocused).toBe(true)
    expect(focusInfo.outlineVisible).toBe(true)

    // Tab desde Reintentar llega a la Card interactiva
    await page.getByRole('button', { name: 'Reintentar' }).focus()
    await page.keyboard.press('Tab')
    await expect(card).toBeFocused()

    // Las tarjetas sin onOpen (<article>) no son enfocables ni contienen controles
    const nonInteractive = page.locator('article', {
      hasText: 'Tarjeta no interactiva',
    })
    await expect(nonInteractive).toBeVisible()
    const focusableArticles = await page.evaluate(
      () =>
        Array.from(document.querySelectorAll('article')).filter(
          (el) =>
            el.hasAttribute('tabindex') ||
            el.querySelector('button, a[href], input') !== null,
        ).length,
    )
    expect(focusableArticles).toBe(0)
  })

  test('título largo legible y sin overflow a 360 px', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto('/dev/componentes')

    const longCard = page.locator('article', { hasText: 't-0002' })
    await expect(longCard).toBeVisible()
    const box = await longCard.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.width).toBeLessThanOrEqual(360)

    const result = await page.evaluate(HORIZONTAL_OVERFLOW_CHECK)
    expect(result.overflow).toBeLessThanOrEqual(1)
    expect(result.outside).toBe(0)
  })

  test('badges con contraste AA en los 5 tonos', async ({ page }) => {
    await page.goto('/dev/componentes')
    for (const label of [
      'Neutral',
      'Info',
      'Éxito',
      'Advertencia',
      'Peligro',
    ]) {
      const badge = page.locator('span', { hasText: new RegExp(`^${label}$`) })
      await expect(badge).toBeVisible()
      const ratio = await badge.evaluate(COMPOSITE_AND_CONTRAST)
      expect(ratio, `contraste del badge "${label}"`).toBeGreaterThanOrEqual(4.5)
    }
  })
})

test.describe('Etapa 1 — Barra de avance (criterio 13)', () => {
  test('porcentaje visible, ancho acorde y contraste AA', async ({ page }) => {
    await page.goto('/dev/componentes')

    // Textos visibles de cada caso
    await expect(page.getByText('0 % · 0/0')).toBeVisible()
    await expect(page.getByText('50 % · 1/2')).toBeVisible()
    await expect(page.getByText('67 % · 2/3')).toBeVisible()
    await expect(page.getByText('100 % · 3/3')).toBeVisible()

    // El ancho del relleno coincide con el porcentaje declarado
    for (const pct of [0, 50, 67, 100]) {
      const bar = page.locator(
        `[role="progressbar"][aria-valuenow="${pct}"]`,
      )
      await expect(bar).toHaveAttribute(
        'aria-valuetext',
        new RegExp(`^${pct} % \\(\\d+ de \\d+ tareas\\)$`),
      )
      const widths = await bar.evaluate((el) => {
        const track = el.firstElementChild as HTMLElement
        const fill = track.firstElementChild as HTMLElement
        return {
          track: track.getBoundingClientRect().width,
          fill: fill.getBoundingClientRect().width,
        }
      })
      const expected = (widths.track * pct) / 100
      expect(
        Math.abs(widths.fill - expected),
        `ancho del relleno al ${pct} %`,
      ).toBeLessThanOrEqual(2)
    }

    // Contraste AA del texto del porcentaje
    const label = page.getByText('50 % · 1/2')
    const ratio = await label.evaluate(COMPOSITE_AND_CONTRAST)
    expect(ratio).toBeGreaterThanOrEqual(4.5)
  })

  test('sin overflow horizontal a 360 px', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto('/dev/componentes')
    await expect(page.getByText('50 % · 1/2')).toBeVisible()
    const result = await page.evaluate(HORIZONTAL_OVERFLOW_CHECK)
    expect(result.overflow).toBeLessThanOrEqual(1)
    expect(result.outside).toBe(0)
  })
})

test.describe('Vitrina solo en desarrollo', () => {
  test('el build de producción no incluye /dev/componentes y responde "Página no encontrada"', async () => {
    test.setTimeout(240_000)
    const frontendDir = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      '..',
    )
    execSync('pnpm build', { cwd: frontendDir, stdio: 'pipe' })

    const preview = spawn('pnpm', ['preview', '--port', '4173', '--strictPort'], {
      cwd: frontendDir,
      stdio: 'ignore',
      detached: true,
    })
    const browser = await chromium.launch()
    try {
      const page = await browser.newPage()
      // Esperar a que preview responda (petición desde Node, sin CORS)
      await expect
        .poll(
          async () => {
            try {
              const res = await page.request.get('http://localhost:4173/')
              return res.status()
            } catch {
              return 0
            }
          },
          { timeout: 30_000 },
        )
        .toBe(200)
      await page.goto('http://localhost:4173/dev/componentes')
      await expect(
        page.getByRole('heading', { name: 'Página no encontrada' }),
      ).toBeVisible()
      await expect(page.getByText('Vitrina de desarrollo')).toHaveCount(0)
    } finally {
      await browser.close()
      process.kill(-preview.pid!, 'SIGTERM')
    }
  })
})
