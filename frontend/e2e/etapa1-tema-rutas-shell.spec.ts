// Regresión de etapa 1 (tema, rutas y shell). Desde la etapa 2 (plan 69dd00e2)
// el shell carga el proyecto por la API real y no pinta el Outlet si la carga
// falla, así que las rutas de proyecto usan un proyecto y una etapa sembrados
// en la base de pruebas (ids UUID reales) en vez de ids literales.
import { execFileSync } from 'node:child_process'
import { test, expect } from '@playwright/test'

const PROYECTO = 'bebe0000-0000-4000-8000-000000000001'
const ETAPA = 'bebe0000-0000-4000-8000-000000000002'
const TAREA = 'bebe0000-0000-4000-8000-000000000003'

function psql(sql: string): void {
  execFileSync(
    'docker',
    [
      'exec', '-i', 'postgresql',
      'psql', '-U', 'games_test', '-d', 'pmt_frontend_test',
      '-v', 'ON_ERROR_STOP=1', '-q', '-c', sql,
    ],
    { stdio: ['ignore', 'ignore', 'inherit'] },
  )
}

test.beforeAll(() => {
  psql(`DELETE FROM tasks WHERE stage_id = '${ETAPA}'`)
  psql(`DELETE FROM stages WHERE project_id = '${PROYECTO}'`)
  psql(`DELETE FROM projects WHERE id = '${PROYECTO}'`)
  psql(
    `INSERT INTO projects (id, name, description, status, product_version) VALUES
      ('${PROYECTO}', 'Proyecto QA tema rutas', 'Fixture de regresión', 'working', '1.0.0')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, description, position, status) VALUES
      ('${ETAPA}', '${PROYECTO}', 'Etapa QA tema rutas', 'Fixture', 1, 'in_dev')`,
  )
  // Desde la etapa 3 (tareas 03/05/06) ?task= abre el detalle real de la
  // tarea: la prueba de rutas necesita una tarea existente que mostrar.
  psql(
    `INSERT INTO tasks (id, stage_id, title, description, status) VALUES
      ('${TAREA}', '${ETAPA}', 'Tarea QA tema rutas', 'Fixture', 'backlog')`,
  )
})

test.afterAll(() => {
  psql(`DELETE FROM tasks WHERE stage_id = '${ETAPA}'`)
  psql(`DELETE FROM stages WHERE project_id = '${PROYECTO}'`)
  psql(`DELETE FROM projects WHERE id = '${PROYECTO}'`)
})

type Rgb = [number, number, number]

function parseRgb(value: string): Rgb {
  const parts = value.match(/\d+(\.\d+)?/g)?.map(Number) ?? []
  return [parts[0], parts[1], parts[2]]
}

function luminance([r, g, b]: Rgb): number {
  const channel = (c: number) => {
    const s = c / 255
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

function contrastRatio(fg: Rgb, bg: Rgb): number {
  const l1 = luminance(fg)
  const l2 = luminance(bg)
  return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05)
}

test.describe('Etapa 1 — Tema (criterio 2)', () => {
  test('fondo oscuro, contraste AA, foco visible, sin fuentes externas ni errores en consola', async ({
    page,
  }) => {
    const consoleErrors: string[] = []
    const externalRequests: string[] = []
    page.on('console', (msg) => {
      if (msg.type() === 'error') consoleErrors.push(msg.text())
    })
    page.on('pageerror', (err) => consoleErrors.push(String(err)))
    page.on('request', (req) => {
      const url = new URL(req.url())
      if (url.hostname !== 'localhost' && url.protocol !== 'data:') {
        externalRequests.push(req.url())
      }
    })

    await page.goto(`/projects/${PROYECTO}/stages/${ETAPA}`)

    const bg = await page.evaluate(
      () => getComputedStyle(document.body).backgroundColor,
    )
    expect(bg).toBe('rgb(20, 22, 26)')

    const fg = await page.evaluate(() => getComputedStyle(document.body).color)
    expect(contrastRatio(parseRgb(fg), parseRgb(bg))).toBeGreaterThanOrEqual(4.5)

    const headingColor = await page
      .getByRole('heading', { name: 'Etapa' })
      .evaluate((el) => getComputedStyle(el).color)
    expect(
      contrastRatio(parseRgb(headingColor), parseRgb(bg)),
    ).toBeGreaterThanOrEqual(4.5)

    const linkColor = await page
      .getByRole('link', { name: 'PMT' })
      .evaluate((el) => getComputedStyle(el).color)
    expect(
      contrastRatio(parseRgb(linkColor), parseRgb(bg)),
    ).toBeGreaterThanOrEqual(4.5)

    await page.keyboard.press('Tab')
    const focused = page.locator(':focus')
    await expect(focused).toHaveCount(1)
    const focusVisible = await focused.evaluate((el) => {
      const cs = getComputedStyle(el)
      return cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2
    })
    expect(focusVisible).toBe(true)

    expect(externalRequests).toEqual([])
    expect(consoleErrors).toEqual([])
  })
})

test.describe('Etapa 1 — Rutas y selección (criterio 6)', () => {
  test('enlaces, atrás/adelante y recarga conservan ruta y selección', async ({
    page,
  }) => {
    // Desde la etapa 2 (grupo 2) la vista de etapa ya no renderiza los textos
    // del placeholder ("Proyecto: …", "Etapa: …", "Seleccionado: …"): la
    // selección vive solo en el query ?task= de la URL. Desde la etapa 3
    // (tareas 03/05/06) ?task= abre un diálogo modal real, que bloquearía el
    // clic en los enlaces: la navegación por enlaces se prueba sin selección
    // y la selección se conserva por historial y recarga.
    const etapa = `/projects/${PROYECTO}/stages/${ETAPA}`
    await page.goto(etapa)
    await expect(
      page.getByRole('heading', { level: 2, name: 'Etapa QA tema rutas' }),
    ).toBeVisible()

    await page.getByRole('link', { name: 'Bugs' }).click()
    await expect(page).toHaveURL(`/projects/${PROYECTO}/bugs`)
    await expect(page.getByRole('heading', { name: 'Bugs' })).toBeVisible()

    await page.goBack()
    await expect(page).toHaveURL(etapa)

    await page.goto(`${etapa}?task=${TAREA}`)
    await expect(page.getByRole('dialog')).toContainText('Tarea QA tema rutas')

    await page.goBack()
    await expect(page).toHaveURL(etapa)
    await expect(page.getByRole('dialog')).toHaveCount(0)

    await page.goForward()
    await expect(page).toHaveURL(`${etapa}?task=${TAREA}`)
    await expect(page.getByRole('dialog')).toContainText('Tarea QA tema rutas')

    await page.reload()
    await expect(page).toHaveURL(`${etapa}?task=${TAREA}`)
    await expect(
      page.getByRole('heading', { level: 2, name: 'Etapa QA tema rutas' }),
    ).toBeVisible()
    await expect(page.getByRole('dialog')).toContainText('Tarea QA tema rutas')
  })

  test('selección de bug por URL directa', async ({ page }) => {
    // Desde la etapa 3 (tareas 02/04) la vista de bugs es un tablero real y la
    // selección ?bug= abre el diálogo; xyz no existe → "Bug no encontrado".
    await page.goto(`/projects/${PROYECTO}/bugs?bug=xyz`)
    await expect(
      page.getByRole('dialog', { name: 'Bug no encontrado' }),
    ).toBeVisible()
    await expect(page).toHaveURL(`/projects/${PROYECTO}/bugs?bug=xyz`)
  })
})

test.describe('Etapa 1 — Shell y navegación (criterio 9)', () => {
  test('recorrido con teclado por cabecera y pestañas con foco visible; Enter navega', async ({
    page,
  }) => {
    await page.goto(`/projects/${PROYECTO}/stages/${ETAPA}`)
    // El tablero se renderiza tras la carga async de la etapa: esperar a que
    // exista antes de tabular, si no el foco lo salta.
    await expect(
      page.getByRole('region', { name: 'Tablero de la etapa' }),
    ).toBeVisible()

    // Los enfocables son los del shell, el tablero (región con tabIndex) y la
    // tarjeta de la tarea sembrada. Se tabula hasta que el foco sale de la
    // página y todo lo enfocable debe tener foco visible.
    const focusedTexts: string[] = []
    for (let i = 0; i < 8; i += 1) {
      await page.keyboard.press('Tab')
      const info = await page.evaluate(() => {
        const el = document.activeElement
        if (!(el instanceof HTMLElement) || el === document.body) return null
        const cs = getComputedStyle(el)
        return {
          text:
            el.getAttribute('aria-label') || el.textContent?.trim() || '',
          focusVisible:
            cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2,
        }
      })
      if (info === null) break
      expect(info.focusVisible, `foco visible en "${info.text}"`).toBe(true)
      focusedTexts.push(info.text)
    }

    expect(focusedTexts).toEqual(
      expect.arrayContaining(['PMT', 'Etapas', 'Bugs', 'Tablero de la etapa']),
    )

    await page.getByRole('link', { name: 'Etapas' }).focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(`/projects/${PROYECTO}`)

    await page.getByRole('link', { name: 'PMT' }).focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL('/')
  })

  test('nombres largos ajustados y sin overflow horizontal a 360 px', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    const longId =
      'proyecto-qa-nombre-extremadamente-largo-sin-espacios-0123456789'
    await page.goto(`/projects/${longId}/stages/${longId}`)

    await expect(page.locator('header')).toContainText(longId)

    const horizontalOverflow = await page.evaluate(() => {
      const doc = document.documentElement
      return doc.scrollWidth - doc.clientWidth
    })
    expect(horizontalOverflow).toBeLessThanOrEqual(1)

    const elementsOutsideViewport = await page.evaluate(() => {
      const vw = document.documentElement.clientWidth
      return Array.from(document.querySelectorAll('body *')).filter(
        (el) => el.getBoundingClientRect().right > vw + 1,
      ).length
    })
    expect(elementsOutsideViewport).toBe(0)
  })
})
