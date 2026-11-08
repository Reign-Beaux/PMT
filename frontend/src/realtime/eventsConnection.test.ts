import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { FakeWebSocket } from '../test/fakeWebSocket'
import { createEventsConnection, eventsUrl } from './eventsConnection'

const URL = 'ws://localhost:5173/api/events'
const NOW = new Date('2026-11-07T12:00:00.000Z')
const RESYNC = JSON.stringify({ type: 'resync' })
const INVALIDATE_TASK = JSON.stringify({ type: 'invalidate', resource: 'task' })

function setup() {
  return createEventsConnection({
    url: URL,
    createSocket: (url) => new FakeWebSocket(url),
    now: () => NOW,
  })
}

beforeEach(() => {
  vi.useFakeTimers()
  FakeWebSocket.reset()
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('eventsUrl', () => {
  it('da ws:// con http: y wss:// con https:', () => {
    expect(eventsUrl({ protocol: 'http:', host: 'localhost:5173' })).toBe(
      'ws://localhost:5173/api/events',
    )
    expect(eventsUrl({ protocol: 'https:', host: 'pmt.example.com' })).toBe(
      'wss://pmt.example.com/api/events',
    )
  })
})

describe('createEventsConnection', () => {
  it('tras start() está en reconnecting y pasa a connected al llegar resync', () => {
    const conn = setup()
    conn.start()
    expect(conn.getStatus()).toEqual({ status: 'reconnecting' })
    const socket = FakeWebSocket.latest()
    expect(socket.url).toBe(URL)
    socket.open()
    expect(conn.getStatus().status).toBe('reconnecting')
    socket.emit(RESYNC)
    expect(conn.getStatus()).toEqual({
      status: 'connected',
      lastSyncedAt: NOW.toISOString(),
    })
  })

  it('reparte el invalidate parseado a todos los suscriptores con un solo socket', () => {
    const conn = setup()
    const a = vi.fn()
    const b = vi.fn()
    conn.subscribe(a)
    conn.subscribe(b)
    conn.start()
    expect(FakeWebSocket.instances).toHaveLength(1)
    FakeWebSocket.latest().emit(INVALIDATE_TASK)
    const expected = { type: 'invalidate', resource: 'task' }
    expect(a).toHaveBeenCalledWith(expected)
    expect(b).toHaveBeenCalledWith(expected)
  })

  it('frame inválido: console.warn, sin cambio de estado y sin reparto', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const conn = setup()
    const listener = vi.fn()
    conn.subscribe(listener)
    conn.start()
    FakeWebSocket.latest().emit('{no es json')
    expect(warn).toHaveBeenCalledWith(
      '[eventos] frame inválido:',
      expect.any(String),
    )
    expect(conn.getStatus().status).toBe('reconnecting')
    expect(listener).not.toHaveBeenCalled()
  })

  it('frame con tipo desconocido y frame no texto se ignoran', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const conn = setup()
    const listener = vi.fn()
    conn.subscribe(listener)
    conn.start()
    const socket = FakeWebSocket.latest()
    socket.emit(JSON.stringify({ type: 'otro' }))
    socket.emit(42)
    expect(warn).toHaveBeenCalledTimes(2)
    expect(listener).not.toHaveBeenCalled()
  })

  it('las esperas son 1, 2, 4, 8, 16, 30 y 30 s, y vuelven a 1 s tras un resync', () => {
    const conn = setup()
    conn.start()
    for (const wait of [1000, 2000, 4000, 8000, 16000, 30000, 30000]) {
      const before = FakeWebSocket.instances.length
      FakeWebSocket.latest().serverClose()
      vi.advanceTimersByTime(wait - 1)
      expect(FakeWebSocket.instances).toHaveLength(before)
      vi.advanceTimersByTime(1)
      expect(FakeWebSocket.instances).toHaveLength(before + 1)
    }
    const socket = FakeWebSocket.latest()
    socket.emit(RESYNC)
    const before = FakeWebSocket.instances.length
    socket.serverClose()
    vi.advanceTimersByTime(999)
    expect(FakeWebSocket.instances).toHaveLength(before)
    vi.advanceTimersByTime(1)
    expect(FakeWebSocket.instances).toHaveLength(before + 1)
  })

  it('a los 30 s sin resync pasa a stale, conserva lastSyncedAt y sigue reintentando', () => {
    const conn = setup()
    conn.start()
    FakeWebSocket.latest().emit(RESYNC)
    FakeWebSocket.latest().serverClose()
    expect(conn.getStatus()).toEqual({
      status: 'reconnecting',
      lastSyncedAt: NOW.toISOString(),
    })
    vi.advanceTimersByTime(30_000)
    expect(conn.getStatus()).toEqual({
      status: 'stale',
      lastSyncedAt: NOW.toISOString(),
    })
    FakeWebSocket.latest().serverClose()
    const before = FakeWebSocket.instances.length
    vi.advanceTimersByTime(2_000)
    expect(FakeWebSocket.instances).toHaveLength(before + 1)
    expect(conn.getStatus().status).toBe('stale')
  })

  it('start() sin resync pasa a stale a los 30 s sin lastSyncedAt', () => {
    const conn = setup()
    conn.start()
    vi.advanceTimersByTime(30_000)
    expect(conn.getStatus()).toEqual({ status: 'stale' })
  })

  it('con el backend caído, stale llega a los 30 s del primer cierre y no se reinicia con los reintentos', () => {
    const conn = setup()
    conn.start()
    FakeWebSocket.latest().emit(RESYNC)
    expect(conn.getStatus().status).toBe('connected')
    FakeWebSocket.latest().serverClose()
    for (const wait of [1_000, 2_000, 4_000, 8_000]) {
      vi.advanceTimersByTime(wait)
      FakeWebSocket.latest().serverClose()
      expect(conn.getStatus().status).toBe('reconnecting')
    }
    vi.advanceTimersByTime(14_999)
    expect(conn.getStatus().status).toBe('reconnecting')
    vi.advanceTimersByTime(1)
    expect(conn.getStatus()).toEqual({
      status: 'stale',
      lastSyncedAt: NOW.toISOString(),
    })
  })

  it('close() no deja temporizadores, cierra el socket y no crea más', () => {
    const conn = setup()
    conn.start()
    const socket = FakeWebSocket.latest()
    conn.close()
    expect(vi.getTimerCount()).toBe(0)
    expect(socket.closedByClient).toBe(true)
    const before = FakeWebSocket.instances.length
    vi.advanceTimersByTime(120_000)
    expect(FakeWebSocket.instances).toHaveLength(before)
  })

  it('frames y caídas después de close() se ignoran', () => {
    const conn = setup()
    const listener = vi.fn()
    conn.subscribe(listener)
    conn.start()
    const socket = FakeWebSocket.latest()
    conn.close()
    socket.emit(INVALIDATE_TASK)
    socket.serverClose()
    expect(listener).not.toHaveBeenCalled()
    expect(vi.getTimerCount()).toBe(0)
  })

  it('start() dos veces y close() sin start() no duplican sockets; start() tras close() crea uno nuevo', () => {
    const conn = setup()
    conn.start()
    conn.start()
    expect(FakeWebSocket.instances).toHaveLength(1)
    conn.close()
    conn.close()
    conn.start()
    expect(FakeWebSocket.instances).toHaveLength(2)
    const other = setup()
    other.close()
    expect(FakeWebSocket.instances).toHaveLength(2)
  })

  it('si un listener lanza, el siguiente recibe el evento igual', () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    const conn = setup()
    const ok = vi.fn()
    conn.subscribe(() => {
      throw new Error('listener roto')
    })
    conn.subscribe(ok)
    conn.start()
    FakeWebSocket.latest().emit(RESYNC)
    expect(ok).toHaveBeenCalledWith({ type: 'resync' })
    expect(error).toHaveBeenCalled()
  })

  it('si createSocket lanza, registra error y programa el siguiente reintento', () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    let falla = true
    const conn = createEventsConnection({
      url: URL,
      createSocket: (url) => {
        if (falla) {
          falla = false
          throw new Error('sin socket')
        }
        return new FakeWebSocket(url)
      },
      now: () => NOW,
    })
    conn.start()
    expect(error).toHaveBeenCalled()
    expect(FakeWebSocket.instances).toHaveLength(0)
    vi.advanceTimersByTime(1_000)
    expect(FakeWebSocket.instances).toHaveLength(1)
  })

  it('desuscribirse durante el reparto no altera el reparto en curso', () => {
    const conn = setup()
    const b = vi.fn()
    const offB = conn.subscribe(b)
    conn.subscribe(() => {
      offB()
    })
    conn.start()
    const socket = FakeWebSocket.latest()
    socket.emit(INVALIDATE_TASK)
    expect(b).toHaveBeenCalledTimes(1)
    socket.emit(INVALIDATE_TASK)
    expect(b).toHaveBeenCalledTimes(1)
  })

  it('resync estando ya conectado actualiza lastSyncedAt y reparte el evento', () => {
    let current = NOW
    const conn = createEventsConnection({
      url: URL,
      createSocket: (url) => new FakeWebSocket(url),
      now: () => current,
    })
    const listener = vi.fn()
    conn.subscribe(listener)
    conn.start()
    const socket = FakeWebSocket.latest()
    socket.emit(RESYNC)
    current = new Date('2026-11-07T13:00:00.000Z')
    socket.emit(RESYNC)
    expect(conn.getStatus()).toEqual({
      status: 'connected',
      lastSyncedAt: '2026-11-07T13:00:00.000Z',
    })
    expect(listener).toHaveBeenCalledTimes(2)
  })
})
