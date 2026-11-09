import { avatarTone, initials } from './avatar'
import styles from './Avatar.module.scss'

export interface AvatarProps {
  name?: string
  decorative?: boolean
}

export function Avatar({ name, decorative = false }: AvatarProps) {
  const accessible = decorative
    ? { 'aria-hidden': true as const }
    : undefined
  if (name !== undefined && name.trim() !== '') {
    return (
      <span
        {...(accessible ?? {
          role: 'img',
          'aria-label': `Responsable: ${name}`,
          title: `Responsable: ${name}`,
        })}
        className={`${styles.avatar} ${styles[avatarTone(name)]}`}
      >
        {initials(name)}
      </span>
    )
  }
  return (
    <span
      {...(accessible ?? {
        role: 'img',
        'aria-label': 'Sin asignar',
        title: 'Sin asignar',
      })}
      className={`${styles.avatar} ${styles.empty}`}
    />
  )
}
