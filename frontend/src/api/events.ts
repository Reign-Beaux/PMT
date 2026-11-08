/**
 * Contrato de /api/events (WebSocket):
 * - Solo recepción: el cliente no envía frames.
 * - El primer frame con un listener disponible es {type: 'resync'}.
 * - Cada cambio confirmado (tras commit) emite {type: 'invalidate', resource}.
 * - Tras recuperar el listener después de una caída llega {type: 'resync'}.
 *
 * La conexión y la reconexión corresponden a la etapa Tiempo real e integración;
 * aquí solo se entrega el parser de frames.
 */

export type ChangeEventResource =
  | 'project'
  | 'stage'
  | 'task'
  | 'bug'
  | 'comment'
  | 'status_change'

export type ChangeEvent =
  | { type: 'resync' }
  | { type: 'invalidate'; resource: ChangeEventResource }

export type ParseResult =
  | { ok: true; event: ChangeEvent }
  | { ok: false; error: string }

const RESOURCES: readonly ChangeEventResource[] = [
  'project',
  'stage',
  'task',
  'bug',
  'comment',
  'status_change',
]

function fail(error: string): ParseResult {
  return { ok: false, error }
}

/** Parsea un frame JSON del WebSocket; nunca lanza. */
export function parseChangeEvent(data: unknown): ParseResult {
  if (typeof data !== 'string') {
    return fail('el frame no es texto')
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(data)
  } catch {
    return fail('el frame no es JSON válido')
  }

  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    return fail('el evento no es un objeto')
  }

  const record = parsed as Record<string, unknown>

  if (record['type'] === 'resync') {
    return { ok: true, event: { type: 'resync' } }
  }

  if (record['type'] === 'invalidate') {
    const resource = record['resource']
    if (typeof resource !== 'string') {
      return fail('invalidate sin resource')
    }
    if (!RESOURCES.includes(resource as ChangeEventResource)) {
      return fail(`recurso no permitido: ${resource}`)
    }
    return {
      ok: true,
      event: { type: 'invalidate', resource: resource as ChangeEventResource },
    }
  }

  return fail(`tipo de evento desconocido: ${String(record['type'])}`)
}
