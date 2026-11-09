const formatter = new Intl.DateTimeFormat('es', {
  dateStyle: 'medium',
  timeStyle: 'short',
})

/** Formatea una fecha ISO en español; si no es válida devuelve el texto original. */
export function formatDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return formatter.format(date)
}

const pad2 = (value: number): string => String(value).padStart(2, '0')

/**
 * Formatea una fecha ISO como dd/mm/aaaa en hora local, con relleno de ceros;
 * si no es válida devuelve el texto original (misma regla que formatDate).
 */
export function formatShortDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return `${pad2(date.getDate())}/${pad2(date.getMonth() + 1)}/${date.getFullYear()}`
}
