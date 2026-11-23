import { get, post } from './http'

/** Estado de un código del CLI pmt: lo que ve la pantalla /cli. */
export type CliLoginStatus = 'pending' | 'authorized' | 'cancelled' | 'expired'

export interface CliLogin {
  code: string
  machine: string
  token_name: string
  status: CliLoginStatus
  expires_at: string
}

const path = (code: string) => `/cli/login/${encodeURIComponent(code)}`

export const getCliLogin = (code: string, signal: AbortSignal) => get<CliLogin>(path(code), { signal })
export const authorizeCliLogin = (code: string) => post<CliLogin>(`${path(code)}/authorize`)
export const cancelCliLogin = (code: string) => post<CliLogin>(`${path(code)}/cancel`)
