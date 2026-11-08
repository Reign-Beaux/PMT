import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import { configDefaults, defineConfig } from 'vitest/config'
import { assertTestTarget, resolveApiTarget } from './config/apiTarget.ts'

// En esta etapa el frontend solo corre para pruebas: se rechaza cualquier
// destino que no sea el backend de pruebas antes de arrancar nada.
const apiTarget = assertTestTarget(resolveApiTarget(process.env))

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    babel({ presets: [reactCompilerPreset()] })
  ],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      // /api/events es el canal WebSocket de eventos: el proxy hace upgrade.
      '/api/events': { target: apiTarget, ws: true, changeOrigin: true },
      '/api': { target: apiTarget, changeOrigin: true },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: false,
    include: ['src/**/*.test.{ts,tsx}', 'config/**/*.test.ts'],
    exclude: [...configDefaults.exclude, 'e2e/**'],
  },
})
