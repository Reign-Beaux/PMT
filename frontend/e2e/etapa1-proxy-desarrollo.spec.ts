// Regresión QA (tarea 19, criterio 3 del plan 1494a1c4 v1; robustecida por la
// tarea 08 del plan 73020de2 v1): el proxy de Vite en 5173 reenvía /api al
// backend de pruebas en 8090 y devuelve el mismo JSON.
// No se compara la lista completa de proyectos: otros specs crean y borran
// proyectos en paralelo y las listas pedidas por ambas vías pueden diferir sin
// que el proxy falle. Se compara el recurso propio sembrado por el harness
// (identificador único): GET /api/projects/{id} igual por proxy y directo, y
// el mismo objeto presente en ambas listas. Sin page.route.
import { expect } from '@playwright/test'
import { API_TARGET } from './harness/env'
import { test } from './harness/fixtures'

test('regresión proxy: GET /api/projects/{id} y la lista coinciden con el backend en 8090', async ({
  page,
  harness,
  request,
}) => {
  const proyecto = harness.createProject()
  const idCodificado = encodeURIComponent(proyecto.id)

  await page.goto('/')

  // El recurso propio por el proxy (fetch desde el navegador) y directo.
  const detalleProxy = await page.evaluate(async (id: string) => {
    const res = await fetch(`/api/projects/${id}`, { headers: { Accept: 'application/json' } })
    return { status: res.status, body: (await res.json()) as unknown }
  }, idCodificado)
  const detalleDirecto = await request.get(`${API_TARGET}/api/projects/${idCodificado}`)

  expect(detalleProxy.status, 'el proxy no respondió 200 para el proyecto propio').toBe(200)
  expect(detalleDirecto.status(), 'el backend no respondió 200 para el proyecto propio').toBe(200)
  expect(detalleProxy.body, 'el cuerpo del proyecto propio difiere entre proxy y directo').toEqual(
    await detalleDirecto.json(),
  )
  expect(detalleProxy.body).toMatchObject({ id: proyecto.id, name: proyecto.name })

  // Las listas por ambas vías: sin compararlas enteras, el proyecto propio
  // debe estar presente en las dos con el mismo objeto.
  const listaProxy = await page.evaluate(async () => {
    const res = await fetch('/api/projects', { headers: { Accept: 'application/json' } })
    return { status: res.status, body: (await res.json()) as unknown }
  })
  const listaDirecto = await request.get(`${API_TARGET}/api/projects`)

  expect(listaProxy.status, 'el proxy no respondió 200 para la lista').toBe(200)
  expect(listaDirecto.status(), 'el backend no respondió 200 para la lista').toBe(200)
  expect(Array.isArray(listaProxy.body), 'la lista del proxy no es un arreglo').toBe(true)
  const directa = (await listaDirecto.json()) as unknown
  expect(Array.isArray(directa), 'la lista directa no es un arreglo').toBe(true)

  const propioProxy = (listaProxy.body as { id: string }[]).find((p) => p.id === proyecto.id)
  const propioDirecto = (directa as { id: string }[]).find((p) => p.id === proyecto.id)
  expect(propioProxy, 'el proyecto propio falta en la lista pedida por proxy').toBeDefined()
  expect(propioDirecto, 'el proyecto propio falta en la lista pedida directo').toBeDefined()
  expect(propioProxy, 'el proyecto propio difiere entre ambas listas').toEqual(propioDirecto)
})
