import type { RouteObject } from 'react-router'
import { ShellLayout } from '../shell/ShellLayout'
import { ProjectsView } from '../views/ProjectsView'
import { ProjectView } from '../views/ProjectView'
import { StageView } from '../views/StageView'
import { BugsView } from '../views/BugsView'
import { NotFoundView } from '../views/NotFoundView'
import { ComponentsDemoView } from '../views/ComponentsDemoView'

const devRoutes: RouteObject[] = import.meta.env.DEV
  ? [{ path: '/dev/componentes', element: <ComponentsDemoView /> }]
  : []

export const routes: RouteObject[] = [
  {
    element: <ShellLayout />,
    children: [
      { path: '/', element: <ProjectsView /> },
      { path: '/projects/:projectId', element: <ProjectView /> },
      { path: '/projects/:projectId/stages/:stageId', element: <StageView /> },
      { path: '/projects/:projectId/bugs', element: <BugsView /> },
      { path: '*', element: <NotFoundView /> },
    ],
  },
  ...devRoutes,
]
