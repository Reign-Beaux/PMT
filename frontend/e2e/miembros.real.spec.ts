import { expect, type Page } from '@playwright/test'
import { test } from './harness/fixtures'

const tabs = (page: Page) => page.getByRole('navigation', { name: 'Secciones del proyecto' })

test('el owner comparte como viewer y como editor; el otro ve o deja de ver el proyecto en vivo', async ({ page, browser, harness }) => {
  const project = harness.createProject({ status: 'working', productVersion: '1.0.0' })
  const ana = await harness.createUser('ana')

  // Owner: su rol y la pestaña de miembros con el formulario.
  await page.goto(`/projects/${project.id}`)
  await expect(page.getByRole('heading', { level: 1, name: project.name })).toBeVisible()
  await expect(page.locator('header').getByText('Owner')).toBeVisible()
  await tabs(page).getByRole('link', { name: /Miembros/ }).click()
  const form = page.getByRole('form', { name: 'Agregar miembro' })

  // Errores del backend con su mensaje.
  await form.getByLabel('Username').fill(`${ana.username}-no-existe`)
  await form.getByRole('button', { name: 'Agregar' }).click()
  await expect(page.getByRole('alert')).not.toBeEmpty()

  // Antes de compartir, ana no ve el proyecto.
  const other = await browser.newContext({ storageState: ana.storageState })
  const anaPage = await other.newPage()
  try {
    await anaPage.goto('/projects')
    await expect(anaPage.getByRole('heading', { level: 1, name: 'Proyectos' })).toBeVisible()
    await expect(anaPage.getByRole('link', { name: new RegExp(project.name) })).toHaveCount(0)

    // Compartir como viewer.
    await form.getByLabel('Username').fill(ana.username)
    await form.getByLabel('Rol').selectOption('viewer')
    await form.getByRole('button', { name: 'Agregar' }).click()
    await expect(page.getByRole('status').filter({ hasText: `${ana.username} ahora es viewer` })).toBeVisible()
    await expect(page.getByLabel(`Rol de ${ana.username}`)).toHaveValue('viewer')

    // Ya existe para ana, en vivo, como viewer y sin botones de escritura.
    await expect(anaPage.getByRole('link', { name: new RegExp(project.name) })).toBeVisible()
    await anaPage.getByRole('link', { name: new RegExp(project.name) }).click()
    await expect(anaPage.locator('header').getByText('Viewer')).toBeVisible()
    await tabs(anaPage).getByRole('link', { name: /Miembros/ }).click()
    await expect(anaPage.getByRole('cell', { name: ana.username })).toBeVisible()
    await expect(anaPage.getByRole('form', { name: 'Agregar miembro' })).toHaveCount(0)
    await expect(anaPage.getByRole('main').getByRole('button')).toHaveCount(0)
    await expect(anaPage.getByRole('main').getByRole('combobox')).toHaveCount(0)

    // Cambiar a editor: ana lo ve sin recargar.
    await page.getByLabel(`Rol de ${ana.username}`).selectOption('editor')
    await expect(page.getByRole('status').filter({ hasText: `${ana.username} ahora es editor` })).toBeVisible()
    await expect(anaPage.locator('header').getByText('Editor')).toBeVisible()

    // Quitarla, con confirmación: ana vuelve a la lista con un aviso.
    page.once('dialog', dialog => void dialog.accept())
    await page.getByRole('button', { name: `Quitar a ${ana.username}` }).click()
    await expect(page.getByRole('cell', { name: ana.username })).toHaveCount(0)
    await expect(anaPage).toHaveURL(/\/projects$/)
    await expect(anaPage.getByText(`Ya no tienes acceso al proyecto «${project.name}».`)).toBeVisible()
    await expect(anaPage.getByRole('link', { name: new RegExp(project.name) })).toHaveCount(0)
  } finally {
    await other.close()
  }
})

test('el owner libera una etapa y un bug tomados con motivo; los demás no ven el botón', async ({ page, browser, harness }) => {
  const project = harness.createProject({ status: 'working', productVersion: '1.0.0' })
  const leo = await harness.createUser('leo')
  const stage = harness.createStage(project.id, { position: 1, status: 'in_dev', takenBy: leo.username })
  harness.createTask(stage.id, { status: 'in_dev' })
  const bug = harness.createBug(project.id, { status: 'in_dev', takenBy: leo.username })

  // leo es editor: ve quién la tiene, pero no el botón Liberar.
  await page.goto(`/projects/${project.id}/members`)
  const form = page.getByRole('form', { name: 'Agregar miembro' })
  await form.getByLabel('Username').fill(leo.username)
  await form.getByRole('button', { name: 'Agregar' }).click()
  await expect(page.getByLabel(`Rol de ${leo.username}`)).toHaveValue('editor')

  const other = await browser.newContext({ storageState: leo.storageState })
  const leoPage = await other.newPage()
  try {
    await leoPage.goto(`/projects/${project.id}/stages/${stage.id}`)
    await expect(leoPage.getByText(`Tomada por ${leo.username}`)).toBeVisible()
    await expect(leoPage.getByRole('button', { name: 'Liberar' })).toHaveCount(0)

    // Owner: la lista muestra quién la tiene; en la etapa, Liberar pide motivo.
    await tabs(page).getByRole('link', { name: /Etapas/ }).click()
    const row = page.getByRole('region', { name: 'Etapas del proyecto' }).getByRole('link', { name: new RegExp(stage.name) })
    await expect(row).toContainText(`Tomada por ${leo.username}`)
    await row.click()
    await page.getByRole('button', { name: 'Liberar' }).click()
    const release = page.getByRole('form', { name: 'Liberar la etapa' })
    await release.getByRole('button', { name: 'Confirmar liberación' }).click()
    await expect(release.getByRole('alert')).toContainText('motivo')
    await release.getByLabel('Motivo').fill('leo está de vacaciones')
    await release.getByRole('button', { name: 'Confirmar liberación' }).click()
    await expect(page.getByText(`Tomada por ${leo.username}`)).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Liberar' })).toHaveCount(0)

    // leo lo ve sin recargar.
    await expect(leoPage.getByText(`Tomada por ${leo.username}`)).toHaveCount(0)

    // Bug: tomado en la lista y en el detalle; el owner lo libera.
    await page.goto(`/projects/${project.id}/bugs`)
    await expect(page.getByRole('row', { name: new RegExp(bug.title) })).toContainText(`Tomado por ${leo.username}`)
    await page.getByRole('button', { name: bug.title, exact: true }).click()
    const dialog = page.getByRole('dialog', { name: bug.title })
    await dialog.getByRole('button', { name: 'Liberar' }).click()
    await dialog.getByLabel('Motivo').fill('reasignar')
    await dialog.getByRole('button', { name: 'Confirmar liberación' }).click()
    await expect(dialog.getByText('Nadie')).toBeVisible()
    await dialog.getByRole('tab', { name: 'Historial' }).click()
    await expect(dialog.getByText(/reasignar/)).toBeVisible()
  } finally {
    await other.close()
  }
})
