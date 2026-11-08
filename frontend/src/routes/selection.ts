export type DetailSelection =
  | { kind: 'task'; id: string }
  | { kind: 'bug'; id: string }

export function parseSelection(
  params: URLSearchParams,
): DetailSelection | null {
  const task = params.get('task')
  const bug = params.get('bug')
  if (task !== null && bug !== null) return null
  if (task) return { kind: 'task', id: task }
  if (bug) return { kind: 'bug', id: bug }
  return null
}

export function toSearch(
  selection: DetailSelection | null,
  current: URLSearchParams = new URLSearchParams(),
): string {
  const params = new URLSearchParams(current)
  params.delete('task')
  params.delete('bug')
  if (selection) params.set(selection.kind, selection.id)
  return params.toString()
}
