export type HttpErrorKind =
  | 'http'
  | 'network'
  | 'timeout'
  | 'invalid_json'
  | 'aborted'
  | 'invalid_response'

export interface HttpErrorOptions {
  status?: number
  cause?: unknown
}

export class HttpError extends Error {
  kind: HttpErrorKind
  status?: number

  constructor(kind: HttpErrorKind, message: string, options?: HttpErrorOptions) {
    super(message, options?.cause !== undefined ? { cause: options.cause } : undefined)
    this.name = 'HttpError'
    this.kind = kind
    if (options?.status !== undefined) {
      this.status = options.status
    }
  }
}

export interface GetOptions {
  signal?: AbortSignal
}

const TIMEOUT_MS = 15_000

/** Evento de window que se emite cuando la API responde 401: la sesión venció. */
export const UNAUTHORIZED_EVENT = 'pmt:unauthorized'

export async function get<T>(path: string, options?: GetOptions): Promise<T> {
  return request<T>('GET', path, undefined, options)
}

/** POST con cuerpo JSON opcional. Un 204 resuelve undefined. */
export function post<T>(path: string, body?: unknown, options?: GetOptions): Promise<T> {
  return request<T>('POST', path, body, options)
}

/** PUT con cuerpo JSON. */
export function put<T>(path: string, body: unknown, options?: GetOptions): Promise<T> {
  return request<T>('PUT', path, body, options)
}

/** DELETE. Un 204 resuelve undefined. */
export function del<T>(path: string, options?: GetOptions): Promise<T> {
  return request<T>('DELETE', path, undefined, options)
}

async function request<T>(
  method: string,
  path: string,
  body: unknown,
  options?: GetOptions,
): Promise<T> {
  const external = options?.signal
  if (external?.aborted) {
    throw new HttpError('aborted', 'Consulta cancelada')
  }

  const controller = new AbortController()
  let timedOut = false
  const timer = setTimeout(() => {
    timedOut = true
    controller.abort()
  }, TIMEOUT_MS)
  const onExternalAbort = () => controller.abort()
  external?.addEventListener('abort', onExternalAbort, { once: true })

  try {
    const headers: Record<string, string> = { Accept: 'application/json' }
    const init: RequestInit = { headers, signal: controller.signal }
    if (method !== 'GET') {
      init.method = method
      if (body !== undefined) {
        headers['Content-Type'] = 'application/json'
        init.body = JSON.stringify(body)
      }
    }
    const response = await fetch(`/api${path}`, init)

    if (!response.ok) {
      if (response.status === 401 && !path.startsWith('/auth/')) {
        window.dispatchEvent(new Event(UNAUTHORIZED_EVENT))
      }
      throw new HttpError('http', await httpErrorMessage(response), {
        status: response.status,
      })
    }

    if (response.status === 204) {
      return undefined as T
    }
    const text = await response.text()
    try {
      return JSON.parse(text) as T
    } catch (cause) {
      throw new HttpError('invalid_json', 'Respuesta no válida del servidor', { cause })
    }
  } catch (error) {
    if (error instanceof HttpError) {
      throw error
    }
    if (timedOut) {
      throw new HttpError('timeout', 'El servidor tardó demasiado en responder', {
        cause: error,
      })
    }
    if (external?.aborted) {
      throw new HttpError('aborted', 'Consulta cancelada', { cause: error })
    }
    throw new HttpError('network', 'No se pudo conectar con el servidor', { cause: error })
  } finally {
    clearTimeout(timer)
    external?.removeEventListener('abort', onExternalAbort)
  }
}

async function httpErrorMessage(response: Response): Promise<string> {
  const text = await response.text()
  try {
    const body: unknown = JSON.parse(text)
    if (typeof body === 'object' && body !== null && 'error' in body) {
      const message = (body as { error: unknown }).error
      if (typeof message === 'string' && message !== '') {
        return message
      }
    }
  } catch {
    // Cuerpo no JSON: mensaje genérico con el status.
  }
  return `Error del servidor (${response.status})`
}
