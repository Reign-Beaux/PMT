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

export default defineConfig({
  testDir: './e2e',
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
      testIgnore: /etapa5-tiempo-real\.spec\.ts/,
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
      // Backend de pruebas: binario propio en frontend/.harness, arrancado sin
      // config.yaml (mandan DATABASE_URL/SERVER_ADDR). Nunca toca backend/bin
      // ni la instancia del 8080. Playwright solo baja lo que él levanta.
      command: `mkdir -p .harness && go build -C ../backend -o ${binario} ./cmd/pmt-server && cd .harness && exec ./pmt-server-pruebas`,
      url: `${API_TARGET}/api/projects`,
      env: {
        DATABASE_URL: process.env.PMT_TEST_DATABASE_URL ?? '',
        SERVER_ADDR: '127.0.0.1:8090',
        // El binario corre desde frontend/.harness: el default relativo
        // ../frontend/dist no aplica; se apunta al dist real del build.
        FRONTEND_DIST: path.join(frontendDir, 'dist'),
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
