import { expect } from '@playwright/test'
import { test } from './harness/fixtures'

test('proyecto, etapa, tarea e historial con la API real aislada', async ({ page, harness }) => {
  const project = harness.createProject({ status: 'working', productVersion: '1.2.0' })
  const stage = harness.createStage(project.id, { position: 1, status: 'in_dev' })
  const task = harness.createTask(stage.id, { status: 'in_dev' })
  harness.createComment('task', task.id, 'Evidencia de la prueba de integración')
  harness.createTaskStatusChange(task.id, 'ready', 'in_dev', 'Inicio del trabajo')
  const bug = harness.createBug(project.id, { status: 'reported' })

  await page.goto(`/projects/${project.id}`)
  await expect(page.getByRole('heading', { level: 1, name: project.name })).toBeVisible()
  await page.getByRole('navigation', { name: 'Secciones del proyecto' }).getByRole('link', { name: /Bugs/ }).click()
  await expect(page.getByRole('button', { name: bug.title, exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { level: 1, name: project.name })).toBeVisible()
  await page.getByRole('navigation', { name: 'Secciones del proyecto' }).getByRole('link', { name: /Etapas/ }).click()
  await page.getByRole('link', { name: new RegExp(stage.name) }).first().click()
  await page.getByRole('button', { name: task.title, exact: true }).click()
  const dialog = page.getByRole('dialog', { name: task.title })
  await expect(dialog.getByText('Evidencia de la prueba de integración')).toBeVisible()
  await dialog.getByRole('tab', { name: 'Historial' }).click()
  await expect(dialog.getByRole('listitem')).toContainText('Lista → En desarrollo')
  await expect(dialog.getByText('Inicio del trabajo')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('button', { name: task.title, exact: true })).toBeFocused()
})
