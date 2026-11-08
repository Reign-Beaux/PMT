import { useCallback } from 'react'
import { useSearchParams } from 'react-router'
import { parseSelection, toSearch, type DetailSelection } from './selection'

export function useDetailSelection() {
  const [searchParams, setSearchParams] = useSearchParams()
  const selection = parseSelection(searchParams)

  const select = useCallback(
    (next: DetailSelection) => {
      setSearchParams(toSearch(next, searchParams))
    },
    [searchParams, setSearchParams],
  )

  const clear = useCallback(() => {
    setSearchParams(toSearch(null, searchParams))
  }, [searchParams, setSearchParams])

  return { selection, select, clear }
}
