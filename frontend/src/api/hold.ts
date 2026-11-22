import { optionalString } from './validate'

/** Quién tiene tomada una etapa o un bug, y desde cuándo. Sin campos, nadie. */
export interface Hold {
  taken_by?: string
  taken_at?: string
}

export function readHold(record: Record<string, unknown>): Hold {
  const hold: Hold = {}
  const takenBy = optionalString(record, 'taken_by')
  const takenAt = optionalString(record, 'taken_at')
  if (takenBy !== undefined && takenBy !== '') hold.taken_by = takenBy
  if (takenAt !== undefined && takenAt !== '') hold.taken_at = takenAt
  return hold
}
