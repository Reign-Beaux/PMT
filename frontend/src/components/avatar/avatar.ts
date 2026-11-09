export type AvatarTone =
  | 'neutral'
  | 'progress'
  | 'success'
  | 'warning'
  | 'danger'
  | 'discovery'

const TONES: readonly AvatarTone[] = [
  'neutral',
  'progress',
  'success',
  'warning',
  'danger',
  'discovery',
]

export function initials(name: string): string {
  return name
    .trim()
    .split(/[\s_-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => Array.from(part)[0].toUpperCase())
    .join('')
}

export function avatarTone(name: string): AvatarTone {
  let hash = 0
  for (const char of name) {
    hash = (hash * 31 + (char.codePointAt(0) ?? 0)) >>> 0
  }
  return TONES[hash % TONES.length]
}
