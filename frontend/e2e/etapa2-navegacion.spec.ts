// Tarea 06 de la etapa 2 (plan 46fd2ac6 v1): recorrido continuo de navegación
// / → proyecto → etapa → recarga → atrás en tres variantes (escritorio con
// mouse, 360 px y solo teclado), con fixtures del harness aislado por prueba
// contra la API real (8090/pmt_frontend_test). Sin page.route: solo datos
// reales sembrados por SQL. Localiza siempre por el nombre único del fixture.
import { expect, type Locator, type Page } from '@playwright/test'
import { test, type Harness } from './harness/fixtures'

const COLUMNAS = [
  'Backlog',
  'Lista',
  'En desarrollo',
  'En revisión',
  'Cambios solicitados',
  'En QA',
  'Pendiente de cierre',
  'Terminada',
  'Cancelada',
]

const ETIQUETA_POR_STATUS: Record<string, string> = {
  backlog: 'Backlog',
  ready: 'Lista',
  in_dev: 'En desarrollo',
  in_review: 'En revisión',
  changes_requested: 'Cambios solicitados',
  in_qa: 'En QA',
  pending_close: 'Pendiente de cierre',
  done: 'Terminada',
  cancelled: 'Cancelada',
}

function sembrar(harness: Harness) {
  const proyecto = harness.createProject({ status: 'working', productVersion: '1.0.0' })
  const etapaA = harness.createStage(proyecto.id, { position: 1, status: 'in_dev' })
  const etapaB = harness.createStage(proyecto.id, { position: 2 })
  const tareas = [
    harness.createTask(etapaA.id, { status: 'backlog' }),
    harness.createTask(etapaA.id, { status: 'in_dev' }),
    harness.createTask(etapaA.id, { status: 'done' }),
    harness.createTask(etapaA.id, { status: 'cancelled' }),
  ]
  return { proyecto, etapaA, etapaB, tareas }
}

type Fixtures = ReturnType<typeof sembrar>

async function sinScrollGlobal(page: Page): Promise<boolean> {
  return page.evaluate(
    () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
  )
}

/** Abre una tarjeta con click, o solo con Tab hasta enfocarla y Enter. */
async function abrir(page: Page, tarjeta: Locator, nombre: string, teclado: boolean) {
  if (!teclado) {
    await tarjeta.click()
    return
  }
  for (let i = 0; i < 150; i += 1) {
    await page.keyboard.press('Tab')
    const texto = await page.evaluate(() => document.activeElement?.textContent ?? '')
    if (texto.includes(nombre)) break
  }
  await expect(tarjeta).toBeFocused()
  await page.keyboard.press('Enter')
}

async function recorrido(page: Page, modo: 'mouse' | 'movil' | 'teclado', fx: Fixtures) {
  const teclado = modo === 'teclado'
  if (modo === 'movil') await page.setViewportSize({ width: 360, height: 800 })

  // 1. Inicio: la tarjeta del proyecto es visible.
  await page.goto('/')
  const tarjetaProyecto = page.getByRole('button').filter({ hasText: fx.proyecto.name })
  await expect(tarjetaProyecto).toBeVisible()
  if (modo === 'movil') expect(await sinScrollGlobal(page)).toBe(true)

  // 2. Abrir el proyecto: ruta, h1 y las dos etapas con su avance.
  await abrir(page, tarjetaProyecto, fx.proyecto.name, teclado)
  await expect(page).toHaveURL(`/projects/${fx.proyecto.id}`)
  await expect(
    page.getByRole('heading', { level: 1, name: fx.proyecto.name }),
  ).toBeVisible()
  const tarjetaEtapaA = page.getByRole('button').filter({ hasText: fx.etapaA.name })
  const tarjetaEtapaB = page.getByRole('button').filter({ hasText: fx.etapaB.name })
  // Etapa A: 1 done de 3 no canceladas = 33 %; etapa B: sin tareas.
  await expect(tarjetaEtapaA).toContainText('33 % · 1/3')
  await expect(
    tarjetaEtapaA.getByRole('progressbar', { name: 'Avance de la etapa' }),
  ).toHaveAttribute('aria-valuenow', '33')
  await expect(tarjetaEtapaB).toContainText('0 % · 0/0')
  if (modo === 'movil') expect(await sinScrollGlobal(page)).toBe(true)

  // 3. Abrir la etapa A: ruta, tablero con las nueve secciones en orden,
  // cada tarea en la sección de su estado y el avance de la etapa.
  await abrir(page, tarjetaEtapaA, fx.etapaA.name, teclado)
  await expect(page).toHaveURL(`/projects/${fx.proyecto.id}/stages/${fx.etapaA.id}`)
  const board = page.getByRole('region', { name: 'Tablero de la etapa' })
  await expect(board).toBeVisible()
  const columnas = board.locator('section[aria-labelledby^="kanban-col-"]')
  await expect(columnas).toHaveCount(9)
  for (const [i, etiqueta] of COLUMNAS.entries()) {
    await expect(columnas.nth(i).locator('h3')).toContainText(etiqueta, { useInnerText: true })
  }
  for (const tarea of fx.tareas) {
    const seccion = board.getByRole('region', {
      name: new RegExp(`^${ETIQUETA_POR_STATUS[tarea.status]}`),
    })
    await expect(seccion.getByRole('button').filter({ hasText: tarea.title })).toBeVisible()
  }
  const headerEtapa = page
    .getByRole('heading', { level: 2, name: fx.etapaA.name })
    .locator('xpath=ancestor::header')
  await expect(headerEtapa).toContainText('33 % · 1/3')
  if (modo === 'movil') expect(await sinScrollGlobal(page)).toBe(true)

  // 4. Recargar: misma URL, mismo tablero y avance.
  await page.reload()
  await expect(page).toHaveURL(`/projects/${fx.proyecto.id}/stages/${fx.etapaA.id}`)
  await expect(board).toBeVisible()
  await expect(columnas).toHaveCount(9)
  await expect(headerEtapa).toContainText('33 % · 1/3')

  // 5. Atrás: vuelta al proyecto con su lista de etapas.
  await page.goBack()
  await expect(page).toHaveURL(`/projects/${fx.proyecto.id}`)
  await expect(
    page.getByRole('heading', { level: 1, name: fx.proyecto.name }),
  ).toBeVisible()
  await expect(tarjetaEtapaA).toContainText('33 % · 1/3')
  await expect(tarjetaEtapaB).toContainText('0 % · 0/0')
}

test.describe('tarea 06 — recorrido de navegación proyecto → etapa → kanban', () => {
  test('escritorio con mouse', async ({ page, harness }) => {
    await recorrido(page, 'mouse', sembrar(harness))
  })

  test('a 360 px sin scroll horizontal global en cada pantalla', async ({ page, harness }) => {
    await recorrido(page, 'movil', sembrar(harness))
  })

  test('solo con teclado (Tab y Enter, sin click)', async ({ page, harness }) => {
    await recorrido(page, 'teclado', sembrar(harness))
  })
})
