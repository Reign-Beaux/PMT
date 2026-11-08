/**
 * WebSocket falso para pruebas unitarias: registra cada instancia y permite
 * simular apertura, frames entrantes y caídas sin servidor.
 */
export class FakeWebSocket {
  static instances: FakeWebSocket[] = []

  static reset(): void {
    FakeWebSocket.instances = []
  }

  static latest(): FakeWebSocket {
    const socket = FakeWebSocket.instances.at(-1)
    if (socket === undefined) {
      throw new Error('no hay instancias de FakeWebSocket')
    }
    return socket
  }

  readonly url: string
  readyState = 0
  closedByClient = false
  onopen: ((ev?: unknown) => void) | null = null
  onmessage: ((ev: { data: unknown }) => void) | null = null
  onerror: ((ev?: unknown) => void) | null = null
  onclose: ((ev?: unknown) => void) | null = null

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  /** Simula que el servidor acepta la conexión. */
  open(): void {
    this.readyState = 1
    this.onopen?.()
  }

  /** Simula un frame entrante. */
  emit(data: unknown): void {
    this.onmessage?.({ data })
  }

  /** Simula una caída iniciada por el servidor. */
  serverClose(): void {
    this.readyState = 3
    this.onclose?.()
  }

  close(): void {
    this.readyState = 3
    this.closedByClient = true
    this.onclose?.()
  }
}
