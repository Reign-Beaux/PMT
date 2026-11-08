import type { GetOptions } from './http'
import { get } from './http'
import { requireArray, requireOneOf, requireRecord, requireString } from './validate'

export interface Comment {
  id: string
  item_kind: 'project' | 'stage' | 'task' | 'bug'
  item_id: string
  body: string
  actor_name: string
  actor_profile: string
  created_at: string
}

const ITEM_KINDS = ['project', 'stage', 'task', 'bug'] as const

function validateComment(value: unknown): Comment {
  const record = requireRecord(value)
  return {
    id: requireString(record, 'id'),
    item_kind: requireOneOf(record, 'item_kind', ITEM_KINDS),
    item_id: requireString(record, 'item_id'),
    body: requireString(record, 'body'),
    actor_name: requireString(record, 'actor_name'),
    actor_profile: requireString(record, 'actor_profile'),
    created_at: requireString(record, 'created_at'),
  }
}

/** El render posterior debe mostrar body como texto plano; aquí solo se entrega tal cual. */
export async function getComments(
  itemKind: 'task' | 'bug',
  itemId: string,
  options?: GetOptions,
): Promise<Comment[]> {
  const query = new URLSearchParams({ item_kind: itemKind, item_id: itemId })
  const data = await get<unknown>(`/comments?${query}`, options)
  return requireArray(data).map(validateComment)
}
