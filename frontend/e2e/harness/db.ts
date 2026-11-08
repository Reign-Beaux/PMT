// Acceso a la base de pruebas con el psql del contenedor (sin instalar nada)
// y migraciones aisladas: solo si la base no tiene esquema, con goose desde
// backend/ y GOCACHE en frontend/.harness (nunca escribe en backend/).
import { execFileSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { DATABASE_URL, DB } from './env'

const here = path.dirname(fileURLToPath(import.meta.url))
export const HARNESS_DIR = path.resolve(here, '../../.harness')
const BACKEND_DIR = path.resolve(here, '../../../backend')

export function psql(sql: string): string {
  return execFileSync(
    'docker',
    [
      'exec', '-i', 'postgresql', 'psql',
      '-U', DB.user,
      '-d', DB.database,
      '-v', 'ON_ERROR_STOP=1',
      '-qtA',
      '-c', sql,
    ],
    { encoding: 'utf8' },
  ).trim()
}

export function ensureSchema(): void {
  if (psql("select to_regclass('public.projects')") !== '') return
  // La URL con contraseña viaja solo por entorno (GOOSE_DBSTRING), nunca como
  // argumento: ni queda en la lista de procesos ni sale en el error si goose falla.
  execFileSync('go', ['tool', 'goose', '-dir', 'internal/platform/postgres/migrations', 'up'], {
    cwd: BACKEND_DIR,
    env: {
      ...process.env,
      DATABASE_URL,
      GOOSE_DRIVER: 'postgres',
      GOOSE_DBSTRING: DATABASE_URL,
      GOCACHE: path.join(HARNESS_DIR, 'go-cache'),
    },
    stdio: 'inherit',
  })
}
