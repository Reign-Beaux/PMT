import { useState, type FormEvent } from 'react'
import { createToken, listTokens, revokeToken } from '../api/auth'
import { QueryState } from '../components/QueryState'
import { Secret } from '../components/Secret'
import { connectCommands } from '../domain/connectCommands'
import { formatDate } from '../domain/formatDate'
import type { NewApiToken } from '../domain/user'
import { useAsync } from '../hooks/useAsync'
import { Icon } from '../components/Icon'
import { PageBreadcrumbs } from '../shell/PageBreadcrumbs'
import app from '../styles/App.module.scss'
import s from './Connect.module.scss'

const SECTIONS = [
  ['tokens', 'Tokens de API'], ['cli', 'Instalar el CLI'], ['mcp', 'Conectar el MCP'],
  ['pp', 'Levantar a PP'], ['perfiles', 'Perfiles'],
] as const
const TEAM_COMMANDS = [
  ['pmt avisar <agente> "mensaje"', 'Envía un aviso y confirma que llegó. Si no se confirma, lo anota en .pmt/para-usuario.md.'],
  ['pmt levantar <agente> "encargo"', 'Reinicia al agente con un encargo. Con TL también arranca el vigía.'],
  ['pmt reiniciar <agente>', 'Abre una sesión vacía; espera si el agente sigue trabajando.'],
  ['pmt relanzar-dp <medium|high>', 'Relanza a DP con ese effort y guarda la elección.'],
  ['pmt vigia', 'Avisa al TL tras 150 segundos de inactividad. Termina si hay un bloqueo o un mensaje para ti.'],
] as const
const CLAUDE_DEFAULT = 'Claude · Opus · medium'
const CODEX_DEFAULT = 'Codex · gpt-6.1-terra · high'
const PROFILES = [
  { id: 'PP', name: 'Project Planning', description: 'Platica contigo y convierte la idea en etapas y tareas con entregables verificables. Planea qué se va a construir, registra bugs y mantiene la versión del proyecto.', when: 'Al planear el proyecto, ampliar su alcance o registrar un bug.', model: 'Lo eliges al lanzar: Claude / opus o Codex / gpt-6.1-terra; effort medium o high.' },
  { id: 'PC', name: 'Project Coordinator', description: 'Te explica qué está haciendo el equipo leyendo sus pestañas de herdr. Solo consulta: no modifica PMT ni da instrucciones a los demás agentes.', when: 'Cuando necesitas consultar el avance sin interrumpir a PP.', model: CODEX_DEFAULT },
  { id: 'TL', name: 'Tech Lead', description: 'Dirige una etapa a la vez: arma el equipo, inicia la etapa y coordina las entregas y correcciones. Cierra tareas y, al cerrar la etapa, hace commit y push.', when: 'Desde el inicio hasta el cierre de cada etapa.', model: CLAUDE_DEFAULT },
  { id: 'DP', name: 'Development Planning', description: 'Lee el código existente y planea cómo implementar un grupo de tareas o corregir un bug. Define criterios de aceptación y quién los comprueba; no escribe código.', when: 'Antes de que los desarrolladores empiecen a implementar.', model: CLAUDE_DEFAULT },
  { id: 'DevBackend', name: 'Developer Backend', description: 'Implementa el backend siguiendo el plan y ejecuta sus pruebas unitarias y linter. Entrega evidencia y corrige lo que encuentren CR o QA.', when: 'Después del plan de DP, para las tareas de backend.', model: CODEX_DEFAULT },
  { id: 'DevFrontend', name: 'Developer Frontend', description: 'Implementa la interfaz siguiendo el plan y entrega evidencia de sus criterios. Corrige lo que devuelvan CR o QA; QA se encarga de verificar la pantalla en el navegador.', when: 'Después del plan de DP, para las tareas de frontend.', model: CODEX_DEFAULT },
  { id: 'CR', name: 'Code Reviewer', description: 'Lee el cambio y lo contrasta con el plan para revisar lógica, integridad y calidad. Registra correcciones concretas o aprueba; no ejecuta pruebas ni builds.', when: 'Cuando Dev entrega el código y después de las correcciones.', model: CLAUDE_DEFAULT },
  { id: 'QAP', name: 'Quality Assurance Planning', description: 'Diseña los casos de prueba de una funcionalidad que afecta frontend y backend, incluidos errores y rechazos del servidor. Envía el plan al TL para su aprobación.', when: 'Después de CR, si la funcionalidad toca frontend y backend.', model: CODEX_DEFAULT },
  { id: 'QA', name: 'Quality Assurance', description: 'Prueba los criterios asignados y los efectos del cambio, y revisa los datos en la base. Registra hallazgos o aprueba con evidencia; también confirma o rechaza bugs.', when: 'Después de CR o del plan QAP aprobado, antes del cierre del TL.', model: CODEX_DEFAULT },
] as const
const FLOW = [
  ['PP', 'Planea'], ['TL', 'Inicia la etapa'], ['DP', 'Planea el cómo'],
  ['Dev', 'Implementa'], ['CR', 'Revisa'], ['QAP', 'Si toca frontend y backend'],
  ['QA', 'Prueba'], ['TL', 'Cierra'],
] as const

function Command({ value, label }: { value: string; label: string }) {
  const [status, setStatus] = useState('')
  async function copy() {
    try {
      await navigator.clipboard.writeText(value)
      setStatus('Copiado')
    } catch {
      setStatus('No se pudo copiar. Selecciona el texto y cópialo manualmente.')
    }
  }
  return <div className={s.command}>
    <pre aria-label={label} tabIndex={0}><code>{value}</code></pre>
    <button type="button" className={s.secondary} onClick={copy} aria-label={`Copiar ${label}`}>{status === 'Copiado' ? 'Copiado' : 'Copiar'}</button>
    <span className={s.copyStatus} role="status">{status}</span>
  </div>
}

export function ConnectView() {
  const tokens = useAsync(listTokens, [])
  const [name, setName] = useState('')
  const [created, setCreated] = useState<NewApiToken | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const commands = connectCommands(window.location.origin, created?.token)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setError('')
    if (!name.trim()) { setError('Escribe un nombre para el token, por ejemplo «laptop»'); return }
    setBusy(true)
    try {
      setCreated(await createToken(name.trim()))
      setName('')
      tokens.retry()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ocurrió un error inesperado')
    } finally { setBusy(false) }
  }

  async function revoke(id: string, tokenName: string) {
    if (!window.confirm(`¿Revocar el token «${tokenName}»? Los agentes que lo usan dejarán de entrar.`)) return
    setError('')
    try {
      await revokeToken(id)
      if (created?.id === id) setCreated(null)
      tokens.retry()
    } catch (err) { setError(err instanceof Error ? err.message : 'Ocurrió un error inesperado') }
  }
  return <>
    <PageBreadcrumbs items={[{ label: 'Conectar' }]} />
    <main id="contenido" className={s.page}>
      <header className={app['page-heading']}>
        <div><h1>Conectar</h1><p className={app['heading-subtitle']}>Conecta tus agentes con PMT y pon a trabajar a tu equipo.</p></div>
      </header>
      <div className={s.sections}>
        <nav className={s.index} aria-label="Índice de Conectar">
          <p>En esta página</p>
          <ol>{SECTIONS.map(([id, label]) => <li key={id}><a href={`#${id}`}>{label}</a></li>)}</ol>
        </nav>

        <section id="tokens" className={s.section} aria-labelledby="connect-tokens-title">
          <h2 id="connect-tokens-title">Tokens de API</h2>
          <p>Para tus agentes: envíalos como <code>Authorization: Bearer</code> al MCP. Actúan con tu username.</p>
          <form className={s.inlineForm} onSubmit={submit} noValidate>
            <label>Nombre<input className={s.input} name="tokenName" value={name} onChange={event => setName(event.target.value)} maxLength={100} placeholder="laptop" /></label>
            <button type="submit" className={s.primary} disabled={busy}>{busy ? 'Creando…' : 'Crear token'}</button>
          </form>
          {created && <div className={s.reveal} role="status">
            <p>Copia el token «{created.name}» ahora: no se vuelve a mostrar.</p>
            <Secret value={created.token} label="Token nuevo" />
            <button type="button" className={s.secondary} onClick={() => setCreated(null)}>Ya lo copié</button>
          </div>}
          {error && <p className={s.error} role="alert">{error}</p>}
          <QueryState result={tokens} label="Cargando tokens…" />
          {tokens.data && (tokens.data.length ? <div className={s.tableWrap} tabIndex={0} role="region" aria-label="Tokens de API"><table className={s.table}>
            <thead><tr><th scope="col">Nombre</th><th scope="col">Creado</th><th scope="col">Último uso</th><th scope="col">Revocar</th></tr></thead>
            <tbody>{tokens.data.map(item => <tr key={item.id}><td>{item.name}</td><td>{formatDate(item.created_at)}</td><td>{item.last_used_at ? formatDate(item.last_used_at) : 'Nunca'}</td><td><button type="button" className={s.danger} aria-label={`Revocar ${item.name}`} onClick={() => revoke(item.id, item.name)}>Revocar</button></td></tr>)}</tbody>
          </table></div> : <p>No tienes tokens.</p>)}
        </section>

        <section id="cli" className={s.section} aria-labelledby="connect-cli-title">
          <h2 id="connect-cli-title">Instalar el CLI <code>pmt</code></h2>
          <p>En Linux o macOS, desde tu terminal.</p>
          <ol className={s.steps}>
            <li><h3>Instala pmt</h3>
              <Command value={commands.install} label="instalar pmt" />
              <p>Lo deja en <code>~/.local/bin</code>, que debe estar en tu <code>PATH</code>; el instalador avisa si falta. Para actualizar, corre el mismo comando otra vez.</p>
            </li>
            <li><h3>Inicia sesión</h3>
              <Command value="pmt login" label="iniciar sesión" />
              <p>Abre el navegador con un código: entra a PMT y pulsa «Autorizar». El código vence en 10 minutos.</p>
              <p>Por SSH o sin navegador, usa este comando y abre la URL que muestra en cualquier equipo con sesión en PMT:</p>
              <Command value="pmt login --sin-navegador" label="iniciar sesión sin navegador" />
              <p>Crea un token llamado <code>{'pmt <máquina>'}</code>, que aparece en la tabla de arriba y puedes revocar. <code>pmt logout</code> también lo revoca.</p>
            </li>
            <li><h3>Comprueba tu sesión</h3><Command value="pmt whoami" label="comprobar sesión" /><p>Confirma con qué usuario y servidor quedó configurado.</p></li>
          </ol>
        </section>

        <section id="mcp" className={s.section} aria-labelledby="connect-mcp-title">
          <h2 id="connect-mcp-title">Conectar el MCP</h2>
          <p>Si acabas de crear un token, los comandos ya lo incluyen. Si muestran <code>{'<tu-token>'}</code>, crea uno: el token solo se ve al crearlo.</p>
          <div className={s.cards}>
            <article className={s.card} aria-labelledby="connect-claude-title"><h3 id="connect-claude-title">Claude Code</h3>
              <Command value={commands.claude} label="conectar Claude Code" />
            </article>
            <article className={s.card} aria-labelledby="connect-codex-title"><h3 id="connect-codex-title">Codex</h3>
              <p>Codex lee el token de una variable de entorno.</p>
              <Command value={commands.codex} label="conectar Codex" />
              <Command value={commands.exportToken} label="guardar PMT_TOKEN en bashrc" />
              <p>Abre una terminal nueva o ejecuta:</p><Command value="source ~/.bashrc" label="recargar bashrc" />
            </article>
          </div>
          <p className={s.note}>Esta conexión es para tu propia sesión. <code>pmt pp</code> y <code>pmt equipo</code> conectan automáticamente a los agentes del equipo.</p>
        </section>

        <section id="pp" className={s.section} aria-labelledby="connect-pp-title">
          <h2 id="connect-pp-title">Levantar a PP con <code>pmt pp</code></h2>
          <p>Necesitas herdr, <code>claude</code> o <code>codex</code> instalados y haber hecho <code>pmt login</code>.</p>
          <ol className={s.steps}>
            <li><h3>Abre el proyecto en herdr</h3><p>Dentro de herdr, en la carpeta del proyecto, ejecuta:</p><Command value="pmt pp" label="levantar a PP" /></li>
            <li><h3>Elige cómo trabajar</h3><p>La primera vez muestra tus proyectos de PMT donde eres owner o editor y la opción «Nuevo: lo crea PP». Elige la herramienta (Claude con <code>opus</code> o Codex con <code>gpt-6.1-terra</code>) y el effort (<code>medium</code> o <code>high</code>).</p><p>Guarda la configuración en <code>.pmt/config.yaml</code> y agrega <code>.pmt/</code> al <code>.gitignore</code> si es un repositorio git.</p></li>
            <li><h3>Platica con PP</h3><p>Lo lanza en la pestaña <code>pp</code> del workspace de herdr que se llama como la carpeta. La primera vez Claude pregunta si confías en esa carpeta.</p><p>Con «Nuevo», PP crea el proyecto platicando contigo. El siguiente <code>pmt pp</code> lo detecta, te pregunta cuál es y lo guarda.</p></li>
          </ol>
          <p>Para volver a elegir la configuración:</p><Command value="pmt pp --configurar" label="configurar PP de nuevo" />
          <h3 className={s.subheading}>Después, levanta al equipo</h3>
          <Command value="pmt equipo" label="levantar al equipo" />
          <p>Levanta a los otros ocho agentes. La primera vez pregunta descripción, base de datos, comando para consultarla, puertos y herramientas, modelos y effort. Para el equipo completo necesitas tanto <code>claude</code> como <code>codex</code>, además de herdr y <code>pmt</code> en el <code>PATH</code>.</p>
          <div className={s.tableWrap} tabIndex={0} role="region" aria-label="Comandos del equipo"><table className={s.table}>
            <thead><tr><th scope="col">Comando</th><th scope="col">Qué hace</th></tr></thead>
            <tbody>{TEAM_COMMANDS.map(([command, description]) => <tr key={command}><td><Command value={command} label={command} /></td><td>{description}</td></tr>)}</tbody>
          </table></div>
          <p className={s.note}>Lo que los agentes necesitan de ti queda en <code>.pmt/para-usuario.md</code>.</p>
        </section>

        <section id="perfiles" className={s.section} aria-labelledby="connect-profiles-title">
          <h2 id="connect-profiles-title">Perfiles</h2>
          <p>Así avanza una etapa. Dev puede ser DevBackend, DevFrontend o ambos; QAP entra si la funcionalidad toca frontend y backend.</p>
          <ol className={s.flow} aria-label="Flujo de una etapa">{FLOW.map(([id, action], index) => <li key={`${id}-${index}`}><div><strong>{id}</strong><span>{action}</span></div>{index < FLOW.length - 1 && <Icon name="arrow" />}</li>)}</ol>
          <div className={s.cards}>{PROFILES.map(profile => <article key={profile.id} className={s.card} aria-labelledby={`profile-${profile.id}`}>
            <span className={s.profileId}>{profile.id}</span><h3 id={`profile-${profile.id}`}>{profile.name}</h3>
            <p>{profile.description}</p><p><strong>Cuándo entra:</strong> {profile.when}</p>
            <p className={s.model}><strong>Por omisión con pmt:</strong> {profile.model}</p>
          </article>)}</div>
        </section>
      </div>
    </main>
  </>
}
