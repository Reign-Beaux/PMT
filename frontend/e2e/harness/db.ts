// Acceso a la base de pruebas con el psql del contenedor (sin instalar nada).
// Las migraciones las aplica el backend de pruebas al arrancar.
import { execFileSync } from 'node:child_process'
import { DB } from './env'

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
