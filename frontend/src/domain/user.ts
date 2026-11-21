export interface User {
  id: string
  email: string
  username: string
  name: string
  is_instance_owner: boolean
  disabled: boolean
  created_at: string
}

export interface ApiToken {
  id: string
  name: string
  created_at: string
  last_used_at: string | null
}

export interface NewApiToken {
  id: string
  name: string
  created_at: string
  token: string
}

export interface Invitation {
  id: string
  email: string
  expires_at: string
  accepted_at: string | null
}

export interface NewInvitation {
  id: string
  email: string
  url: string
  expires_at: string
}

export type InvitationState = 'pendiente' | 'aceptada' | 'vencida'

export function invitationState(invitation: Invitation, now: Date): InvitationState {
  if (invitation.accepted_at !== null) return 'aceptada'
  return new Date(invitation.expires_at).getTime() <= now.getTime() ? 'vencida' : 'pendiente'
}

/** Mismo formato que valida el backend: el username no cambia. */
export const USERNAME_PATTERN = /^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$/
