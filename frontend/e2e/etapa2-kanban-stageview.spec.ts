// Regresión QA de las tareas 04–05 de la etapa 2 (criterios "QA en pantalla"
// del plan f1765b5b v1): tablero kanban de /projects/:id/stages/:stageId con
// las tareas de la base en su columna, assignee, avance (canceladas fuera del
// porcentaje), nombre de etapa en el shell, "Etapa no encontrada" para un
// stageId ajeno, desplazamiento horizontal del tablero a 360 px sin scroll
// global y alcance por teclado de columnas y tarjetas. API real (backend de
// pruebas en 8090, base pmt_frontend_test). Siembra y limpia sus propios
// fixtures por SQL con el cliente del proyecto (docker exec postgresql psql).
// Si la instancia de pruebas no está levantada, los tests se omiten.
import { execFileSync } from 'node:child_process'
import { expect, test, type Page } from '@playwright/test'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'

const TL = 'qa-r3-tl'
const DEV = 'qa-r3-dev'
const PROYECTO = 'dddd0000-0000-4000-8000-000000000001'
const ETAPA = 'dddd0000-0000-4000-8000-000000000011' // con tareas variadas
const ETAPA_CANCELADA = 'dddd0000-0000-4000-8000-000000000012' // todo cancelado
// Etapa real de otro proyecto: existe en la base pero no en el proyecto de la URL.
const PROYECTO_OTRO = 'dddd0000-0000-4000-8000-000000000002'
const ETAPA_AJENA = 'dddd0000-0000-4000-8000-000000000021'

const NOMBRE_P = 'Proyecto QA kanban'
const NOMBRE_ETAPA = 'Etapa kanban QA'

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

// statuses en el mismo orden que COLUMNAS (TASK_COLUMN_ORDER).
const STATUSES = [
  'backlog',
  'ready',
  'in_dev',
  'in_review',
  'changes_requested',
  'in_qa',
  'pending_close',
  'done',
  'cancelled',
]

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
  for (const etapa of [ETAPA, ETAPA_CANCELADA]) {
    psql(`DELETE FROM tasks WHERE stage_id = '${etapa}'`)
  }
  psql(`DELETE FROM stages WHERE project_id IN ('${PROYECTO}', '${PROYECTO_OTRO}')`)
  psql(`DELETE FROM projects WHERE id IN ('${PROYECTO}', '${PROYECTO_OTRO}')`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-r3-%'`)
}

/** Tarjetas de la etapa desde la base: title|status|assignee. */
function tareasDesdeLaBase(etapaId: string): { title: string; status: string; assignee: string | null }[] {
  return psql(
    `SELECT title, status, assignee FROM tasks WHERE stage_id = '${etapaId}' AND retired_at IS NULL`,
  )
    .split('\n')
    .filter(Boolean)
    .map((fila) => {
      const [title, status, assignee] = fila.split('|')
      return { title, status, assignee: assignee === '' ? null : assignee }
    })
}

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
      ('dddd0000-0000-4000-8000-0000000000a1', '${TL}', 'TL', '\\x4a'),
      ('dddd0000-0000-4000-8000-0000000000a2', '${DEV}', 'DevFrontend', '\\x4b')`,
  )
  psql(
    `INSERT INTO projects (id, name, description, status, product_version) VALUES
      ('${PROYECTO}', '${NOMBRE_P}', 'Fixture de regresión QA kanban', 'working', '1.0.0'),
      ('${PROYECTO_OTRO}', 'Proyecto QA ajeno', 'Otro proyecto del fixture', 'working', '1.0.0')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, description, position, status, lead) VALUES
      ('${ETAPA}', '${PROYECTO}', '${NOMBRE_ETAPA}', 'Con tareas variadas', 1, 'in_dev', '${TL}'),
      ('${ETAPA_CANCELADA}', '${PROYECTO}', 'Etapa toda cancelada', 'Sin avance posible', 2, 'done', '${TL}'),
      ('${ETAPA_AJENA}', '${PROYECTO_OTRO}', 'Etapa de otro proyecto', 'No pertenece al proyecto principal', 1, 'in_dev', '${TL}')`,
  )
  psql(
    `INSERT INTO tasks (stage_id, title, status, assignee) VALUES
      ('${ETAPA}', 'T1 backlog', 'backlog', NULL),
      ('${ETAPA}', 'T2 lista', 'ready', '${DEV}'),
      ('${ETAPA}', 'T3 en dev', 'in_dev', '${DEV}'),
      ('${ETAPA}', 'T4 terminada', 'done', NULL),
      ('${ETAPA}', 'T5 terminada', 'done', '${TL}'),
      ('${ETAPA}', 'T6 cancelada', 'cancelled', NULL),
      ('${ETAPA_CANCELADA}', 'T7 cancelada', 'cancelled', NULL),
      ('${ETAPA_CANCELADA}', 'T8 cancelada', 'cancelled', NULL)`,
  )
})

test.afterAll(() => {
  if (!apiDisponible) return
  limpiar()
})

test.beforeEach(() => {
  test.skip(!apiDisponible, 'Instancia de pruebas (8090/pmt_frontend_test) no disponible')
})

async function irAlTablero(page: Page, etapaId = ETAPA) {
  await page.goto(`/projects/${PROYECTO}/stages/${etapaId}`)
  await expect(
    page.getByRole('region', { name: 'Tablero de la etapa' }),
  ).toBeVisible()
}

test.describe('tareas 04/05 — tablero kanban de la etapa', () => {
  test('las nueve columnas en orden con los conteos de las tareas de la base', async ({ page }) => {
    await irAlTablero(page)

    const board = page.getByRole('region', { name: 'Tablero de la etapa' })
    const columnas = board.locator('section[aria-labelledby^="kanban-col-"]')
    await expect(columnas).toHaveCount(9)
    for (const [i, etiqueta] of COLUMNAS.entries()) {
      await expect(columnas.nth(i).locator('h3')).toContainText(etiqueta, { useInnerText: true })
    }

    // Cada tarea de la base está en la columna de su status, con su conteo.
    const tareas = tareasDesdeLaBase(ETAPA)
    const porStatus = new Map<string, number>()
    for (const t of tareas) porStatus.set(t.status, (porStatus.get(t.status) ?? 0) + 1)
    for (const [i, status] of STATUSES.entries()) {
      await expect(columnas.nth(i).locator('h3')).toContainText(
        String(porStatus.get(status) ?? 0),
      )
    }
  })

  test('cada tarjeta muestra su título y assignee de la base o "Sin asignar"', async ({ page }) => {
    await irAlTablero(page)

    const board = page.getByRole('region', { name: 'Tablero de la etapa' })
    const etiquetasPorStatus: Record<string, string> = {
      backlog: 'Backlog', ready: 'Lista', in_dev: 'En desarrollo',
      in_review: 'En revisión', changes_requested: 'Cambios solicitados',
      in_qa: 'En QA', pending_close: 'Pendiente de cierre',
      done: 'Terminada', cancelled: 'Cancelada',
    }
    for (const tarea of tareasDesdeLaBase(ETAPA)) {
      const columna = board.getByRole('region', {
        name: new RegExp(`^${etiquetasPorStatus[tarea.status]}`),
      })
      const tarjeta = columna.getByRole('button').filter({ hasText: tarea.title })
      await expect(tarjeta).toBeVisible()
      await expect(tarjeta).toContainText(tarea.assignee ?? 'Sin asignar')
    }
  })

  test('encabezado con nombre, estado y avance cotejado con la base; el shell muestra la etapa', async ({ page }) => {
    await irAlTablero(page)

    const header = page.getByRole('heading', { level: 2, name: NOMBRE_ETAPA }).locator('xpath=ancestor::header')
    await expect(header).toContainText('En desarrollo')

    const avance = avanceDesdeLaBase(ETAPA)
    const denom = avance.total - avance.cancelled
    const pct = Math.min(100, Math.round((100 * avance.done) / denom))
    await expect(header.getByRole('progressbar', { name: 'Avance de la etapa' })).toHaveAttribute(
      'aria-valuenow',
      String(pct),
    )
    await expect(header).toContainText(`${pct} % · ${avance.done}/${denom}`)
    // Semilla concreta: 2 done de 5 no canceladas = 40 %.
    await expect(header).toContainText('40 % · 2/5')

    // El contexto del shell muestra el nombre de la etapa.
    await expect(page.locator('header').first()).toContainText(NOMBRE_ETAPA)
  })

  test('etapa con todo cancelado muestra 0 %', async ({ page }) => {
    await irAlTablero(page, ETAPA_CANCELADA)

    const header = page.getByRole('heading', { level: 2, name: 'Etapa toda cancelada' }).locator('xpath=ancestor::header')
    await expect(header.getByRole('progressbar', { name: 'Avance de la etapa' })).toHaveAttribute(
      'aria-valuenow',
      '0',
    )
    await expect(header).toContainText('0 % · 0/0')
  })

  test('click en una tarjeta deja ?task=:id en la URL', async ({ page }) => {
    await irAlTablero(page)

    const board = page.getByRole('region', { name: 'Tablero de la etapa' })
    await board.getByRole('button').filter({ hasText: 'T3 en dev' }).click()
    const search = new URL(page.url()).searchParams.get('task')
    const id = psql(`SELECT id FROM tasks WHERE stage_id = '${ETAPA}' AND title = 'T3 en dev'`)
    expect(search).toBe(id)
    // La ruta no cambia, solo el query.
    expect(page.url()).toContain(`/projects/${PROYECTO}/stages/${ETAPA}`)
  })

  test('un stageId ajeno al proyecto muestra "Etapa no encontrada" con enlace', async ({ page }) => {
    // La etapa existe en la base pero pertenece a otro proyecto.
    await page.goto(`/projects/${PROYECTO}/stages/${ETAPA_AJENA}`)

    await expect(page.getByText('Etapa no encontrada', { exact: true })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Volver al proyecto' })).toHaveAttribute(
      'href',
      `/projects/${PROYECTO}`,
    )
    await expect(
      page.getByRole('region', { name: 'Tablero de la etapa' }),
    ).toHaveCount(0)
  })

  test('un stageId inexistente en la base muestra "Etapa no encontrada" sin Reintentar', async ({ page }) => {
    // La etapa no existe: GET /api/stages/:id/tasks responde 404 y debe caer
    // en la rama propia de no encontrada, no en el ErrorState con reintento.
    const INEXISTENTE = 'dddd0000-0000-4000-8000-ffffffffffff'
    await page.goto(`/projects/${PROYECTO}/stages/${INEXISTENTE}`)

    await expect(page.getByText('Etapa no encontrada', { exact: true })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Volver al proyecto' })).toHaveAttribute(
      'href',
      `/projects/${PROYECTO}`,
    )
    await expect(page.getByRole('button', { name: 'Reintentar' })).toHaveCount(0)
    await expect(
      page.getByRole('region', { name: 'Tablero de la etapa' }),
    ).toHaveCount(0)
  })

  test('a 360 px el tablero desplaza en horizontal sin scroll global', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await irAlTablero(page)

    const medidas = await page.evaluate(() => {
      const board = document.querySelector('[aria-label="Tablero de la etapa"]')
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

  test('columnas y tarjetas se alcanzan con teclado y Enter activa la tarjeta', async ({ page }) => {
    await irAlTablero(page)

    const board = page.getByRole('region', { name: 'Tablero de la etapa' })
    await board.focus()

    // Tabulando desde el tablero se recorren las tarjetas de todas las
    // columnas en orden del DOM (las tarjetas son <button>).
    const esperadas = ['T1 backlog', 'T2 lista', 'T3 en dev', 'T4 terminada', 'T5 terminada', 'T6 cancelada']
    const alcanzadas: string[] = []
    for (let i = 0; i < 60 && alcanzadas.length < esperadas.length; i += 1) {
      await page.keyboard.press('Tab')
      const texto = await page.evaluate(
        () => document.activeElement?.textContent ?? '',
      )
      const siguiente = esperadas.find(
        (t) => texto.includes(t) && !alcanzadas.includes(t),
      )
      if (siguiente !== undefined) alcanzadas.push(siguiente)
    }
    expect(alcanzadas).toEqual(esperadas)

    // Enter sobre la tarjeta enfocada (T6 cancelada) escribe ?task=:id.
    await page.keyboard.press('Enter')
    const search = new URL(page.url()).searchParams.get('task')
    const id = psql(`SELECT id FROM tasks WHERE stage_id = '${ETAPA}' AND title = 'T6 cancelada'`)
    expect(search).toBe(id)
  })
})
