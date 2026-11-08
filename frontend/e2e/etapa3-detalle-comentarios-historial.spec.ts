// Regresión QA de las tareas 03/05/06 de la etapa 3 (criterios "QA en pantalla"
// del plan ecab7e90 v1): detalle de tarea desde el kanban de la etapa cotejado
// con la base, recarga con ?task=, foco devuelto a la tarjeta con Escape y
// diálogo usable a 360 px; comentarios reales en orden (del más antiguo al más
// reciente) en el detalle de tarea y de bug, con "Sin comentarios" cuando no
// hay; historial de estados coincidente con la base (estados, autor, motivo)
// en orden, con etiquetas de tarea y de bug, motivo vacío omitido y "Sin
// cambios de estado" cuando no hay. API real (backend de pruebas en 8090,
// base pmt_frontend_test). Siembra y limpia sus propios fixtures por SQL con
// el cliente del proyecto (docker exec postgresql psql); las tablas de cambios
// son inmutables y la limpieza desactiva sus triggers solo durante el borrado.
// Si la instancia de pruebas no está levantada, los tests se omiten.
import { execFileSync } from 'node:child_process'
import { expect, test, type Page } from '@playwright/test'
import { BUG_STATUS_LABELS, TASK_STATUS_LABELS } from '../src/domain/status'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'

const PROYECTO = 'ffff0000-0000-4000-8000-000000000001'
const NOMBRE_P = 'Proyecto QA detalle etapa3'
const ETAPA = 'ffff0000-0000-4000-8000-000000000002'
const TAREA_CON_DATOS = 'ffff0000-0000-4000-8000-000000000003'
const TAREA_VACIA = 'ffff0000-0000-4000-8000-000000000004'
const BUG_CON_DATOS = 'ffff0000-0000-4000-8000-000000000005'
const BUG_VACIO = 'ffff0000-0000-4000-8000-000000000006'

const DESCRIPCION_HTML = '<b>no negrita</b> & <i>literal</i>\nsegunda línea'
const COMENTARIO_HTML = '<script>no</script> & salto.\nSegunda línea.'
const MOTIVO_TAREA = 'Arranque de la etapa'
const MOTIVO_BUG = 'Reproducido en la instancia de pruebas'

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

const TABLAS_INMUTABLES = [
  'comments',
  'task_status_changes',
  'bug_status_changes',
  'stage_status_changes',
  'project_status_changes',
]

function inmutabilidad(modo: 'DISABLE' | 'ENABLE'): string {
  return TABLAS_INMUTABLES.map(
    (tabla) => `ALTER TABLE ${tabla} ${modo} TRIGGER ${tabla}_immutable;`,
  ).join('\n')
}

function limpiar(): void {
  psql(`
BEGIN;
${inmutabilidad('DISABLE')}
DELETE FROM comments WHERE item_id IN ('${TAREA_CON_DATOS}', '${TAREA_VACIA}', '${BUG_CON_DATOS}', '${BUG_VACIO}');
DELETE FROM task_status_changes WHERE task_id IN ('${TAREA_CON_DATOS}', '${TAREA_VACIA}');
DELETE FROM bug_status_changes WHERE bug_id IN ('${BUG_CON_DATOS}', '${BUG_VACIO}');
DELETE FROM stage_status_changes WHERE stage_id = '${ETAPA}';
DELETE FROM project_status_changes WHERE project_id = '${PROYECTO}';
${inmutabilidad('ENABLE')}
COMMIT;`)
  psql(`DELETE FROM bugs WHERE project_id = '${PROYECTO}'`)
  psql(`DELETE FROM tasks WHERE stage_id = '${ETAPA}'`)
  psql(`DELETE FROM stage_members WHERE stage_id = '${ETAPA}'`)
  psql(`DELETE FROM stages WHERE project_id = '${PROYECTO}'`)
  psql(`DELETE FROM projects WHERE id = '${PROYECTO}'`)
  psql(`DELETE FROM agents WHERE name LIKE 'qa-r5-%'`)
}

/** Fecha de la base formateada como hace formatDate (Intl es, medium/short). */
function fecha(sql: string): string {
  const epoch = Number(psql(sql))
  return new Intl.DateTimeFormat('es', { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(epoch * 1000),
  )
}

interface Comentario {
  body: string
  actor: string
  created: string
}

/** Comentarios de la base en el orden que la UI debe mostrarlos (por fecha). */
function comentariosDe(kind: 'task' | 'bug', id: string): Comentario[] {
  // Los saltos de línea del body viajan escapados para no romper el parseo.
  return psql(
    `SELECT replace(body, E'\\n', '\\n'), actor_name, actor_profile, extract(epoch from created_at)
     FROM comments WHERE item_kind = '${kind}' AND item_id = '${id}' ORDER BY created_at`,
  )
    .split('\n')
    .filter(Boolean)
    .map((fila) => {
      const [body, actorName, actorProfile, epoch] = fila.split('|')
      return {
        body: body.replaceAll('\\n', '\n'),
        actor: actorProfile === '' ? actorName : `${actorName} (${actorProfile})`,
        created: new Intl.DateTimeFormat('es', {
          dateStyle: 'medium',
          timeStyle: 'short',
        }).format(new Date(Number(epoch) * 1000)),
      }
    })
}

interface Cambio {
  de: string
  a: string
  actor: string
  motivo: string
  fecha: string
}

function historialDe(kind: 'task' | 'bug', id: string): Cambio[] {
  const tabla = kind === 'task' ? 'task_status_changes' : 'bug_status_changes'
  const columna = kind === 'task' ? 'task_id' : 'bug_id'
  const etiquetas = kind === 'task' ? TASK_STATUS_LABELS : BUG_STATUS_LABELS
  return psql(
    `SELECT from_status, to_status, actor_name, actor_profile, reason, extract(epoch from changed_at)
     FROM ${tabla} WHERE ${columna} = '${id}' ORDER BY changed_at`,
  )
    .split('\n')
    .filter(Boolean)
    .map((fila) => {
      const [de, a, actorName, actorProfile, motivo, epoch] = fila.split('|')
      return {
        de: etiquetas[de as keyof typeof etiquetas],
        a: etiquetas[a as keyof typeof etiquetas],
        actor: actorProfile === '' ? actorName : `${actorName} (${actorProfile})`,
        motivo,
        fecha: new Intl.DateTimeFormat('es', {
          dateStyle: 'medium',
          timeStyle: 'short',
        }).format(new Date(Number(epoch) * 1000)),
      }
    })
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
      ('ffff0000-0000-4000-8000-0000000000a1', 'qa-r5-tl', 'TL', '\\x5a'),
      ('ffff0000-0000-4000-8000-0000000000a2', 'qa-r5-dev', 'DevFrontend', '\\x5b')`,
  )
  psql(
    `INSERT INTO projects (id, name, description, status, product_version, created_at, updated_at) VALUES
      ('${PROYECTO}', '${NOMBRE_P}', 'Fixture de regresión QA detalle', 'working', '1.0.0',
       '2026-11-01 09:00:00-06', '2026-11-06 09:00:00-06')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, description, position, status, created_at, updated_at, lead) VALUES
      ('${ETAPA}', '${PROYECTO}', 'Etapa QA detalle', 'Con lead y miembro', 1, 'in_dev',
       '2026-11-01 09:05:00-06', '2026-11-05 11:00:00-06', 'qa-r5-tl')`,
  )
  psql(
    `INSERT INTO stage_members (stage_id, agent, added_by) VALUES
      ('${ETAPA}', 'qa-r5-dev', 'qa-r5-tl')`,
  )
  psql(
    `INSERT INTO tasks (id, stage_id, title, description, status, created_at, updated_at, assignee) VALUES
      ('${TAREA_CON_DATOS}', '${ETAPA}', 'T1 con comentarios', '${DESCRIPCION_HTML.replace(/'/g, "''")}', 'in_dev',
       '2026-11-03 10:00:00-06', '2026-11-04 10:00:00-06', 'qa-r5-dev'),
      ('${TAREA_VACIA}', '${ETAPA}', 'T2 sin comentarios', 'Sin movimientos', 'ready',
       '2026-11-03 10:01:00-06', '2026-11-03 10:01:00-06', NULL)`,
  )
  psql(
    `INSERT INTO bugs (id, project_id, title, description, steps_to_reproduce, product_version, status, reported_by, created_at, updated_at) VALUES
      ('${BUG_CON_DATOS}', '${PROYECTO}', 'B1 con comentarios', 'Descripción B1', 'Pasos B1', '1.2.3', 'confirmed', 'qa-r5-usuario',
       '2026-11-05 08:00:00-06', '2026-11-06 09:00:00-06'),
      ('${BUG_VACIO}', '${PROYECTO}', 'B2 sin comentarios', 'Descripción B2', 'Pasos B2', '1.2.3', 'reported', 'qa-r5-usuario',
       '2026-11-05 09:00:00-06', '2026-11-05 09:00:00-06')`,
  )
  // El comentario más reciente se inserta primero: la UI debe reordenar por fecha.
  psql(
    `INSERT INTO comments (id, item_kind, item_id, body, actor_name, actor_profile, created_at) VALUES
      ('ffff0000-0000-4000-8000-0000000000b2', 'task', '${TAREA_CON_DATOS}',
       'Segundo comentario, más reciente.', 'qa-r5-tl', 'TL', '2026-11-07 09:00:00-06'),
      ('ffff0000-0000-4000-8000-0000000000b1', 'task', '${TAREA_CON_DATOS}',
       '${COMENTARIO_HTML}', 'qa-r5-dev', 'DevFrontend', '2026-11-06 10:00:00-06'),
      ('ffff0000-0000-4000-8000-0000000000b4', 'bug', '${BUG_CON_DATOS}',
       'Segundo comentario del bug.', 'qa-r5-tl', 'TL', '2026-11-06 12:00:00-06'),
      ('ffff0000-0000-4000-8000-0000000000b3', 'bug', '${BUG_CON_DATOS}',
       'Comentario de QA en el bug.', 'qa-r5-qa', 'QA', '2026-11-06 11:00:00-06')`,
  )
  psql(
    `INSERT INTO task_status_changes (id, task_id, from_status, to_status, actor_name, actor_profile, reason, changed_at) VALUES
      ('ffff0000-0000-4000-8000-0000000000c2', '${TAREA_CON_DATOS}', 'ready', 'in_dev',
       'qa-r5-dev', 'DevFrontend', '', '2026-11-04 10:00:00-06'),
      ('ffff0000-0000-4000-8000-0000000000c1', '${TAREA_CON_DATOS}', 'backlog', 'ready',
       'qa-r5-tl', 'TL', '${MOTIVO_TAREA}', '2026-11-03 11:00:00-06')`,
  )
  psql(
    `INSERT INTO bug_status_changes (id, bug_id, from_status, to_status, actor_name, actor_profile, reason, changed_at) VALUES
      ('ffff0000-0000-4000-8000-0000000000d1', '${BUG_CON_DATOS}', 'reported', 'confirmed',
       'qa-r5-qa', 'QA', '${MOTIVO_BUG}', '2026-11-06 09:00:00-06')`,
  )
})

test.afterAll(() => {
  if (!apiDisponible) return
  limpiar()
})

test.beforeEach(() => {
  test.skip(!apiDisponible, 'Instancia de pruebas (8090/pmt_frontend_test) no disponible')
})

const rutaEtapa = `/projects/${PROYECTO}/stages/${ETAPA}`

async function irAlTablero(page: Page) {
  await page.goto(rutaEtapa)
  await expect(page.getByRole('region', { name: 'Tablero de la etapa' })).toBeVisible()
}

function seccion(dialog: ReturnType<Page['getByRole']>, nombre: string) {
  return dialog.getByRole('region', { name: nombre })
}

/** Compara la lista renderizada (ol de items) con los datos de la base. */
async function esperarLista(
  items: ReturnType<Page['getByRole']>,
  esperados: { encabezado: string; fecha: string; texto?: string }[],
) {
  await expect(items).toHaveCount(esperados.length)
  for (const [i, esperado] of esperados.entries()) {
    const item = items.nth(i)
    await expect(item).toContainText(esperado.encabezado)
    await expect(item).toContainText(esperado.fecha)
    if (esperado.texto !== undefined) {
      await expect(item).toContainText(esperado.texto)
    }
  }
}

test.describe('tarea 03 — detalle de tarea', () => {
  test('abrir una tarea desde el kanban muestra sus datos correctos frente a la base', async ({ page }) => {
    await irAlTablero(page)

    const board = page.getByRole('region', { name: 'Tablero de la etapa' })
    await board.getByRole('button').filter({ hasText: 'T1 con comentarios' }).click()
    expect(new URL(page.url()).searchParams.get('task')).toBe(TAREA_CON_DATOS)

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog).toContainText('T1 con comentarios')

    const facts = dialog.locator('dl')
    await expect(facts).toContainText('En desarrollo') // Badge en español
    await expect(facts).toContainText('qa-r5-dev')
    await expect(facts).toContainText(
      fecha(`SELECT extract(epoch from created_at) FROM tasks WHERE id = '${TAREA_CON_DATOS}'`),
    )
    await expect(facts).toContainText(
      fecha(`SELECT extract(epoch from updated_at) FROM tasks WHERE id = '${TAREA_CON_DATOS}'`),
    )

    // Descripción literal: el HTML no se interpreta y se respetan los saltos.
    const descripcion = seccion(dialog, 'Descripción').locator('p')
    await expect(descripcion).toContainText(DESCRIPCION_HTML)
    expect(await descripcion.evaluate((el) => getComputedStyle(el).whiteSpace)).toBe('pre-wrap')
    await expect(dialog.locator('b, i')).toHaveCount(0)
  })

  test('recargar con ?task= abre el detalle y sin assignee muestra "Sin asignar"', async ({ page }) => {
    await page.goto(`${rutaEtapa}?task=${TAREA_VACIA}`)

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog).toContainText('T2 sin comentarios')
    const facts = dialog.locator('dl')
    await expect(facts).toContainText('Lista')
    await expect(facts).toContainText('Sin asignar')
  })

  test('Escape quita ?task= y devuelve el foco a la tarjeta', async ({ page }) => {
    await irAlTablero(page)

    const board = page.getByRole('region', { name: 'Tablero de la etapa' })
    await board.getByRole('button').filter({ hasText: 'T1 con comentarios' }).click()
    await expect(page.getByRole('dialog')).toBeVisible()

    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(new URL(page.url()).searchParams.get('task')).toBeNull()
    const enfocada = await page.evaluate(() => document.activeElement?.textContent ?? '')
    expect(enfocada).toContain('T1 con comentarios')
  })

  test('el diálogo es usable a 360 px, con comentarios e historial visibles', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto(`${rutaEtapa}?task=${TAREA_CON_DATOS}`)

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
    await expect(dialog).toContainText('T1 con comentarios')
    await expect(seccion(dialog, 'Comentarios')).toBeVisible()
    await expect(seccion(dialog, 'Historial de estados')).toBeVisible()
  })
})

test.describe('tarea 05 — comentarios', () => {
  test('los comentarios de la tarea coinciden con la base, del más antiguo al más reciente', async ({ page }) => {
    await page.goto(`${rutaEtapa}?task=${TAREA_CON_DATOS}`)

    const comentarios = comentariosDe('task', TAREA_CON_DATOS)
    expect(comentarios).toHaveLength(2)
    await esperarLista(
      seccion(page.getByRole('dialog'), 'Comentarios').getByRole('listitem'),
      comentarios.map((c) => ({ encabezado: c.actor, fecha: c.created, texto: c.body })),
    )
    // El cuerpo con HTML se muestra como texto literal.
    const items = seccion(page.getByRole('dialog'), 'Comentarios').getByRole('listitem')
    expect(await items.first().locator('b, i, script').count()).toBe(0)
  })

  test('los comentarios del bug coinciden con la base, del más antiguo al más reciente', async ({ page }) => {
    await page.goto(`/projects/${PROYECTO}/bugs?bug=${BUG_CON_DATOS}`)

    const comentarios = comentariosDe('bug', BUG_CON_DATOS)
    expect(comentarios).toHaveLength(2)
    await esperarLista(
      seccion(page.getByRole('dialog'), 'Comentarios').getByRole('listitem'),
      comentarios.map((c) => ({ encabezado: c.actor, fecha: c.created, texto: c.body })),
    )
  })

  test('una tarea y un bug sin comentarios muestran "Sin comentarios"', async ({ page }) => {
    await page.goto(`${rutaEtapa}?task=${TAREA_VACIA}`)
    await expect(seccion(page.getByRole('dialog'), 'Comentarios')).toContainText(
      'Sin comentarios',
    )

    await page.goto(`/projects/${PROYECTO}/bugs?bug=${BUG_VACIO}`)
    await expect(seccion(page.getByRole('dialog'), 'Comentarios')).toContainText(
      'Sin comentarios',
    )
  })
})

test.describe('tarea 06 — historial de estados', () => {
  test('el historial de la tarea coincide con la base, en orden y con el motivo vacío omitido', async ({ page }) => {
    await page.goto(`${rutaEtapa}?task=${TAREA_CON_DATOS}`)

    const cambios = historialDe('task', TAREA_CON_DATOS)
    expect(cambios).toHaveLength(2)
    const dialog = page.getByRole('dialog')
    const items = seccion(dialog, 'Historial de estados').getByRole('listitem')
    await esperarLista(
      items,
      cambios.map((c) => ({ encabezado: `${c.de} → ${c.a}`, fecha: c.fecha })),
    )
    await expect(items.nth(0)).toContainText(cambios[0].actor)
    await expect(items.nth(0)).toContainText(MOTIVO_TAREA)
    await expect(items.nth(1)).toContainText(cambios[1].actor)
    // El cambio con motivo vacío no muestra motivo.
    expect((await items.nth(1).textContent()) ?? '').not.toContain(MOTIVO_TAREA)
  })

  test('el historial del bug usa etiquetas de bug y coincide con la base', async ({ page }) => {
    await page.goto(`/projects/${PROYECTO}/bugs?bug=${BUG_CON_DATOS}`)

    const cambios = historialDe('bug', BUG_CON_DATOS)
    expect(cambios).toHaveLength(1)
    const items = seccion(page.getByRole('dialog'), 'Historial de estados').getByRole('listitem')
    await esperarLista(
      items,
      cambios.map((c) => ({ encabezado: `${c.de} → ${c.a}`, fecha: c.fecha })),
    )
    await expect(items.first()).toContainText('Reportado → Confirmado')
    await expect(items.first()).toContainText(cambios[0].actor)
    await expect(items.first()).toContainText(MOTIVO_BUG)
  })

  test('una tarea y un bug sin cambios muestran "Sin cambios de estado"', async ({ page }) => {
    await page.goto(`${rutaEtapa}?task=${TAREA_VACIA}`)
    await expect(seccion(page.getByRole('dialog'), 'Historial de estados')).toContainText(
      'Sin cambios de estado',
    )

    await page.goto(`/projects/${PROYECTO}/bugs?bug=${BUG_VACIO}`)
    await expect(seccion(page.getByRole('dialog'), 'Historial de estados')).toContainText(
      'Sin cambios de estado',
    )
  })
})
