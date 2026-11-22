// Fixtures aislados del harness (tarea 20). Nombres e ids solo con [a-z0-9-]
// generados aquí: seguros como literales SQL. La limpieza borra solo lo creado
// por esta ejecución, incluso cuando la prueba falla.
import crypto from 'node:crypto'
import { test as base, type APIRequestContext, type BrowserContextOptions } from '@playwright/test'
import { psql } from './db'
import { API_TARGET, DB, RUN_ID, TARGET_LINE } from './env'
import { TEST_OWNER } from './owner'

export type HarnessProject = { id: string; name: string }
export type HarnessStage = { id: string; name: string; projectId: string; position: number }
export type HarnessTask = { id: string; title: string; stageId: string; status: string }
export type HarnessBug = { id: string; title: string; projectId: string; status: string }
export type HarnessComment = { id: string; itemKind: string; itemId: string; body: string }
export type HarnessStatusChange = { id: string }
/** Usuario extra creado por invitación, con su sesión del navegador. */
export type HarnessUser = { username: string; name: string; email: string; password: string; storageState: StorageState }
export type Harness = {
  createProject: (options?: { status?: string; productVersion?: string }) => HarnessProject
  /** takenBy: username de quien la tiene tomada (bloqueo de la fase 1d). */
  createStage: (projectId: string, options: { position: number; status?: string; takenBy?: string }) => HarnessStage
  createTask: (stageId: string, options: { status: string }) => HarnessTask
  createBug: (
    projectId: string,
    options: { status: string; productVersion?: string; takenBy?: string },
  ) => HarnessBug
  /** Segundo usuario: el dueño lo invita y él acepta por la API, como en la app. */
  createUser: (label: string) => Promise<HarnessUser>
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

// Con miembros (fase 1c), un proyecto solo lo ve quien es miembro: los que crea
// el harness por SQL quedan del dueño de pruebas, como si él los hubiera creado.
function ownedByTestOwner(projectId: string): string {
  return `insert into project_members (project_id, user_id, role) select '${projectId}', id, 'owner' from users where username = '${TEST_OWNER.username}'`
}

type StorageState = Exclude<BrowserContextOptions['storageState'], string | undefined>

// Columnas del bloqueo para quien tiene ese username; sin él, nadie.
function holdColumns(takenBy: string | undefined): { cols: string; values: string } {
  if (takenBy === undefined) return { cols: '', values: '' }
  return {
    cols: ', taken_by, taken_by_name, taken_at',
    values: `, (select id from users where username = '${takenBy}'), '${takenBy}', now()`,
  }
}

// Abre sesión con el dueño de pruebas; sin ella la API responde 401.
async function login(ctx: APIRequestContext, baseURL: string, who: { email: string; password: string } = TEST_OWNER) {
  const res = await ctx.post(`${baseURL}/api/auth/login`, {
    data: { email: who.email, password: who.password },
  })
  if (!res.ok()) {
    throw new Error(`No se pudo entrar con el dueño de pruebas (${res.status()}): ${await res.text()}`)
  }
}

export const test = base.extend<{ harness: Harness }, { harnessTarget: void; harnessSession: StorageState }>({
  // Sesión del navegador: login por el proxy de Vite para que la cookie quede
  // en localhost:5173, y cada página arranca con ella.
  harnessSession: [
    async ({ playwright }, provide) => {
      const ctx = await playwright.request.newContext()
      try {
        await login(ctx, 'http://localhost:5173')
        await provide(await ctx.storageState())
      } finally {
        await ctx.dispose()
      }
    },
    { scope: 'worker' },
  ],
  storageState: async ({ harnessSession }, provide) => {
    await provide(harnessSession)
  },

  // Centinela de worker: confirma que el backend en 8090 usa pmt_frontend_test
  // antes de crear ningún dato de prueba. El callback de Playwright se llama
  // "provide" (no "use") para no confundir al linter de hooks de React.
  harnessTarget: [
    async ({ playwright }, provide) => {
      const id = crypto.randomUUID()
      sentinelId = id
      psql(`insert into projects (id, name) values ('${id}', '${SENTINEL_NAME}'); ${ownedByTestOwner(id)}`)
      try {
        const ctx = await playwright.request.newContext()
        try {
          await login(ctx, API_TARGET)
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

  harness: async ({ harnessTarget, playwright }, provide, testInfo) => {
    void harnessTarget // ordena el centinela antes que los datos de la prueba
    const usuarios: HarnessUser[] = []
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
            `insert into projects (id, name, status, product_version) values ('${project.id}', '${project.name}', '${status}', '${version}'); ${ownedByTestOwner(project.id)}`,
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
          const hold = holdColumns(options.takenBy)
          psql(
            `insert into stages (id, project_id, name, position, status${hold.cols}) values ('${stage.id}', '${projectId}', '${stage.name}', ${stage.position}, '${status}'${hold.values})`,
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
          const hold = holdColumns(options.takenBy)
          psql(
            `insert into bugs (id, project_id, title, description, steps_to_reproduce, product_version, status${hold.cols}) values ('${bug.id}', '${projectId}', '${bug.title}', 'descripcion del bug', 'pasos para reproducir', '${version}', '${bug.status}'${hold.values})`,
          )
          bugs.push(bug)
          return bug
        },
        async createUser(label) {
          const username = `e2e-${RUN_ID}-${label}`
          const user = {
            username,
            name: `Prueba ${label}`,
            email: `${username}@pmt.test`,
            password: crypto.randomBytes(12).toString('hex'),
          }
          const owner = await playwright.request.newContext()
          let token = ''
          try {
            await login(owner, 'http://localhost:5173')
            const res = await owner.post('http://localhost:5173/api/invitations', { data: { email: user.email } })
            if (!res.ok()) throw new Error(`No se pudo invitar a ${username} (${res.status()})`)
            const url: string = (await res.json()).url
            token = url.slice(url.lastIndexOf('/') + 1)
          } finally {
            await owner.dispose()
          }
          const ctx = await playwright.request.newContext()
          try {
            const res = await ctx.post(`http://localhost:5173/api/invitations/${token}/accept`, {
              data: { name: user.name, username, password: user.password },
            })
            if (!res.ok()) throw new Error(`No se pudo aceptar la invitación de ${username} (${res.status()})`)
            const created: HarnessUser = { ...user, storageState: await ctx.storageState() }
            usuarios.push(created)
            return created
          } finally {
            await ctx.dispose()
          }
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
      // Liberar escribe un registro de status sin llave foránea: se borran los
      // de las etapas y bugs de esta ejecución junto con los del harness.
      const conHistorial = etapas.length + bugs.length > 0
      if (comentarios.length + cambiosTarea.length + cambiosBug.length > 0 || conHistorial) {
        const ids = (filas: { id: string }[]) => filas.map((f) => `'${f.id}'`).join(', ')
        const tablas = ['comments', 'task_status_changes', 'bug_status_changes', 'stage_status_changes']
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
          etapas.length > 0
            ? `DELETE FROM stage_status_changes WHERE stage_id IN (${ids(etapas)});`
            : '',
          bugs.length > 0
            ? `DELETE FROM bug_status_changes WHERE bug_id IN (${ids(bugs)});`
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
      // Sesiones, tokens y membresías caen en cascada con el usuario.
      for (const user of usuarios) {
        psql(`delete from invitations where email = '${user.email}'; delete from users where username = '${user.username}'`)
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
      const restoUsuarios = Number(
        psql(`select (select count(*) from users where username like 'e2e-${RUN_ID}-%') + (select count(*) from invitations where email like 'e2e-${RUN_ID}-%')`),
      )
      const restantes =
        restoProjects + restoStages + restoTasks + restoBugs + restoComments + restoCambios + restoUsuarios
      if (restantes !== 0) {
        // Intencional: el teardown debe fallar la prueba si la limpieza queda
        // incompleta, aunque eso reemplace el error original del test.
        // eslint-disable-next-line no-unsafe-finally
        throw new Error(
          `Limpieza incompleta: quedan ${restoProjects} projects, ${restoStages} stages, ${restoTasks} tasks, ${restoBugs} bugs, ${restoComments} comments, ${restoCambios} cambios y ${restoUsuarios} usuarios o invitaciones e2e-${RUN_ID}-… en pmt_frontend_test`,
        )
      }
      const linea = `${TARGET_LINE} · creados ${creados.length} · restantes 0`
      console.log(linea)
      testInfo.annotations.push({ type: 'destino', description: linea })
    }
  },
})

export { API_TARGET, DB, RUN_ID }
