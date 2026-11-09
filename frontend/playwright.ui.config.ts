import { defineConfig } from '@playwright/test'

/** Pruebas de interfaz con HTTP y WebSocket controlados: sin base de datos. */
export default defineConfig({
  testDir: './e2e/ui',
  fullyParallel: true,
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'retain-on-failure',
    launchOptions: process.env.PMT_BROWSER_PATH ? { executablePath: process.env.PMT_BROWSER_PATH } : {},
  },
  webServer: { command: 'pnpm dev', url: 'http://localhost:5173', reuseExistingServer: false },
})
