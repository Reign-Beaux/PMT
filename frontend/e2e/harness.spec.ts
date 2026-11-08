// Pruebas del harness aislado (tarea 20, criterios 4 y 6 del plan 1494a1c4 v1):
// fixture legible por la API real, limpieza tras fallo y verificación por SQL.
import { expect } from '@playwright/test'
import { psql } from './harness/db'
import { API_TARGET } from './harness/env'
import { test } from './harness/fixtures'

let idDelFallo = ''
let falloCompleto = { proyecto: '', etapa: '' }

test.describe.serial('Harness aislado de pruebas', () => {
  test('crea un proyecto fixture y lo lee por la API real en 8090', async ({ harness, request }) => {
    const proyecto = harness.createProject()

    const detalle = await request.get(`${API_TARGET}/api/projects/${proyecto.id}`)
    expect(detalle.status()).toBe(200)
    const body = await detalle.json()
    expect(body.name).toBe(proyecto.name)

    const lista = await request.get(`${API_TARGET}/api/projects`)
    expect(lista.status()).toBe(200)
    const proyectos = await lista.json()
    expect(proyectos.some((p: { id: string }) => p.id === proyecto.id)).toBe(true)
  })

  test.fail('el teardown limpia sus fixtures aunque la prueba falle', async ({ harness }) => {
    const proyecto = harness.createProject()
    idDelFallo = proyecto.id
    throw new Error('fallo a propósito para probar la limpieza')
  })

  test('el proyecto de la prueba fallida ya no está en la base', async () => {
    expect(idDelFallo).not.toBe('')
    expect(psql(`select count(*) from projects where id = '${idDelFallo}'`)).toBe('0')
  })

  test.fail('el teardown limpia proyecto, etapa y tarea aunque la prueba falle', async ({ harness }) => {
    const proyecto = harness.createProject({ status: 'working', productVersion: '1.0.0' })
    const etapa = harness.createStage(proyecto.id, { position: 1, status: 'in_dev' })
    harness.createTask(etapa.id, { status: 'backlog' })
    falloCompleto = { proyecto: proyecto.id, etapa: etapa.id }
    throw new Error('fallo a propósito para probar la limpieza de proyecto+etapa+tarea')
  })

  test('tras el fallo completo no quedan tasks, stages ni projects de ese id', async () => {
    expect(falloCompleto.etapa).not.toBe('')
    expect(psql(`select count(*) from tasks where stage_id = '${falloCompleto.etapa}'`)).toBe('0')
    expect(psql(`select count(*) from stages where id = '${falloCompleto.etapa}'`)).toBe('0')
    expect(psql(`select count(*) from projects where id = '${falloCompleto.proyecto}'`)).toBe('0')
  })
})
