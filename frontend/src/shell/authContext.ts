import { createContext, useContext } from 'react'
import type { User } from '../domain/user'

export const AuthContext = createContext<User | null>(null)

/** Usuario con sesión. Solo dentro de RequireAuth. */
export function useCurrentUser(): User {
  const user = useContext(AuthContext)
  if (user === null) throw new Error('Usuario no disponible fuera de RequireAuth')
  return user
}
