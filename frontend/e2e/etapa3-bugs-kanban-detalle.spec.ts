// Regresión QA de las tareas 02/04 de la etapa 3 (criterios "QA en pantalla"
// del plan 1eb2165f v1): kanban de /projects/:id/bugs con los bugs de la base
// en su columna (título y versión, incluidos done y rejected), desplazamiento
// horizontal del tablero a 360 px sin scroll global, alcance por teclado de
// columnas y tarjetas, detalle de bug cotejado con la base, recarga con
// ?bug=, foco devuelto a la tarjeta con Escape y diálogo usable a 360 px.
// API real (backend de pruebas en 8090, base pmt_frontend_test). Siembra y
// limpia sus propios fixtures por SQL con el cliente del proyecto
// (docker exec postgresql psql). Si la instancia de pruebas no está
// levantada, los tests se omiten.
import { execFileSync } from 'node:child_process'
import { expect, test, type Page } from '@playwright/test'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'

const PROYECTO = 'eeee0000-0000-4000-8000-000000000001'
const NOMBRE_P = 'Proyecto QA bugs etapa3'

// Bugs con id fijo del fixture; b01 y b10 comparten la columna "Reportado".
const BUGS = {
  html: 'eeee0000-0000-4000-8000-0000000000b1', // reported, HTML y saltos
  confirmado: 'eeee0000-0000-4000-8000-0000000000b2', // confirmed, sin reported_by
  enDev: 'eeee0000-0000-4000-8000-0000000000b3',
  enRevision: 'eeee0000-0000-4000-8000-0000000000b4',
  cambios: 'eeee0000-0000-4000-8000-0000000000b5',
  enQa: 'eeee0000-0000-4000-8000-0000000000b6',
  pendiente: 'eeee0000-0000-4000-8000-0000000000b7',
  hecho: 'eeee0000-0000-4000-8000-0000000000b8',
  rechazado: 'eeee0000-0000-4000-8000-0000000000b9',
  reciente: 'eeee0000-0000-4000-8000-0000000000ba', // reported más nuevo que b1
} as const

const COLUMNAS = [
  'Reportado',
  'Confirmado',
  'En desarrollo',
  'En revisión',
  'Cambios solicitados',
  'En QA',
  'Pendiente de cierre',
  'Corregido',
  'Rechazado',
]

const STATUSES = [
  'reported',
  'confirmed',
  'in_dev',
  'in_review',
  'changes_requested',
  'in_qa',
  'pending_close',
  'done',
  'rejected',
]

const DESCRIPCION_HTML = '<b>no negrita</b> & <i>literal</i>\nsegunda línea'
const PASOS = '1. abrir\n2. pulsar <enter>\n3. ver fallo'

function psql(sql: string): string {
  return execFileSync(
    'docker',
    [
      'exec', '-i', 'postgresql',
      'psql', '-U', 'games_test', '-d', 'pmt_frontend_test',
      '-v', 'ON_ERROR_STOP=1', '-qtA', '-c', sql,
    ],
    { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] },
  ).trim()
}

function limpiar(): void {
  psql(`DELETE FROM bugs WHERE project_id = '${PROYECTO}'`)
  psql(`DELETE FROM projects WHERE id = '${PROYECTO}'`)
}

/** Bugs de la base: title|status|product_version|reported_by. */
function bugsDesdeLaBase(): { title: string; status: string; version: string; reportedBy: string | null }[] {
  return psql(
    `SELECT title, status, product_version, reported_by FROM bugs WHERE project_id = '${PROYECTO}'`,
  )
    .split('\n')
    .filter(Boolean)
    .map((fila) => {
      const [title, status, version, reportedBy] = fila.split('|')
      return { title, status, version, reportedBy: reportedBy === '' ? null : reportedBy }
    })
}

/** Fecha de la base formateada como hace formatDate (Intl es, medium/short). */
function fechaDesdeLaBase(bugId: string, campo: 'created_at' | 'updated_at'): string {
  const epoch = Number(psql(`SELECT extract(epoch from ${campo}) FROM bugs WHERE id = '${bugId}'`))
  return new Intl.DateTimeFormat('es', { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(epoch * 1000),
  )
}

let apiDisponible = false

test.beforeAll(async () => {
  try {
    apiDisponible = (await fetch(`${API}/api/projects`)).ok
  } catch {
    apiDisponible = false
  }
  if (!apiDisponible) return

  limpiar()
  psql(
    `INSERT INTO projects (id, name, description, status, product_version) VALUES
      ('${PROYECTO}', '${NOMBRE_P}', 'Fixture de regresión QA bugs', 'working', '1.0.0')`,
  )
  psql(
    `INSERT INTO bugs (id, project_id, title, description, steps_to_reproduce, product_version, status, reported_by, created_at, updated_at) VALUES
      ('${BUGS.html}', '${PROYECTO}', 'B1 con html', '${DESCRIPCION_HTML.replace(/'/g, "''")}', '${PASOS}', '1.2.3', 'reported', 'qa-r4-usuario', '2026-10-30 09:30:00-06', '2026-11-01 10:00:00-06'),
      ('${BUGS.confirmado}', '${PROYECTO}', 'B2 confirmado', 'Sin reportador', 'Paso único', '1.2.3', 'confirmed', NULL, '2026-10-31 09:00:00-06', '2026-10-31 09:00:00-06'),
      ('${BUGS.enDev}', '${PROYECTO}', 'B3 en dev', 'Descripción B3', 'Pasos B3', '1.3.0', 'in_dev', 'qa-r4-dev', '2026-11-01 09:00:00-06', '2026-11-01 09:00:00-06'),
      ('${BUGS.enRevision}', '${PROYECTO}', 'B4 en revision', 'Descripción B4', 'Pasos B4', '1.3.0', 'in_review', 'qa-r4-dev', '2026-11-01 09:10:00-06', '2026-11-01 09:10:00-06'),
      ('${BUGS.cambios}', '${PROYECTO}', 'B5 cambios', 'Descripción B5', 'Pasos B5', '1.3.0', 'changes_requested', 'qa-r4-usuario', '2026-11-01 09:20:00-06', '2026-11-01 09:20:00-06'),
      ('${BUGS.enQa}', '${PROYECTO}', 'B6 en qa', 'Descripción B6', 'Pasos B6', '1.4.0', 'in_qa', 'qa-r4-qa', '2026-11-02 09:00:00-06', '2026-11-02 09:00:00-06'),
      ('${BUGS.pendiente}', '${PROYECTO}', 'B7 pendiente', 'Descripción B7', 'Pasos B7', '1.4.0', 'pending_close', 'qa-r4-qa', '2026-11-02 09:10:00-06', '2026-11-02 09:10:00-06'),
      ('${BUGS.hecho}', '${PROYECTO}', 'B8 corregido', 'Descripción B8', 'Pasos B8', '1.4.0', 'done', 'qa-r4-dev', '2026-11-03 09:00:00-06', '2026-11-03 09:00:00-06'),
      ('${BUGS.rechazado}', '${PROYECTO}', 'B9 rechazado', 'Descripción B9', 'Pasos B9', '1.4.0', 'rejected', 'qa-r4-usuario', '2026-11-03 09:10:00-06', '2026-11-03 09:10:00-06'),
      ('${BUGS.reciente}', '${PROYECTO}', 'Ba reportado reciente', 'Descripción Ba', 'Pasos Ba', '1.5.0', 'reported', 'qa-r4-usuario', '2026-11-04 09:00:00-06', '2026-11-05 10:00:00-06')`,
  )
})

test.afterAll(() => {
  if (!apiDisponible) return
  limpiar()
})

test.beforeEach(() => {
  test.skip(!apiDisponible, 'Instancia de pruebas (8090/pmt_frontend_test) no disponible')
})

async function irAlTablero(page: Page) {
  await page.goto(`/projects/${PROYECTO}/bugs`)
  await expect(page.getByRole('region', { name: 'Tablero de bugs' })).toBeVisible()
}

function tarjeta(page: Page, titulo: string) {
  return page
    .getByRole('region', { name: 'Tablero de bugs' })
    .getByRole('button')
    .filter({ hasText: titulo })
}

test.describe('tarea 02 — kanban de bugs', () => {
  test('las nueve columnas en orden con los bugs de la base, título y versión, incluidos done y rejected', async ({ page }) => {
    await irAlTablero(page)

    const board = page.getByRole('region', { name: 'Tablero de bugs' })
    const columnas = board.locator('section[aria-labelledby^="kanban-col-"]')
    await expect(columnas).toHaveCount(9)
    for (const [i, etiqueta] of COLUMNAS.entries()) {
      await expect(columnas.nth(i).locator('h3')).toContainText(etiqueta, { useInnerText: true })
    }

    // Cada bug de la base está en la columna de su status, con título y versión.
    const bugs = bugsDesdeLaBase()
    expect(bugs.length).toBe(10)
    const porStatus = new Map<string, number>()
    for (const b of bugs) porStatus.set(b.status, (porStatus.get(b.status) ?? 0) + 1)
    for (const [i, status] of STATUSES.entries()) {
      await expect(columnas.nth(i).locator('h3')).toContainText(
        String(porStatus.get(status) ?? 0),
      )
    }
    const etiquetasPorStatus: Record<string, string> = Object.fromEntries(
      STATUSES.map((s, i) => [s, COLUMNAS[i]]),
    )
    for (const bug of bugs) {
      const columna = board.getByRole('region', {
        name: new RegExp(`^${etiquetasPorStatus[bug.status]}`),
      })
      const card = columna.getByRole('button').filter({ hasText: bug.title })
      await expect(card).toBeVisible()
      await expect(card).toContainText(`Versión ${bug.version}`)
    }
  })

  test('dentro de una columna el orden es por updated_at descendente', async ({ page }) => {
    await irAlTablero(page)

    const reportados = page
      .getByRole('region', { name: 'Tablero de bugs' })
      .getByRole('region', { name: /^Reportado/ })
    const titulos = await reportados.getByRole('button').allInnerTexts()
    expect(titulos[0]).toContain('Ba reportado reciente') // updated 2026-11-05
    expect(titulos[1]).toContain('B1 con html') // updated 2026-11-01
  })

  test('a 360 px el tablero desplaza en horizontal sin scroll global', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await irAlTablero(page)

    const medidas = await page.evaluate(() => {
      const board = document.querySelector('[aria-label="Tablero de bugs"]')
      if (board === null) return null
      const antes = board.scrollLeft
      board.scrollLeft = 300
      const despues = board.scrollLeft
      return {
        sinScrollGlobal:
          document.documentElement.scrollWidth <= document.documentElement.clientWidth,
        desborda: board.scrollWidth > board.clientWidth,
        desplaza: despues > antes,
      }
    })
    expect(medidas).not.toBeNull()
    expect(medidas!.sinScrollGlobal).toBe(true)
    expect(medidas!.desborda).toBe(true)
    expect(medidas!.desplaza).toBe(true)
  })

  test('columnas y tarjetas se alcanzan con teclado y Enter deja ?bug=:id', async ({ page }) => {
    await irAlTablero(page)

    const board = page.getByRole('region', { name: 'Tablero de bugs' })
    // Las nueve columnas están en el DOM en orden y son alcanzables: al
    // tabular desde el tablero se recorren las tarjetas de todas ellas.
    await expect(board.locator('section[aria-labelledby^="kanban-col-"]')).toHaveCount(9)
    await board.focus()

    const esperadas = [
      'Ba reportado reciente', 'B1 con html', 'B2 confirmado', 'B3 en dev',
      'B4 en revision', 'B5 cambios', 'B6 en qa', 'B7 pendiente',
      'B8 corregido', 'B9 rechazado',
    ]
    const alcanzadas: string[] = []
    for (let i = 0; i < 80 && alcanzadas.length < esperadas.length; i += 1) {
      await page.keyboard.press('Tab')
      const texto = await page.evaluate(() => document.activeElement?.textContent ?? '')
      const siguiente = esperadas.find(
        (t) => texto.includes(t) && !alcanzadas.includes(t),
      )
      if (siguiente !== undefined) alcanzadas.push(siguiente)
    }
    expect(alcanzadas).toEqual(esperadas)

    // Enter sobre la tarjeta enfocada (B9 rechazado) escribe ?bug=:id.
    await page.keyboard.press('Enter')
    const search = new URL(page.url()).searchParams.get('bug')
    expect(search).toBe(BUGS.rechazado)
  })
})

test.describe('tarea 04 — detalle de bug', () => {
  test('abrir un bug muestra sus datos correctos frente a la base', async ({ page }) => {
    await irAlTablero(page)

    await tarjeta(page, 'B1 con html').click()
    expect(new URL(page.url()).searchParams.get('bug')).toBe(BUGS.html)

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog).toContainText('B1 con html')

    // Estado, versión y reportado por, cotejados con la base.
    const [status, version, reportedBy] = psql(
      `SELECT status, product_version, reported_by FROM bugs WHERE id = '${BUGS.html}'`,
    ).split('|')
    expect(status).toBe('reported')
    expect(version).toBe('1.2.3')
    expect(reportedBy).toBe('qa-r4-usuario')
    const facts = dialog.locator('dl')
    await expect(facts).toContainText('Reportado')
    await expect(facts).toContainText(version)
    await expect(facts).toContainText(reportedBy)

    // Fechas formateadas desde los valores de la base.
    await expect(facts).toContainText(fechaDesdeLaBase(BUGS.html, 'created_at'))
    await expect(facts).toContainText(fechaDesdeLaBase(BUGS.html, 'updated_at'))

    // Descripción y pasos literales: el HTML no se interpreta y se respetan
    // los saltos de línea (white-space: pre-wrap).
    const descripcion = dialog.getByRole('region', { name: 'Descripción' }).locator('p')
    await expect(descripcion).toContainText(DESCRIPCION_HTML)
    expect(await descripcion.evaluate((el) => getComputedStyle(el).whiteSpace)).toBe('pre-wrap')
    await expect(
      dialog.getByRole('region', { name: 'Pasos para reproducir' }).locator('p'),
    ).toContainText(PASOS)
    await expect(dialog.locator('b, i')).toHaveCount(0)
  })

  test('recargar con ?bug= abre el detalle tras cargar la lista', async ({ page }) => {
    await page.goto(`/projects/${PROYECTO}/bugs?bug=${BUGS.enQa}`)

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog).toContainText('B6 en qa')
    await expect(dialog.locator('dl')).toContainText('En QA')
  })

  test('un id inexistente muestra "Bug no encontrado" y se cierra normalmente', async ({ page }) => {
    await page.goto(`/projects/${PROYECTO}/bugs?bug=eeee0000-0000-4000-8000-ffffffffffff`)

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog).toContainText('Bug no encontrado')
    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    expect(new URL(page.url()).searchParams.get('bug')).toBeNull()
  })

  test('sin reported_by la fila no aparece', async ({ page }) => {
    await irAlTablero(page)

    await tarjeta(page, 'B2 confirmado').click()
    const facts = page.getByRole('dialog').locator('dl')
    await expect(facts).toContainText('Confirmado')
    await expect(facts).not.toContainText('Reportado por')
  })

  test('Escape quita ?bug= y devuelve el foco a la tarjeta', async ({ page }) => {
    await irAlTablero(page)

    await tarjeta(page, 'B3 en dev').click()
    await expect(page.getByRole('dialog')).toBeVisible()

    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(new URL(page.url()).searchParams.get('bug')).toBeNull()
    const enfocada = await page.evaluate(() => document.activeElement?.textContent ?? '')
    expect(enfocada).toContain('B3 en dev')
  })

  test('el diálogo es usable a 360 px', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto(`/projects/${PROYECTO}/bugs?bug=${BUGS.html}`)

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    const medidas = await dialog.evaluate((el) => {
      const rect = el.getBoundingClientRect()
      return {
        ancho: rect.width,
        dentroDelViewport: rect.left >= 0 && rect.right <= window.innerWidth,
        sinDesbordeInterno: el.scrollWidth <= el.clientWidth + 1,
      }
    })
    expect(medidas.ancho).toBeLessThanOrEqual(360)
    expect(medidas.dentroDelViewport).toBe(true)
    expect(medidas.sinDesbordeInterno).toBe(true)
    // El contenido clave sigue visible y el texto largo se ajusta.
    await expect(dialog).toContainText('B1 con html')
    await expect(dialog.locator('dl')).toContainText('qa-r4-usuario')
  })
})
