import type { ApiToken, Invitation, NewApiToken, NewInvitation, User } from '../domain/user'
import { del, get, post } from './http'

export const getMe = (signal: AbortSignal) => get<User>('/auth/me', { signal })
export const login = (email: string, password: string) =>
  post<User>('/auth/login', { email, password })
export const logout = () => post<void>('/auth/logout')
export const changePassword = (currentPassword: string, newPassword: string) =>
  post<void>('/auth/me/password', { current_password: currentPassword, new_password: newPassword })

export const listTokens = (signal: AbortSignal) => get<ApiToken[]>('/tokens', { signal })
export const createToken = (name: string) => post<NewApiToken>('/tokens', { name })
export const revokeToken = (id: string) => del<void>(`/tokens/${encodeURIComponent(id)}`)

export const listInvitations = (signal: AbortSignal) =>
  get<Invitation[]>('/invitations', { signal })
export const createInvitation = (email: string) => post<NewInvitation>('/invitations', { email })
export const getInvitation = (token: string, signal: AbortSignal) =>
  get<{ email: string; valid: boolean }>(`/invitations/${encodeURIComponent(token)}`, { signal })
export const acceptInvitation = (
  token: string,
  input: { name: string; username: string; password: string },
) => post<User>(`/invitations/${encodeURIComponent(token)}/accept`, input)

export const listUsers = (signal: AbortSignal) => get<User[]>('/users', { signal })
export const disableUser = (id: string) =>
  post<User>(`/users/${encodeURIComponent(id)}/disable`)
export const enableUser = (id: string) => post<User>(`/users/${encodeURIComponent(id)}/enable`)
