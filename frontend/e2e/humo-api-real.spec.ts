// Humo con API real (tarea 21, criterios 7 y 8 del plan 1494a1c4 v1):
// el shell abre en 5173 y GET /api/projects viaja desde el navegador por el
// proxy de Vite hasta el backend de pruebas en 8090/pmt_frontend_test.
// Sin page.route ni interceptación: solo datos reales del harness.
import { expect } from '@playwright/test'
import { test } from './harness/fixtures'

test('humo: el shell abre y GET /api/projects real por el proxy de 5173', async ({
  page,
  harness,
}) => {
  const proyecto = harness.createProject()

  await page.goto('/')
  await expect(page.getByRole('link', { name: 'PMT' })).toBeVisible()

  const respuesta = await page.evaluate(async () => {
    const res = await fetch('/api/projects', { headers: { Accept: 'application/json' } })
    return {
      status: res.status,
      body: (await res.json()) as { id: string; name: string }[],
    }
  })

  expect(respuesta.status).toBe(200)
  expect(Array.isArray(respuesta.body)).toBe(true)
  const encontrado = respuesta.body.find((p) => p.id === proyecto.id)
  expect(encontrado?.name).toBe(proyecto.name)
})
