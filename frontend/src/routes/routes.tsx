import { Navigate, type RouteObject } from 'react-router'
import { AppShell } from '../shell/AppShell'
import { ProjectLayout } from '../shell/ProjectLayout'
import { DashboardView } from '../views/DashboardView'
import { ProjectsView } from '../views/ProjectsView'
import { StageList } from '../views/StageList'
import { StageView } from '../views/StageView'
import { BugsView } from '../views/BugsView'
import { NotFoundView } from '../views/NotFoundView'
import { ComponentsDemoView } from '../views/ComponentsDemoView'
import { RequireAuth } from '../shell/RequireAuth'
import { LoginView } from '../views/LoginView'
import { InvitationView } from '../views/InvitationView'
import { AccountView } from '../views/AccountView'

export const routes: RouteObject[] = [
  { path: '/login', element: <LoginView /> },
  { path: '/invitacion/:token', element: <InvitationView /> },
  { element: <RequireAuth />, children: [{
  element: <AppShell />,
  children: [
    { path: '/', element: <Navigate to="/projects" replace /> },
    { path: '/dashboard', element: <DashboardView /> },
    { path: '/projects', element: <ProjectsView /> },
    { path: '/projects/:projectId', element: <ProjectLayout />, children: [
      { index: true, element: <StageList /> },
      { path: 'bugs', element: <BugsView /> },
      { path: 'stages/:stageId', element: <StageView /> },
    ] },
    { path: '/cuenta', element: <AccountView /> },
    ...(import.meta.env.DEV ? [{ path: '/dev/componentes', element: <ComponentsDemoView /> }] : []),
    { path: '*', element: <NotFoundView /> },
  ],
  }] },
]
