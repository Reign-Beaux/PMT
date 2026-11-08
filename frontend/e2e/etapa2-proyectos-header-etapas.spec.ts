// Regresión QA de las tareas 01–03 de la etapa 2 (criterios "QA en pantalla"
// del plan 69dd00e2 v1): lista de proyectos, encabezado del proyecto con su
// nombre en el shell y 404, y lista de etapas con avance verificado contra
// las tareas de la base. API real (backend de pruebas en 8090, base
// pmt_frontend_test). Siembra y limpia sus propios fixtures por SQL con el
// cliente del proyecto (docker exec postgresql psql), como en la etapa 1.
// Si la instancia de pruebas no está levantada, los tests se omiten.
import { execFileSync } from 'node:child_process'
import { expect, test, type Page } from '@playwright/test'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'

const TL = 'qa-r2-tl'
const PROYECTO = 'caca0000-0000-4000-8000-000000000001'
const PROYECTO_LARGO = 'caca0000-0000-4000-8000-000000000002'
const ETAPA_A = 'caca0000-0000-4000-8000-000000000011' // position 1, con avance
const ETAPA_B = 'caca0000-0000-4000-8000-000000000012' // position 2, sin tareas
const ETAPA_C = 'caca0000-0000-4000-8000-000000000013' // position 3, canceladas
const INEXISTENTE = '00000000-0000-4000-8000-0000000000ff'

const NOMBRE_P1 = 'Proyecto QA etapa2'
const NOMBRE_P2 = `Proyecto de prueba con nombre larguísimo ${'tokenlarguisimo'.repeat(12)} fin`
const DESCRIPCION_P2 = `Descripción con <b>etiqueta</b> literal ${'palabragrande'.repeat(20)}\nsalto de línea final`

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
  for (const etapa of [ETAPA_A, ETAPA_B, ETAPA_C]) {
    psql(`DELETE FROM tasks WHERE stage_id = '${etapa}'`)
  }
  psql(`DELETE FROM stages WHERE project_id IN ('${PROYECTO}', '${PROYECTO_LARGO}')`)
  psql(`DELETE FROM projects WHERE id IN ('${PROYECTO}', '${PROYECTO_LARGO}')`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-r2-%'`)
}

/** done/cancelled/total por etapa desde la base, como los calcula la UI. */
function avanceDesdeLaBase(etapaId: string): { done: number; cancelled: number; total: number } {
  const filas = psql(
    `SELECT status, count(*) FROM tasks WHERE stage_id = '${etapaId}' AND retired_at IS NULL GROUP BY status`,
  )
  const counts: Record<string, number> = {}
  for (const fila of filas.split('\n').filter(Boolean)) {
    const [status, count] = fila.split('|')
    counts[status] = Number(count)
  }
  const total = Object.values(counts).reduce((a, b) => a + b, 0)
  return { done: counts['done'] ?? 0, cancelled: counts['cancelled'] ?? 0, total }
}

function etiquetaAvance(avance: { done: number; cancelled: number; total: number }): string {
  const denom = avance.total - avance.cancelled
  const pct = denom <= 0 ? 0 : Math.min(100, Math.round((100 * avance.done) / denom))
  return `${pct} % · ${avance.done}/${denom}`
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
    `INSERT INTO agents (id, name, profile, token_hash) VALUES
      ('caca0000-0000-4000-8000-0000000000a1', '${TL}', 'TL', '\\x3a')`,
  )
  psql(
    `INSERT INTO projects (id, name, description, status, product_version) VALUES
      ('${PROYECTO}', '${NOMBRE_P1}', 'Fixture de regresión QA etapa 2', 'working', '2.3.4'),
      ('${PROYECTO_LARGO}', '${NOMBRE_P2}', E'${DESCRIPCION_P2}', 'planning', '')`,
  )
  // Se insertan desordenadas (B antes que A) para probar el orden por position.
  psql(
    `INSERT INTO stages (id, project_id, name, description, position, status, lead) VALUES
      ('${ETAPA_B}', '${PROYECTO}', 'Etapa B sin tareas', 'Sin lead ni tareas', 2, 'planning', NULL),
      ('${ETAPA_A}', '${PROYECTO}', 'Etapa A con avance', 'Con lead y tareas', 1, 'in_dev', '${TL}'),
      ('${ETAPA_C}', '${PROYECTO}', 'Etapa C canceladas', 'Todo cancelado', 3, 'done', '${TL}')`,
  )
  psql(
    `INSERT INTO tasks (stage_id, title, status) VALUES
      ('${ETAPA_A}', 'T1 done', 'done'),
      ('${ETAPA_A}', 'T2 done', 'done'),
      ('${ETAPA_A}', 'T3 cancelada', 'cancelled'),
      ('${ETAPA_A}', 'T4 en dev', 'in_dev'),
      ('${ETAPA_C}', 'T5 cancelada', 'cancelled'),
      ('${ETAPA_C}', 'T6 cancelada', 'cancelled')`,
  )
})

test.afterAll(() => {
  if (!apiDisponible) return
  limpiar()
})

test.beforeEach(() => {
  test.skip(!apiDisponible, 'Instancia de pruebas (8090/pmt_frontend_test) no disponible')
})

function tarjeta(page: Page, texto: string) {
  return page.getByRole('button').filter({ hasText: texto })
}

test.describe('tarea 01 — lista de proyectos en /', () => {
  test('lista los proyectos de la base con su estado y versión', async ({ page }) => {
    // La instantánea de la API se toma antes de cargar la página: otros
    // workers siembran y limpian sus propios fixtures en paralelo sobre la
    // misma base.
    const proyectos = (await (await fetch(`${API}/api/projects`)).json()) as {
      id: string
      name: string
    }[]
    await page.goto('/')
    await expect(page.getByRole('heading', { level: 1, name: 'Proyectos' })).toBeVisible()

    // Todo proyecto que devuelve la API real aparece en la pantalla; solo se
    // tolera su ausencia si otro worker ya lo borró de la base.
    for (const proyecto of proyectos) {
      const tarjetaProyecto = page.getByRole('button').filter({ hasText: proyecto.name })
      if (await tarjetaProyecto.first().isVisible()) continue
      const respuesta = await fetch(`${API}/api/projects/${proyecto.id}`)
      expect(respuesta.status, `falta la tarjeta de "${proyecto.name}"`).toBe(404)
    }

    const p1 = tarjeta(page, NOMBRE_P1)
    await expect(p1).toContainText('Versión 2.3.4')
    await expect(p1).toContainText('En operación')

    // P2: sin product_version no se pinta "Versión"; estado en español.
    const p2 = tarjeta(page, 'nombre larguísimo')
    await expect(p2).toContainText('Planeación')
    await expect(p2).not.toContainText('Versión')
  })

  test('abrir un proyecto con mouse lleva a su ruta', async ({ page }) => {
    await page.goto('/')
    await tarjeta(page, NOMBRE_P1).click()
    await expect(page).toHaveURL(`/projects/${PROYECTO}`)
    await expect(page.getByRole('heading', { level: 1, name: NOMBRE_P1 })).toBeVisible()
  })

  test('abrir un proyecto con teclado lleva a su ruta', async ({ page }) => {
    await page.goto('/')
    await tarjeta(page, NOMBRE_P1).focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(`/projects/${PROYECTO}`)
  })

  test('a 360 px no hay scroll horizontal global', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto('/')
    await expect(tarjeta(page, NOMBRE_P1)).toBeVisible()
    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
    )
    expect(scroll).toBe(true)
  })
})

test.describe('tarea 02 — encabezado del proyecto', () => {
  test('muestra nombre, estado, versión y descripción; el shell muestra el nombre', async ({ page }) => {
    await page.goto(`/projects/${PROYECTO}`)
    await expect(page.getByRole('heading', { level: 1, name: NOMBRE_P1 })).toBeVisible()
    await expect(page.getByText('En operación')).toBeVisible()
    await expect(page.getByText('Versión 2.3.4')).toBeVisible()
    await expect(page.getByText('Fixture de regresión QA etapa 2')).toBeVisible()
    // El contexto del shell (primer header; el del proyecto va dentro de main).
    await expect(page.locator('header').first().getByText(NOMBRE_P1)).toBeVisible()
  })

  test('el encabezado y el nombre en el shell persisten en /stages/:stageId y /bugs', async ({ page }) => {
    await page.goto(`/projects/${PROYECTO}/stages/${ETAPA_A}`)
    await expect(page.getByRole('heading', { level: 1, name: NOMBRE_P1 })).toBeVisible()
    await expect(page.locator('header').first().getByText(NOMBRE_P1)).toBeVisible()
    // Desde el grupo 2 la vista pinta h2 con el nombre de la etapa (tareas 04/05).
    await expect(page.getByRole('heading', { level: 2, name: 'Etapa A con avance' })).toBeVisible()

    await page.goto(`/projects/${PROYECTO}/bugs`)
    await expect(page.getByRole('heading', { level: 1, name: NOMBRE_P1 })).toBeVisible()
    await expect(page.locator('header').first().getByText(NOMBRE_P1)).toBeVisible()
    // Desde la etapa 3 la vista de bugs pinta h2 (como la vista de etapa), ya no h1.
    await expect(page.getByRole('heading', { level: 2, name: 'Bugs', exact: true })).toBeVisible()
  })

  test('un id inexistente muestra "Proyecto no encontrado" con enlace a /', async ({ page }) => {
    await page.goto(`/projects/${INEXISTENTE}`)
    await expect(page.getByText('Proyecto no encontrado')).toBeVisible()
    const enlace = page.getByRole('link', { name: 'Volver a proyectos' })
    await expect(enlace).toHaveAttribute('href', '/')
  })

  test('nombre y descripción largos no desbordan a 360 px y la HTML se ve literal', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto(`/projects/${PROYECTO_LARGO}`)
    await expect(page.getByRole('heading', { level: 1, name: NOMBRE_P2 })).toBeVisible()
    // La descripción con etiquetas se muestra como texto: no existe <b>.
    await expect(page.locator('b')).toHaveCount(0)
    await expect(page.getByText('<b>etiqueta</b> literal', { exact: false })).toBeVisible()
    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
    )
    expect(scroll).toBe(true)
  })
})

test.describe('tarea 03 — lista de etapas con avance', () => {
  test('las etapas aparecen en orden por position con estado, TL y el avance de la base', async ({ page }) => {
    await page.goto(`/projects/${PROYECTO}`)
    await expect(page.getByRole('heading', { level: 2, name: 'Etapas' })).toBeVisible()

    const tarjetas = page.locator('ul button')
    await expect(tarjetas).toHaveCount(3)
    // Orden por position (B se insertó antes que A en la base).
    await expect(tarjetas.nth(0)).toContainText('Etapa A con avance')
    await expect(tarjetas.nth(1)).toContainText('Etapa B sin tareas')
    await expect(tarjetas.nth(2)).toContainText('Etapa C canceladas')

    await expect(tarjetas.nth(0)).toContainText('En desarrollo')
    await expect(tarjetas.nth(1)).toContainText('Planeación')
    await expect(tarjetas.nth(2)).toContainText('Terminada')

    // TL solo donde hay lead.
    await expect(tarjetas.nth(0)).toContainText(`TL: ${TL}`)
    await expect(tarjetas.nth(1)).not.toContainText('TL:')

    // El avance mostrado coincide con las tareas guardadas en la base.
    for (const [i, etapa] of [ETAPA_A, ETAPA_B, ETAPA_C].entries()) {
      const avance = avanceDesdeLaBase(etapa)
      await expect(tarjetas.nth(i)).toContainText(etiquetaAvance(avance))
      const barra = tarjetas.nth(i).getByRole('progressbar', { name: 'Avance de la etapa' })
      const denom = avance.total - avance.cancelled
      const pct = denom <= 0 ? 0 : Math.min(100, Math.round((100 * avance.done) / denom))
      await expect(barra).toHaveAttribute('aria-valuenow', String(pct))
    }
    // Casos concretos de la semilla: A = 67 % (2/3), canceladas fuera; B y C = 0 %.
    await expect(tarjetas.nth(0)).toContainText('67 % · 2/3')
    await expect(tarjetas.nth(1)).toContainText('0 % · 0/0')
    await expect(tarjetas.nth(2)).toContainText('0 % · 0/0')
  })

  test('navegar a una etapa con teclado lleva a su ruta', async ({ page }) => {
    await page.goto(`/projects/${PROYECTO}`)
    await tarjeta(page, 'Etapa A con avance').focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(`/projects/${PROYECTO}/stages/${ETAPA_A}`)
  })

  test('a 360 px no hay scroll horizontal global', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto(`/projects/${PROYECTO}`)
    await expect(tarjeta(page, 'Etapa A con avance')).toBeVisible()
    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
    )
    expect(scroll).toBe(true)
  })
})
