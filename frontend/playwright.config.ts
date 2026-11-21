import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, devices } from '@playwright/test'
import { assertTestDatabase, assertTestTarget, resolveApiTarget } from './config/apiTarget'

// Validación al cargar: destino (solo 127.0.0.1:8090) y base (solo
// pmt_frontend_test) antes de levantar servidores o crear datos.
const API_TARGET = assertTestTarget(resolveApiTarget(process.env))
assertTestDatabase(process.env.PMT_TEST_DATABASE_URL)

const frontendDir = path.dirname(fileURLToPath(import.meta.url))
const harnessDir = path.join(frontendDir, '.harness')
const binario = path.join(harnessDir, 'pmt-server-pruebas')
const configPruebas = path.join(frontendDir, 'e2e', 'harness', 'config.e2e.yaml')

export default defineConfig({
  testDir: './e2e',
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'retain-on-failure',
    launchOptions: process.env.PMT_BROWSER_PATH ? { executablePath: process.env.PMT_BROWSER_PATH } : {},
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
      testIgnore: [/etapa5-tiempo-real\.spec\.ts/, /\/ui\//],
    },
    {
      // La reconexión detiene y relanza el 8090: corre al final, cuando ningún
      // otro spec usa el backend. Aislado: pnpm test:e2e --project=reconexion --no-deps
      name: 'reconexion',
      use: { ...devices['Desktop Chrome'] },
      testMatch: /etapa5-tiempo-real\.spec\.ts/,
      dependencies: ['chromium'],
    },
  ],
  webServer: [
    {
      // Backend de pruebas: binario propio en frontend/.harness con su propio
      // CONFIG_FILE (127.0.0.1:8090, frontend/dist); nunca lee
      // backend/config/config.dev.yaml ni toca backend/bin o el 8080. La base
      // llega por DATABASE_CONNECTION_STRING y el servidor aplica las
      // migraciones al arrancar. Playwright solo baja lo que él levanta.
      command: `mkdir -p .harness && go build -C ../backend -o ${binario} ./cmd/pmt-server && cd .harness && exec ./pmt-server-pruebas`,
      url: `${API_TARGET}/readyz`,
      env: {
        CONFIG_FILE: configPruebas,
        DATABASE_CONNECTION_STRING: process.env.PMT_TEST_DATABASE_URL ?? '',
        GOCACHE: path.join(harnessDir, 'go-cache'),
      },
      reuseExistingServer: true,
      timeout: 180_000,
    },
    {
      command: 'pnpm dev --port 5173 --strictPort',
      url: 'http://localhost:5173',
      reuseExistingServer: true,
    },
  ],
})
