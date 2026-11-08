// E2E de tiempo real (tarea 04 de la etapa "Tiempo real e integración", plan
// 1a56f403 v1): detener y volver a levantar solo el proceso de prueba propio
// del 8090 muestra "Reconectando…" y luego "En vivo" con los datos al día, sin
// recargar la página y sin interceptar HTTP ni WebSocket. La tarea creada con
// el backend abajo (por psql, sin evento) aparece tras el resync.
// También cubre el criterio 15 de QA (plan 1d0244fd v1): a los ~30 s abajo el
// indicador pasa a "Sin conexión, datos de las HH:MM" con la hora del último
// resync, y el foco no se mueve con los cambios de estado (criterio 16).
// Este spec necesita el 8090 en exclusiva: corre al final, en el proyecto
// 'reconexion' de playwright.config.ts. Si el 8090 lo levantó otro proceso
// (reuseExistingServer), la prueba se salta con su motivo y nunca lo mata.
// Fixtures del harness con ids e2e-<RUN_ID>, limpiados por su teardown aunque
// la prueba falle.
import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { expect } from '@playwright/test'
import { API_TARGET, test } from './harness/fixtures'
import { HARNESS_DIR } from './harness/db'
import { DATABASE_URL } from './harness/env'

const BINARIO = path.join(HARNESS_DIR, 'pmt-server-pruebas')
// El binario corre desde frontend/.harness: el default relativo del backend
// ../frontend/dist no aplica; se apunta al dist real, igual que el webServer.
const FRONTEND_DIST = path.resolve(HARNESS_DIR, '../dist')

// Backend relanzado por este spec: afterAll lo baja para dejar el 8090 libre.
let relanzado: ChildProcess | null = null

/** PID que escucha en 8090, o null si el puerto está libre. */
function pidEn8090(): number | null {
  let salida: string
  try {
    salida = execFileSync('ss', ['-Hltnp', 'sport = :8090'], { encoding: 'utf8' })
  } catch {
    return null
  }
  const encontrado = salida.match(/pid=(\d+)/)
  return encontrado ? Number(encontrado[1]) : null
}

/**
 * Solo es propio si es el binario del harness y su cadena de PPID incluye el
 * padre de este worker (el runner de Playwright que lo levantó como webServer).
 * Así nunca se detiene un 8090 ajeno reutilizado por reuseExistingServer.
 */
function esProcesoPropio(pid: number): boolean {
  try {
    if (fs.readlinkSync(`/proc/${pid}/exe`) !== BINARIO) return false
  } catch {
    return false
  }
  let actual = pid
  for (let salto = 0; salto < 16; salto++) {
    let stat: string
    try {
      stat = fs.readFileSync(`/proc/${actual}/stat`, 'utf8')
    } catch {
      return false
    }
    const ppid = Number(stat.slice(stat.lastIndexOf(')') + 1).trim().split(/\s+/)[1])
    if (ppid === process.ppid) return true
    if (ppid <= 1) return false
    actual = ppid
  }
  return false
}

async function esperar(
  condicion: () => boolean | Promise<boolean>,
  ms: number,
  mensaje: string,
): Promise<void> {
  const fin = Date.now() + ms
  while (Date.now() < fin) {
    if (await condicion()) return
    await new Promise((resolver) => setTimeout(resolver, 250))
  }
  throw new Error(mensaje)
}

async function apiResponde(): Promise<boolean> {
  try {
    const respuesta = await fetch(`${API_TARGET}/api/projects`)
    return respuesta.ok
  } catch {
    return false
  }
}

test.afterAll(async () => {
  if (
    relanzado !== null &&
    relanzado.pid !== undefined &&
    relanzado.exitCode === null &&
    relanzado.signalCode === null
  ) {
    const pid = relanzado.pid
    relanzado = null
    process.kill(pid, 'SIGTERM')
    await esperar(
      () => pidEn8090() === null,
      10_000,
      'el 8090 no se liberó tras bajar el backend relanzado en afterAll',
    )
  }
})

test('detener y relanzar el 8090 propio muestra "Reconectando…" y vuelve a "En vivo" con datos al día', async ({
  page,
  harness,
}) => {
  const pid = pidEn8090()
  if (pid === null || !esProcesoPropio(pid)) {
    test.skip(true, '8090 no lo levantó este run: no se detiene')
    return
  }
  // ~31 s hasta "Sin conexión" más hasta 40 s para volver a "En vivo".
  test.setTimeout(150_000)

  const proyecto = harness.createProject({ status: 'working' })
  const etapa = harness.createStage(proyecto.id, { position: 1, status: 'in_dev' })
  const tareaInicial = harness.createTask(etapa.id, { status: 'ready' })

  const antesDeSincronizar = Date.now()
  await page.goto(`/projects/${proyecto.id}/stages/${etapa.id}`)
  const indicador = page.getByRole('banner').getByRole('status')
  await expect(indicador, 'el indicador no llegó a "En vivo" al abrir el kanban').toHaveText(
    'En vivo',
    { timeout: 15_000 },
  )
  const despuesDeSincronizar = Date.now()
  const botonInicial = page.getByRole('button', { name: new RegExp(tareaInicial.title) })
  await expect(botonInicial).toBeVisible()
  // Foco en un elemento propio de la página: no debe moverse con el indicador.
  await botonInicial.focus()
  const focoEnTarea = () =>
    botonInicial.evaluate((el) => el === document.activeElement)
  await page.evaluate(() => {
    ;(window as unknown as { __sinRecarga?: boolean }).__sinRecarga = true
  })

  // Solo el proceso propio: SIGTERM y puerto libre antes de seguir.
  process.kill(pid, 'SIGTERM')
  await esperar(() => pidEn8090() === null, 10_000, 'el 8090 no se liberó tras SIGTERM')

  await expect(indicador, 'no apareció "Reconectando…" con el backend abajo').toHaveText(
    'Reconectando…',
    { timeout: 15_000 },
  )

  expect(await focoEnTarea(), 'el foco se movió al pasar a "Reconectando…"').toBe(true)

  // Criterio 15: sin reconectar en 30 s, el indicador muestra la hora del
  // último resync (al abrir la página), con el formato y la zona del navegador.
  const horasValidas = await page.evaluate(
    ([desde, hasta]) => {
      const formato = new Intl.DateTimeFormat('es', {
        hour: '2-digit',
        minute: '2-digit',
        hourCycle: 'h23',
      })
      const horas = new Set<string>()
      for (let t = desde; t < hasta; t += 60_000) horas.add(formato.format(new Date(t)))
      horas.add(formato.format(new Date(hasta)))
      return [...horas]
    },
    [antesDeSincronizar, despuesDeSincronizar],
  )
  await expect(indicador, 'no apareció "Sin conexión, datos de las HH:MM" a los ~30 s').toHaveText(
    /^Sin conexión, datos de las \d{2}:\d{2}$/,
    { timeout: 45_000 },
  )
  const horaMostrada = (await indicador.textContent())?.match(/\d{2}:\d{2}/)?.[0]
  expect(horasValidas, 'la hora mostrada no es la del último resync').toContain(horaMostrada)
  expect(await focoEnTarea(), 'el foco se movió al pasar a "Sin conexión"').toBe(true)

  // Con el backend abajo la tarea se crea por psql y no genera evento: solo el
  // resync al reconectar puede traerla.
  const tareaNueva = harness.createTask(etapa.id, { status: 'ready' })

  relanzado = spawn(BINARIO, {
    cwd: HARNESS_DIR,
    env: {
      ...process.env,
      DATABASE_URL,
      SERVER_ADDR: '127.0.0.1:8090',
      FRONTEND_DIST,
    },
    stdio: 'ignore',
    detached: true,
  })
  relanzado.unref()
  await esperar(apiResponde, 20_000, 'el backend relanzado no respondió en 20 s')

  // La espera de reintento crece hasta 30 s: "En vivo" puede tardar.
  await expect(indicador, 'no volvió a "En vivo" en 40 s').toHaveText('En vivo', {
    timeout: 40_000,
  })
  await expect(
    page.getByRole('button', { name: new RegExp(tareaNueva.title) }),
    'la tarea creada con el backend abajo no apareció tras el resync',
  ).toBeVisible({ timeout: 10_000 })
  expect(
    await page.evaluate(() => (window as unknown as { __sinRecarga?: boolean }).__sinRecarga === true),
    'la página no debe haberse recargado',
  ).toBe(true)
})
