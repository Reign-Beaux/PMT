import { useParams } from 'react-router'
import { StageList } from './StageList'

export function ProjectView() {
  const { projectId } = useParams()
  if (projectId === undefined) return null
  return <StageList projectId={projectId} />
}
