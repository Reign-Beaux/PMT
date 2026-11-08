import { useState } from 'react'
import { Outlet, useParams } from 'react-router'
import { getProject } from '../api/projects'
import { useAsync } from '../hooks/useAsync'
import { PROJECT_RESOURCES } from '../realtime/refreshResources'
import { ProjectHeader } from './ProjectHeader'
import { Shell } from './Shell'

export function ShellLayout() {
  const { projectId } = useParams()
  if (projectId === undefined) {
    return (
      <Shell>
        <Outlet />
      </Shell>
    )
  }
  return <ProjectShell projectId={projectId} />
}

function ProjectShell({ projectId }: { projectId: string }) {
  const [stageName, setStageName] = useState<string | undefined>(undefined)
  const { status, data, error, retry } = useAsync(
    (signal) => getProject(projectId, { signal }),
    [projectId],
    { refreshOn: PROJECT_RESOURCES },
  )
  return (
    <Shell projectId={projectId} projectName={data?.name} stageName={stageName}>
      <ProjectHeader
        status={status}
        project={data}
        error={error}
        onRetry={retry}
      />
      {/* El Outlet sigue montado mientras haya datos, aunque un refresco falle. */}
      {data !== undefined && <Outlet context={{ setStageName }} />}
    </Shell>
  )
}
