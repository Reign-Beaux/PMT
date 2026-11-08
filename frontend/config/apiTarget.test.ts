import { describe, expect, it } from 'vitest'
import {
  DEFAULT_API_TARGET,
  assertTestDatabase,
  assertTestTarget,
  maskDatabaseUrl,
  resolveApiTarget,
} from './apiTarget.ts'

describe('resolveApiTarget', () => {
  it('usa el destino por defecto si PMT_API_TARGET está ausente o vacío', () => {
    expect(resolveApiTarget({})).toBe(DEFAULT_API_TARGET)
    expect(resolveApiTarget({ PMT_API_TARGET: '' })).toBe(DEFAULT_API_TARGET)
    expect(resolveApiTarget({ PMT_API_TARGET: '   ' })).toBe(DEFAULT_API_TARGET)
  })

  it('normaliza el "/" final', () => {
    expect(resolveApiTarget({ PMT_API_TARGET: 'http://127.0.0.1:8090/' })).toBe(
      'http://127.0.0.1:8090',
    )
    expect(resolveApiTarget({ PMT_API_TARGET: ' http://localhost:8090 ' })).toBe(
      'http://localhost:8090',
    )
  })

  it('rechaza URL inválida, con ruta y protocolos que no son http(s)', () => {
    expect(() => resolveApiTarget({ PMT_API_TARGET: 'no-es-url' })).toThrow(
      'PMT_API_TARGET no es una URL válida: no-es-url',
    )
    expect(() => resolveApiTarget({ PMT_API_TARGET: 'http://127.0.0.1:8090/api' })).toThrow(
      'PMT_API_TARGET no es una URL válida: http://127.0.0.1:8090/api',
    )
    expect(() => resolveApiTarget({ PMT_API_TARGET: 'ws://127.0.0.1:8090' })).toThrow(
      /no es una URL válida/,
    )
    expect(() => resolveApiTarget({ PMT_API_TARGET: 'ftp://127.0.0.1:8090' })).toThrow(
      /no es una URL válida/,
    )
  })
})

describe('assertTestTarget', () => {
  it('acepta 127.0.0.1:8090 y localhost:8090', () => {
    expect(assertTestTarget('http://127.0.0.1:8090')).toBe('http://127.0.0.1:8090')
    expect(assertTestTarget('http://localhost:8090')).toBe('http://localhost:8090')
  })

  it('rechaza 8080, otro host y otro puerto', () => {
    expect(() => assertTestTarget('http://127.0.0.1:8080')).toThrow(
      'Las pruebas solo pueden usar el backend de pruebas en 127.0.0.1:8090 (recibido http://127.0.0.1:8080)',
    )
    expect(() => assertTestTarget('http://example.com:8090')).toThrow(/127\.0\.0\.1:8090/)
    expect(() => assertTestTarget('http://127.0.0.1:9090')).toThrow(/127\.0\.0\.1:8090/)
    expect(() => assertTestTarget('http://127.0.0.1')).toThrow(/127\.0\.0\.1:8090/)
  })
})

describe('assertTestDatabase', () => {
  const valida = 'postgres://usuario:clave@localhost:5432/pmt_frontend_test?sslmode=disable'

  it('acepta pmt_frontend_test con ?sslmode y devuelve usuario y base', () => {
    expect(assertTestDatabase(valida)).toEqual({
      user: 'usuario',
      database: 'pmt_frontend_test',
    })
    expect(assertTestDatabase('postgresql://otro_usuario@localhost:5432/pmt_frontend_test')).toEqual(
      { user: 'otro_usuario', database: 'pmt_frontend_test' },
    )
  })

  it('rechaza ausente, otra base y otro protocolo', () => {
    expect(() => assertTestDatabase(undefined)).toThrow(/obligatoria/)
    expect(() => assertTestDatabase('  ')).toThrow(/obligatoria/)
    expect(() => assertTestDatabase('postgres://usuario@localhost:5432/pmt')).toThrow(
      /solo pueden usar la base pmt_frontend_test \(recibido pmt\)/,
    )
    expect(() =>
      assertTestDatabase('mysql://usuario@localhost:3306/pmt_frontend_test'),
    ).toThrow(/protocolo postgres\/postgresql/)
  })

  it('no filtra la contraseña en los mensajes de error', () => {
    let mensaje = ''
    try {
      assertTestDatabase('postgres://u:secreto@mala:5432/otra_base')
    } catch (error) {
      mensaje = (error as Error).message
    }
    expect(mensaje).toMatch(/solo pueden usar la base pmt_frontend_test \(recibido otra_base\)/)
    expect(mensaje).not.toContain('secreto')
  })
})

describe('maskDatabaseUrl', () => {
  it('oculta la contraseña', () => {
    expect(
      maskDatabaseUrl('postgres://usuario:clave@localhost:5432/pmt_frontend_test?sslmode=disable'),
    ).toBe('postgres://usuario:***@localhost:5432/pmt_frontend_test?sslmode=disable')
  })

  it('deja intacta una URL sin contraseña', () => {
    expect(maskDatabaseUrl('postgres://localhost:5432/pmt_frontend_test')).toBe(
      'postgres://localhost:5432/pmt_frontend_test',
    )
  })
})
