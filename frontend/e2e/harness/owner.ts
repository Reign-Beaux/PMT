// Dueño de la instancia de pruebas. Solo existe en pmt_frontend_test: el backend
// de pruebas lo crea al arrancar si la base no tiene usuarios (OWNER_* en
// playwright.config.ts) y el harness entra con él.
export const TEST_OWNER = {
  email: 'e2e-dueno@pmt.test',
  password: 'e2e-dueno-contrasena',
  username: 'e2e-dueno',
} as const
