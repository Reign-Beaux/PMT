import { useOutletContext } from 'react-router'

export interface StageNameContext {
  setStageName: (name: string | undefined) => void
}

/** Setter del nombre de etapa que el shell muestra; lo expone el Outlet de ProjectShell. */
export function useStageName(): (name: string | undefined) => void {
  return useOutletContext<StageNameContext>().setStageName
}
