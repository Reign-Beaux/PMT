// Fixtures aislados del harness (tarea 20). Nombres e ids solo con [a-z0-9-]
// generados aquí: seguros como literales SQL. La limpieza borra solo lo creado
// por esta ejecución, incluso cuando la prueba falla.
import crypto from 'node:crypto'
import { test as base } from '@playwright/test'
import { ensureSchema, psql } from './db'
import { API_TARGET, DB, RUN_ID, TARGET_LINE } from './env'

export type HarnessProject = { id: string; name: string }
export type HarnessStage = { id: string; name: string; projectId: string; position: number }
export type HarnessTask = { id: string; title: string; stageId: string; status: string }
export type HarnessBug = { id: string; title: string; projectId: string; status: string }
export type HarnessComment = { id: string; itemKind: string; itemId: string; body: string }
export type HarnessStatusChange = { id: string }
export type Harness = {
  createProject: (options?: { status?: string; productVersion?: string }) => HarnessProject
  createStage: (projectId: string, options: { position: number; status?: string }) => HarnessStage
  createTask: (stageId: string, options: { status: string }) => HarnessTask
  createBug: (
    projectId: string,
    options: { status: string; productVersion?: string },
  ) => HarnessBug
  createComment: (itemKind: 'task' | 'bug', itemId: string, body: string) => HarnessComment
  createTaskStatusChange: (
    taskId: string,
    from: string,
    to: string,
    reason?: string,
  ) => HarnessStatusChange
  createBugStatusChange: (
    bugId: string,
    from: string,
    to: string,
    reason?: string,
  ) => HarnessStatusChange
}

const ACTOR = `e2e-${RUN_ID}-actor`
const ACTOR_PROFILE = 'QA'

const SENTINEL_NAME = `e2e-${RUN_ID}-centinela`
let sentinelId = ''

export const test = base.extend<{ harness: Harness }, { harnessTarget: void }>({
  // Centinela de worker: confirma que el backend en 8090 usa pmt_frontend_test
  // antes de crear ningún dato de prueba. El callback de Playwright se llama
  // "provide" (no "use") para no confundir al linter de hooks de React.
  harnessTarget: [
    async ({ playwright }, provide) => {
      ensureSchema()
      const id = crypto.randomUUID()
      sentinelId = id
      psql(`insert into projects (id, name) values ('${id}', '${SENTINEL_NAME}')`)
      try {
        const ctx = await playwright.request.newContext()
        try {
          const res = await ctx.get(`${API_TARGET}/api/projects/${id}`)
          const body = res.ok() ? await res.json() : null
          if (!res.ok() || body?.name !== SENTINEL_NAME) {
            throw new Error('El backend en 8090 no usa la base pmt_frontend_test')
          }
        } finally {
          await ctx.dispose()
        }
        await provide()
      } finally {
        psql(`delete from projects where id = '${id}'`)
        sentinelId = ''
      }
    },
    { scope: 'worker', auto: true },
  ],

  harness: async ({ harnessTarget }, provide, testInfo) => {
    void harnessTarget // ordena el centinela antes que los datos de la prueba
    const creados: HarnessProject[] = []
    const etapas: HarnessStage[] = []
    const tareas: HarnessTask[] = []
    const bugs: HarnessBug[] = []
    const comentarios: HarnessComment[] = []
    const cambiosTarea: HarnessStatusChange[] = []
    const cambiosBug: HarnessStatusChange[] = []
    try {
      await provide({
        createProject(options = {}) {
          const project: HarnessProject = {
            id: crypto.randomUUID(),
            name: `e2e-${RUN_ID}-${creados.length + 1}`,
          }
          const status = options.status ?? 'planning'
          const version = options.productVersion ?? ''
          psql(
            `insert into projects (id, name, status, product_version) values ('${project.id}', '${project.name}', '${status}', '${version}')`,
          )
          creados.push(project)
          return project
        },
        createStage(projectId, options) {
          const stage: HarnessStage = {
            id: crypto.randomUUID(),
            projectId,
            position: options.position,
            name: `e2e-${RUN_ID}-etapa-${etapas.length + 1}`,
          }
          const status = options.status ?? 'planning'
          psql(
            `insert into stages (id, project_id, name, position, status) values ('${stage.id}', '${projectId}', '${stage.name}', ${stage.position}, '${status}')`,
          )
          etapas.push(stage)
          return stage
        },
        createTask(stageId, options) {
          const task: HarnessTask = {
            id: crypto.randomUUID(),
            stageId,
            status: options.status,
            title: `e2e-${RUN_ID}-tarea-${tareas.length + 1}`,
          }
          psql(
            `insert into tasks (id, stage_id, title, status) values ('${task.id}', '${stageId}', '${task.title}', '${task.status}')`,
          )
          tareas.push(task)
          return task
        },
        createBug(projectId, options) {
          const bug: HarnessBug = {
            id: crypto.randomUUID(),
            projectId,
            status: options.status,
            title: `e2e-${RUN_ID}-bug-${bugs.length + 1}`,
          }
          const version = options.productVersion ?? '1.0.0'
          psql(
            `insert into bugs (id, project_id, title, description, steps_to_reproduce, product_version, status) values ('${bug.id}', '${projectId}', '${bug.title}', 'descripcion del bug', 'pasos para reproducir', '${version}', '${bug.status}')`,
          )
          bugs.push(bug)
          return bug
        },
        createComment(itemKind, itemId, body) {
          const comment: HarnessComment = { id: crypto.randomUUID(), itemKind, itemId, body }
          psql(
            `insert into comments (id, item_kind, item_id, body, actor_name, actor_profile) values ('${comment.id}', '${itemKind}', '${itemId}', '${body}', '${ACTOR}', '${ACTOR_PROFILE}')`,
          )
          comentarios.push(comment)
          return comment
        },
        createTaskStatusChange(taskId, from, to, reason = '') {
          const cambio: HarnessStatusChange = { id: crypto.randomUUID() }
          psql(
            `insert into task_status_changes (id, task_id, from_status, to_status, actor_name, actor_profile, reason) values ('${cambio.id}', '${taskId}', '${from}', '${to}', '${ACTOR}', '${ACTOR_PROFILE}', '${reason}')`,
          )
          cambiosTarea.push(cambio)
          return cambio
        },
        createBugStatusChange(bugId, from, to, reason = '') {
          const cambio: HarnessStatusChange = { id: crypto.randomUUID() }
          psql(
            `insert into bug_status_changes (id, bug_id, from_status, to_status, actor_name, actor_profile, reason) values ('${cambio.id}', '${bugId}', '${from}', '${to}', '${ACTOR}', '${ACTOR_PROFILE}', '${reason}')`,
          )
          cambiosBug.push(cambio)
          return cambio
        },
      })
    } finally {
      // comments, task_status_changes y bug_status_changes son inmutables: el
      // trigger se desactiva solo durante el borrado y se reactiva al cerrar.
      if (comentarios.length + cambiosTarea.length + cambiosBug.length > 0) {
        const ids = (filas: { id: string }[]) => filas.map((f) => `'${f.id}'`).join(', ')
        const tablas = ['comments', 'task_status_changes', 'bug_status_changes']
        const desactivar = tablas.map(
          (t) => `ALTER TABLE ${t} DISABLE TRIGGER ${t}_immutable;`,
        ).join('\n')
        const activar = tablas.map(
          (t) => `ALTER TABLE ${t} ENABLE TRIGGER ${t}_immutable;`,
        ).join('\n')
        const borrados = [
          comentarios.length > 0 ? `DELETE FROM comments WHERE id IN (${ids(comentarios)});` : '',
          cambiosTarea.length > 0
            ? `DELETE FROM task_status_changes WHERE id IN (${ids(cambiosTarea)});`
            : '',
          cambiosBug.length > 0
            ? `DELETE FROM bug_status_changes WHERE id IN (${ids(cambiosBug)});`
            : '',
        ].filter(Boolean).join('\n')
        psql(`BEGIN;\n${desactivar}\n${borrados}\n${activar}\nCOMMIT;`)
      }
      // Las FK son RESTRICT: borrar en orden bugs y tasks → stages → projects.
      for (const bug of bugs) {
        psql(`delete from bugs where id = '${bug.id}'`)
      }
      for (const task of tareas) {
        psql(`delete from tasks where id = '${task.id}'`)
      }
      for (const stage of etapas) {
        psql(`delete from stages where id = '${stage.id}'`)
      }
      for (const project of creados) {
        psql(`delete from projects where id = '${project.id}'`)
      }
      // El centinela sigue vivo hasta el teardown del worker: se excluye del conteo.
      const filtro = sentinelId === '' ? '' : ` and id <> '${sentinelId}'`
      const restoProjects = Number(
        psql(`select count(*) from projects where name like 'e2e-${RUN_ID}-%'${filtro}`),
      )
      const restoStages = Number(
        psql(`select count(*) from stages where name like 'e2e-${RUN_ID}-%'`),
      )
      const restoTasks = Number(
        psql(`select count(*) from tasks where title like 'e2e-${RUN_ID}-%'`),
      )
      const restoBugs = Number(
        psql(`select count(*) from bugs where title like 'e2e-${RUN_ID}-%'`),
      )
      const restoComments = Number(
        psql(`select count(*) from comments where actor_name like 'e2e-${RUN_ID}-%'`),
      )
      const restoCambios = Number(
        psql(`select
          (select count(*) from task_status_changes where actor_name like 'e2e-${RUN_ID}-%')
          + (select count(*) from bug_status_changes where actor_name like 'e2e-${RUN_ID}-%')`),
      )
      const restantes =
        restoProjects + restoStages + restoTasks + restoBugs + restoComments + restoCambios
      if (restantes !== 0) {
        // Intencional: el teardown debe fallar la prueba si la limpieza queda
        // incompleta, aunque eso reemplace el error original del test.
        // eslint-disable-next-line no-unsafe-finally
        throw new Error(
          `Limpieza incompleta: quedan ${restoProjects} projects, ${restoStages} stages, ${restoTasks} tasks, ${restoBugs} bugs, ${restoComments} comments y ${restoCambios} cambios e2e-${RUN_ID}-… en pmt_frontend_test`,
        )
      }
      const linea = `${TARGET_LINE} · creados ${creados.length} · restantes 0`
      console.log(linea)
      testInfo.annotations.push({ type: 'destino', description: linea })
    }
  },
})

export { API_TARGET, DB, RUN_ID }
