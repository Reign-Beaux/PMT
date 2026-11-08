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

export async function get<T>(path: string, options?: GetOptions): Promise<T> {
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
    const response = await fetch(`/api${path}`, {
      headers: { Accept: 'application/json' },
      signal: controller.signal,
    })

    if (!response.ok) {
      throw new HttpError('http', await httpErrorMessage(response), {
        status: response.status,
      })
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
