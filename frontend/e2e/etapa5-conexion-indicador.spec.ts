// Regresión QA de las tareas 01 y 02 de la etapa "Tiempo real e integración"
// (criterios 13, 14 y 16 de QA del plan 1d0244fd v1): la app mantiene un solo
// WebSocket a /api/events al navegar entre vistas, sin sondeo HTTP periódico;
// el header muestra "En vivo" en la lista de proyectos y dentro de un proyecto;
// a 360 px el indicador se ve completo, sin overflow horizontal, y está en un
// role="status" (anuncio a lectores de pantalla).
// El criterio 15 (backend caído → "Reconectando…" → "Sin conexión, datos de
// las HH:MM" a los ~30 s → "En vivo" al volver) NO se cubre aquí: exige
// tumbar y levantar el 8090 en exclusiva y rompería los specs paralelos; lo
// cubre etapa5-tiempo-real.spec.ts, que corre al final en el proyecto
// 'reconexion' de playwright.config.ts.
// Backend de pruebas en 8090 contra pmt_frontend_test; si no está levantado se
// omite. Siembra y limpia sus propios datos (prefijo qa-r6-, reservado de este
// spec) por SQL con el psql del contenedor.
import { execFileSync } from 'node:child_process'
import { expect, test } from '@playwright/test'

const API = process.env.PMT_API_TARGET ?? 'http://127.0.0.1:8090'
const PROJECT_ID = 'd0060000-0000-4000-8000-000000000001'
const STAGE_ID = 'd0060000-0000-4000-8000-000000000002'
const PROJECT_NAME = 'qa-r6-Proyecto eventos'
const STAGE_NAME = 'qa-r6-Etapa uno'

function psql(sql: string): void {
  execFileSync(
    'docker',
    [
      'exec', '-i', 'postgresql',
      'psql', '-U', 'games_test', '-d', 'pmt_frontend_test',
      '-v', 'ON_ERROR_STOP=1', '-qtA', '-c', sql,
    ],
    { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] },
  )
}

function limpiar(): void {
  psql(`DELETE FROM stages WHERE name LIKE 'qa-r6-%'`)
  psql(`DELETE FROM projects WHERE name LIKE 'qa-r6-%'`)
}

let disponible = false

test.beforeAll(async () => {
  try {
    const respuesta = await fetch(`${API}/api/projects`)
    disponible = respuesta.ok
  } catch {
    disponible = false
  }
  if (!disponible) return
  limpiar()
  psql(
    `INSERT INTO projects (id, name, description, status) VALUES ('${PROJECT_ID}', '${PROJECT_NAME}', 'Regresión etapa 5', 'working')`,
  )
  psql(
    `INSERT INTO stages (id, project_id, name, position, status) VALUES ('${STAGE_ID}', '${PROJECT_ID}', '${STAGE_NAME}', 1, 'in_dev')`,
  )
})

test.afterAll(() => {
  if (disponible) limpiar()
})

test.describe('Etapa 5 — conexión de eventos e indicador', () => {
  test('"En vivo" al cargar, un solo WS al navegar y sin sondeo HTTP (criterios 13 y 14)', async ({
    page,
  }) => {
    test.skip(!disponible, 'backend de pruebas no disponible en 8090')

    const socketsEvents: { cerrado: boolean }[] = []
    const framesEvento: number[] = []
    page.on('websocket', (ws) => {
      if (!ws.url().includes('/api/events')) return
      const registro = { cerrado: false }
      socketsEvents.push(registro)
      ws.on('close', () => {
        registro.cerrado = true
      })
      // Con el refresco por evento (tarea 03 de la etapa 5), un invalidate o
      // resync recibido justifica peticiones HTTP: se anota cuándo llegó.
      ws.on('framereceived', (evento) => {
        const texto = String(evento.payload)
        if (texto.includes('"invalidate"') || texto.includes('"resync"')) {
          framesEvento.push(Date.now())
        }
      })
    })
    const peticionesApi: { url: string; t: number }[] = []
    page.on('request', (req) => {
      const url = req.url()
      if (url.includes('/api') && !url.includes('/api/events')) peticionesApi.push({ url, t: Date.now() })
    })

    await page.goto('/')

    // Criterio 14: "En vivo" en la lista de proyectos…
    await expect(page.getByRole('banner').getByRole('status')).toHaveText('En vivo', { timeout: 10_000 })

    // StrictMode monta y desmonta el provider: puede crear un socket que en
    //seguida se cierra; solo debe quedar uno abierto.
    await page.waitForTimeout(2_000)
    await page.getByRole('button', { name: new RegExp(PROJECT_NAME) }).click()
    await page.waitForURL(`**/projects/${PROJECT_ID}`)

    // …y dentro del proyecto.
    await expect(page.getByRole('banner').getByRole('status')).toHaveText('En vivo')

    // Criterio 13: navegar a la etapa y volver no crea sockets nuevos.
    const alCargar = socketsEvents.length
    await page.getByRole('button', { name: new RegExp(STAGE_NAME) }).first().click()
    await page.waitForURL('**/stages/**')
    await page.goBack()
    await page.waitForURL(`**/projects/${PROJECT_ID}`)
    await expect(page.getByRole('banner').getByRole('status')).toHaveText('En vivo')
    expect(socketsEvents.length).toBe(alCargar)
    expect(socketsEvents.filter((s) => !s.cerrado)).toHaveLength(1)

    // Criterio 13: sin peticiones HTTP periódicas en reposo. Los specs que
    // corren en paralelo publican cambios y cada invalidate recibido refresca
    // las vistas montadas (comportamiento correcto desde la tarea 03): toda
    // petición en reposo debe seguir a un frame recibido poco antes; sin
    // eventos no debe haber ninguna.
    const enReposo = peticionesApi.length
    await page.waitForTimeout(6_000)
    const sinEventoPrevio = peticionesApi
      .slice(enReposo)
      .filter((peticion) => !framesEvento.some((t) => t <= peticion.t && peticion.t - t <= 2_500))
    expect(sinEventoPrevio.map((peticion) => peticion.url)).toHaveLength(0)
  })

  test('indicador completo y anunciado a 360 px (criterio 16)', async ({ page }) => {
    test.skip(!disponible, 'backend de pruebas no disponible en 8090')
    await page.setViewportSize({ width: 360, height: 640 })
    await page.goto('/')

    const indicador = page.getByRole('banner').getByRole('status')
    await expect(indicador).toHaveText('En vivo', { timeout: 10_000 })

    const box = await indicador.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.x).toBeGreaterThanOrEqual(0)
    expect(box!.x + box!.width).toBeLessThanOrEqual(360.5)
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(360)

    // role="status" implica aria-live="polite": los cambios se anuncian sin
    // mover el foco. El indicador no es enfocable, conserva el mismo nodo al
    // navegar (no se reanuncia por remontaje) y no roba el foco.
    await expect(indicador).toHaveAttribute('role', 'status')
    // Clic en un elemento no interactivo: con specs en paralelo la lista de
    // '/' tiene tarjetas de otros specs bajo cualquier coordenada fija.
    await page.getByRole('heading', { name: 'Proyectos' }).click()
    await page.waitForTimeout(2_000)
    expect(await page.evaluate(() => document.activeElement?.tagName)).toBe('BODY')

    await indicador.evaluate((el) => {
      ;(el as HTMLElement & { __qaMarcado?: boolean }).__qaMarcado = true
    })
    await page.getByRole('button', { name: new RegExp(PROJECT_NAME) }).click()
    await page.waitForURL(`**/projects/${PROJECT_ID}`)
    await page.goBack()
    await expect(page.getByRole('banner').getByRole('status')).toHaveText('En vivo')
    const mismoNodo = await page
      .getByRole('banner')
      .getByRole('status')
      .evaluate((el) => (el as HTMLElement & { __qaMarcado?: boolean }).__qaMarcado === true)
    expect(mismoNodo).toBe(true)
    const focoEnIndicador = await page.evaluate(() =>
      (document.activeElement?.closest('[role="status"]') ?? null) !== null,
    )
    expect(focoEnIndicador).toBe(false)
  })
})
