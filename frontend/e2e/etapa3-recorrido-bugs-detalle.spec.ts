// Tarea 07 de la etapa 3 (plan a7686281 v2): recorrido continuo contra la API
// real (8090/pmt_frontend_test) sin page.route: pestaña Bugs con cada bug en su
// columna (incluidos done y rejected) → detalle del bug con versión, comentario
// e historial → Escape quita ?bug= y devuelve el foco a la tarjeta → URL
// directa con ?task= muestra el detalle de la tarea con comentario e historial
// → recargar conserva el detalle. Se repite a 360 px. Siembra por prueba con
// los fixtures del harness aislado; localiza siempre por el título único.
import { expect, type Locator, type Page } from '@playwright/test'
import { BUG_STATUS_LABELS, TASK_STATUS_LABELS } from '../src/domain/status'
import { test, type Harness } from './harness/fixtures'

const VERSION_BUG = '1.2.3'
const COMENTARIO_BUG = 'comentario del bug'
const MOTIVO_BUG = 'confirmado y tomado'
const COMENTARIO_TAREA = 'comentario de la tarea'
const MOTIVO_TAREA = 'arranque de la tarea'

function sembrar(harness: Harness) {
  const proyecto = harness.createProject({ status: 'working', productVersion: '1.0.0' })
  const etapa = harness.createStage(proyecto.id, { position: 1, status: 'in_dev' })
  const tarea = harness.createTask(etapa.id, { status: 'in_dev' })
  harness.createComment('task', tarea.id, COMENTARIO_TAREA)
  harness.createTaskStatusChange(tarea.id, 'ready', 'in_dev', MOTIVO_TAREA)
  const bugReported = harness.createBug(proyecto.id, { status: 'reported' })
  const bugInDev = harness.createBug(proyecto.id, {
    status: 'in_dev',
    productVersion: VERSION_BUG,
  })
  const bugDone = harness.createBug(proyecto.id, { status: 'done' })
  const bugRejected = harness.createBug(proyecto.id, { status: 'rejected' })
  harness.createComment('bug', bugInDev.id, COMENTARIO_BUG)
  harness.createBugStatusChange(bugInDev.id, 'reported', 'in_dev', MOTIVO_BUG)
  const bugs = [bugReported, bugInDev, bugDone, bugRejected].map((bug) => ({
    ...bug,
    etiqueta: BUG_STATUS_LABELS[bug.status as keyof typeof BUG_STATUS_LABELS],
  }))
  return { proyecto, etapa, tarea, bugs, bugConDatos: bugInDev }
}

type Fixtures = ReturnType<typeof sembrar>

function columna(board: Locator, etiqueta: string) {
  return board.getByRole('region', { name: new RegExp(`^${etiqueta}`) })
}

function tarjetaDe(board: Locator, fx: Fixtures, title: string) {
  const bug = fx.bugs.find((b) => b.title === title)
  return columna(board, bug?.etiqueta ?? '')
    .getByRole('button')
    .filter({ hasText: title })
}

async function recorrido(page: Page, movil: boolean, fx: Fixtures) {
  if (movil) await page.setViewportSize({ width: 360, height: 640 })

  // 1. Pestaña Bugs: cada bug sembrado aparece en la columna de su etiqueta.
  await page.goto(`/projects/${fx.proyecto.id}/bugs`)
  const board = page.getByRole('region', { name: 'Tablero de bugs' })
  await expect(board).toBeVisible()
  for (const bug of fx.bugs) {
    const tarjeta = tarjetaDe(board, fx, bug.title)
    await tarjeta.scrollIntoViewIfNeeded()
    await expect(tarjeta).toBeVisible()
  }

  // 2. Abrir el bug con datos: título, versión, comentario y cambio de estado.
  const tarjetaBug = tarjetaDe(board, fx, fx.bugConDatos.title)
  await tarjetaBug.scrollIntoViewIfNeeded()
  await tarjetaBug.click()
  expect(new URL(page.url()).searchParams.get('bug')).toBe(fx.bugConDatos.id)
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog).toContainText(fx.bugConDatos.title)
  await expect(dialog).toContainText(VERSION_BUG)
  await expect(dialog.getByRole('region', { name: 'Comentarios' })).toContainText(COMENTARIO_BUG)
  const historialBug = dialog.getByRole('region', { name: 'Historial de estados' })
  await expect(historialBug).toContainText(
    `${BUG_STATUS_LABELS.reported} → ${BUG_STATUS_LABELS.in_dev}`,
  )
  await expect(historialBug).toContainText(MOTIVO_BUG)

  // 3. Escape quita ?bug= y el foco vuelve a la tarjeta.
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  expect(new URL(page.url()).searchParams.get('bug')).toBeNull()
  const enfocada = await page.evaluate(() => document.activeElement?.textContent ?? '')
  expect(enfocada).toContain(fx.bugConDatos.title)

  // 4. URL directa con ?task=: detalle de la tarea con comentario e historial.
  await page.goto(`/projects/${fx.proyecto.id}/stages/${fx.etapa.id}?task=${fx.tarea.id}`)
  const dialogTarea = page.getByRole('dialog')
  await expect(dialogTarea).toBeVisible()
  await expect(dialogTarea).toContainText(fx.tarea.title)
  await expect(dialogTarea.getByRole('region', { name: 'Comentarios' })).toContainText(
    COMENTARIO_TAREA,
  )
  const historialTarea = dialogTarea.getByRole('region', { name: 'Historial de estados' })
  await expect(historialTarea).toContainText(
    `${TASK_STATUS_LABELS.ready} → ${TASK_STATUS_LABELS.in_dev}`,
  )
  await expect(historialTarea).toContainText(MOTIVO_TAREA)

  // 5. Recargar conserva el detalle.
  await page.reload()
  await expect(page).toHaveURL(
    `/projects/${fx.proyecto.id}/stages/${fx.etapa.id}?task=${fx.tarea.id}`,
  )
  await expect(page.getByRole('dialog')).toContainText(fx.tarea.title)
}

test.describe('tarea 07 — recorrido de bugs y detalle', () => {
  test('escritorio', async ({ page, harness }) => {
    await recorrido(page, false, sembrar(harness))
  })

  test('a 360 px', async ({ page, harness }) => {
    await recorrido(page, true, sembrar(harness))
  })
})
