import type { AsyncResult } from '../hooks/useAsync'
import { ErrorState, Loading } from './feedback'

/** Un fallo de refresco se muestra sin ocultar los datos anteriores. */
export function QueryState<T>({ result, label }: { result: AsyncResult<T>; label: string }) {
  return <>
    {result.status === 'loading' && result.data === undefined && <Loading label={label} variant="list" />}
    {result.status === 'error' && <ErrorState message={result.error?.message ?? 'No se pudo cargar la información'} onRetry={result.retry} />}
  </>
}
