import type { BadgeTone } from '../components/card/Badge'
import type { ProjectRole } from '../api/projects'

export const ROLE_LABELS: Record<ProjectRole, string> = {
  owner: 'Owner',
  editor: 'Editor',
  viewer: 'Viewer',
}

export const ROLE_TONES: Record<ProjectRole, BadgeTone> = {
  owner: 'info',
  editor: 'success',
  viewer: 'neutral',
}

export const ROLE_HELP: Record<ProjectRole, string> = {
  owner: 'Administra los miembros y libera etapas o bugs tomados',
  editor: 'Trabaja en el proyecto con sus agentes',
  viewer: 'Solo consulta desde el sitio web',
}
