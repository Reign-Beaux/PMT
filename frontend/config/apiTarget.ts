export const DEFAULT_API_TARGET = 'http://127.0.0.1:8090'
export const TEST_DATABASE = 'pmt_frontend_test'

const TEST_HOSTS = new Set(['127.0.0.1', 'localhost'])
const TEST_PORT = '8090'

function invalidTarget(value: string): Error {
  return new Error(`PMT_API_TARGET no es una URL válida: ${value}`)
}

export function resolveApiTarget(env: NodeJS.ProcessEnv = process.env): string {
  const raw = (env.PMT_API_TARGET ?? '').trim()
  if (raw === '') return DEFAULT_API_TARGET
  let url: URL
  try {
    url = new URL(raw)
  } catch {
    throw invalidTarget(raw)
  }
  if (
    (url.protocol !== 'http:' && url.protocol !== 'https:') ||
    url.pathname !== '/' ||
    url.search !== '' ||
    url.hash !== ''
  ) {
    throw invalidTarget(raw)
  }
  return url.origin
}

export function assertTestTarget(url: string): string {
  const parsed = new URL(url)
  if (!TEST_HOSTS.has(parsed.hostname) || parsed.port !== TEST_PORT) {
    throw new Error(
      `Las pruebas solo pueden usar el backend de pruebas en 127.0.0.1:${TEST_PORT} (recibido ${url})`,
    )
  }
  return url
}

export type TestDatabase = { user: string; database: string }

export function assertTestDatabase(url: string | undefined): TestDatabase {
  const raw = (url ?? '').trim()
  if (raw === '') {
    throw new Error('PMT_TEST_DATABASE_URL es obligatoria para las pruebas')
  }
  let parsed: URL
  try {
    parsed = new URL(raw)
  } catch {
    throw new Error(`PMT_TEST_DATABASE_URL no es una URL válida: ${maskDatabaseUrl(raw)}`)
  }
  if (parsed.protocol !== 'postgres:' && parsed.protocol !== 'postgresql:') {
    throw new Error(
      `PMT_TEST_DATABASE_URL debe usar el protocolo postgres/postgresql (recibido ${parsed.protocol.replace(/:$/, '')})`,
    )
  }
  const database = parsed.pathname.replace(/^\/+/, '')
  if (database !== TEST_DATABASE) {
    throw new Error(
      `Las pruebas solo pueden usar la base ${TEST_DATABASE} (recibido ${database === '' ? '(sin nombre)' : database})`,
    )
  }
  return { user: parsed.username, database }
}

export function maskDatabaseUrl(url: string): string {
  try {
    const parsed = new URL(url)
    if (parsed.password !== '') parsed.password = '***'
    return parsed.toString()
  } catch {
    return url.replace(/(\/\/[^:/@]+:)[^@]+(@)/, '$1***$2')
  }
}
