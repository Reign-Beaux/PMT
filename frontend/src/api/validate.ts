import { HttpError } from './http'

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function invalid(detail: string): HttpError {
  return new HttpError('invalid_response', 'Respuesta no válida del servidor', {
    cause: detail,
  })
}

export function requireRecord(value: unknown): Record<string, unknown> {
  if (!isRecord(value)) throw invalid('se esperaba un objeto')
  return value
}

export function requireArray(value: unknown): unknown[] {
  if (!Array.isArray(value)) throw invalid('se esperaba un array')
  return value
}

export function requireString(record: Record<string, unknown>, field: string): string {
  const value = record[field]
  if (typeof value !== 'string') throw invalid(`${field} debe ser string`)
  return value
}

export function optionalString(
  record: Record<string, unknown>,
  field: string,
): string | undefined {
  const value = record[field]
  if (value === undefined) return undefined
  if (typeof value !== 'string') throw invalid(`${field} debe ser string o ausente`)
  return value
}

export function requireInt(record: Record<string, unknown>, field: string): number {
  const value = record[field]
  if (typeof value !== 'number' || !Number.isInteger(value))
    throw invalid(`${field} debe ser entero`)
  return value
}

export function requireStatus<S extends string>(
  record: Record<string, unknown>,
  field: string,
  labels: Record<S, string>,
): S {
  const value = record[field]
  if (typeof value !== 'string' || !Object.hasOwn(labels, value))
    throw invalid(`${field} tiene un estado desconocido: ${String(value)}`)
  return value as S
}

export function optionalStringArray(
  record: Record<string, unknown>,
  field: string,
): string[] | null | undefined {
  const value = record[field]
  if (value === undefined) return undefined
  if (value === null) return null
  if (!Array.isArray(value))
    throw invalid(`${field} debe ser array de strings, null o ausente`)
  for (const item of value) {
    if (typeof item !== 'string')
      throw invalid(`${field} contiene un elemento que no es string`)
  }
  return value
}

export function requireStringArray(
  record: Record<string, unknown>,
  field: string,
): string[] {
  return requireArrayField(record, field).map((item) => {
    if (typeof item !== 'string')
      throw invalid(`${field} contiene un elemento que no es string`)
    return item
  })
}

export function requireArrayField(record: Record<string, unknown>, field: string): unknown[] {
  return requireArray(record[field])
}

export function requireCounts(
  record: Record<string, unknown>,
  field: string,
): Record<string, number> {
  const value = requireRecord(record[field])
  const counts: Record<string, number> = {}
  for (const [key, item] of Object.entries(value)) {
    if (typeof item !== 'number' || !Number.isInteger(item) || item < 0)
      throw invalid(`${field}.${key} debe ser un entero >= 0`)
    counts[key] = item
  }
  return counts
}

export function requireOneOf<V extends string>(
  record: Record<string, unknown>,
  field: string,
  values: readonly V[],
): V {
  const value = record[field]
  if (typeof value !== 'string' || !values.includes(value as V))
    throw invalid(`${field} tiene un valor desconocido: ${String(value)}`)
  return value as V
}
