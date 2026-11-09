import { useOutletContext } from 'react-router'
import type { Project, Stage } from '../api/projects'
import type { Progress } from '../api/progress'
import type { Bug } from '../api/bugs'
import type { AsyncResult } from '../hooks/useAsync'

export interface ProjectContext {
  project: Project
  stages: AsyncResult<{ stages: Stage[]; progress: Progress }>
  bugs: AsyncResult<Bug[]>
}

export function useProjectContext() { return useOutletContext<ProjectContext>() }
