import type { Page, WebSocketRoute } from '@playwright/test'
import type { Project, Stage } from '../../src/api/projects'
import type { Task } from '../../src/api/tasks'
import type { Bug } from '../../src/api/bugs'
import type { Comment } from '../../src/api/comments'
import type { TaskHistory } from '../../src/api/history'
import type { ChangeEventResource } from '../../src/api/events'

const date = '2026-11-07T16:00:00-06:00'
export const ids = { project: '00000000-0000-4000-8000-000000000001', stage: '00000000-0000-4000-8000-000000000002', bug: '00000000-0000-4000-8000-000000000003' }
export const project: Project = { id: ids.project, name: 'PMT', description: 'Gestión de proyectos y coordinación de agentes.\nDescripción del proyecto en una segunda línea.', status: 'working', product_version: '1.2.0', created_at: date, updated_at: date }
export const stage: Stage = { id: ids.stage, project_id: ids.project, name: 'Frontend — Diseño de interfaz', description: 'Diseño de las vistas de proyectos, etapas y tareas.\nMantener los contratos y el comportamiento de consulta.', position: 8, status: 'in_dev', lead: 'pmt-frontend-tl', members: ['pmt-frontend-devfront'], created_at: date, updated_at: date }

export async function mockApi(page: Page) {
  const tasks: Task[] = Array.from({ length: 24 }, (_, index) => ({ id: `tarea-${index + 1}`, stage_id: ids.stage, title: `Tarea ${String(index + 1).padStart(2, '0')} — Validación de interfaz`, description: ('Descripción técnica de la tarea.\nEl contenido conserva sus saltos de línea y se muestra como texto.\n\n').repeat(9), status: index === 11 ? 'in_review' : index === 23 ? 'cancelled' : 'done', assignee: 'pmt-frontend-devfront', created_at: date, updated_at: date, done_at: index === 11 || index === 23 ? undefined : date }))
  const bugs: Bug[] = [{ id: ids.bug, project_id: ids.project, title: 'Desbordamiento en móvil', description: 'Una descripción extensa desborda la pantalla.', steps_to_reproduce: 'Abrir una etapa.\nConsultar su descripción en móvil.', status: 'reported', product_version: '1.2.0', reported_by: 'pmt-frontend-qa', created_at: date, updated_at: date }]
  const comments: Comment[] = [
    { id: 'c2', item_kind: 'task', item_id: 'tarea-12', body: 'Segundo comentario.', actor_name: 'pmt-frontend-tl', actor_profile: 'TL', created_at: '2026-11-08T10:00:00-06:00' },
    { id: 'c1', item_kind: 'task', item_id: 'tarea-12', body: '<script>literal</script>\nPrimera evidencia.', actor_name: 'pmt-frontend-devfront', actor_profile: 'DevFrontend', created_at: date },
  ]
  const history: TaskHistory[] = [
    { id: 'h2', task_id: 'tarea-12', from_status: 'in_dev', to_status: 'in_review', actor_name: 'pmt-frontend-devfront', actor_profile: 'DevFrontend', reason: 'Revisión solicitada.\nEvidencia registrada.', changed_at: date },
    { id: 'h1', task_id: 'tarea-12', from_status: 'ready', to_status: 'in_dev', actor_name: 'pmt-frontend-devfront', actor_profile: 'DevFrontend', reason: '', changed_at: '2026-11-06T10:00:00-06:00' },
  ]
  const state = { projects: [{ ...project }], stages: [{ ...stage }], tasks, bugs, comments, history, failProjects: 0 }
  const sockets = new Set<WebSocketRoute>()
  let connections = 0
  await page.routeWebSocket('**/api/events', socket => {
    connections += 1
    sockets.add(socket)
    socket.onClose(() => sockets.delete(socket))
    socket.send(JSON.stringify({ type: 'resync' }))
  })
  await page.route('**/api/**', async route => {
    const url = new URL(route.request().url())
    if (!url.pathname.startsWith('/api/')) { await route.fallback(); return }
    const pathname = url.pathname
    let data: unknown
    if (pathname === '/api/projects') {
      if (state.failProjects > 0) { state.failProjects -= 1; await route.fulfill({ status: 503, json: { error: 'Servicio temporalmente no disponible' } }); return }
      data = state.projects
    } else if (pathname === `/api/projects/${ids.project}`) data = state.projects.find(item => item.id === ids.project)
    else if (pathname === `/api/projects/${ids.project}/stages`) data = state.stages
    else if (pathname === `/api/projects/${ids.project}/bugs`) data = state.bugs
    else if (pathname === `/api/stages/${ids.stage}/tasks`) data = state.tasks
    else if (pathname === `/api/projects/${ids.project}/progress`) {
      const counts: Record<string, number> = {}
      for (const task of state.tasks) counts[task.status] = (counts[task.status] ?? 0) + 1
      const open = state.bugs.filter(bug => bug.status !== 'done' && bug.status !== 'rejected')
      data = { project: { id: ids.project, name: project.name, status: project.status, last_change_at: date }, stages: state.stages.map(item => ({ id: item.id, name: item.name, status: item.status, lead: item.lead, members: item.members ?? [], counts, tasks: state.tasks })), open_bugs: { counts: { reported: open.length }, items: open }, signals: [] }
    } else if (pathname === '/api/comments') data = state.comments.filter(comment => comment.item_kind === url.searchParams.get('item_kind') && comment.item_id === url.searchParams.get('item_id'))
    else if (/^\/api\/tasks\/[^/]+\/status-changes$/.test(pathname)) data = state.history.filter(change => change.task_id === pathname.split('/')[3])
    else if (/^\/api\/tasks\/[^/]+$/.test(pathname)) data = state.tasks.find(task => task.id === pathname.split('/')[3])
    else if (/^\/api\/bugs\/[^/]+\/status-changes$/.test(pathname)) data = []
    if (data === undefined) await route.fulfill({ status: 404, json: { error: 'No encontrado' } })
    else await route.fulfill({ json: data })
  })
  return { state, send: (resource: ChangeEventResource) => { for (const socket of sockets) socket.send(JSON.stringify({ type: 'invalidate', resource })) }, connections: () => connections }
}
