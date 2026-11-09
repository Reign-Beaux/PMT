import type { ReactNode } from 'react'
import { useState } from 'react'
import type { BadgeTone } from '../components/card/Badge'
import { Badge } from '../components/card/Badge'
import { DetailDialog } from '../components/dialog/DetailDialog'
import {
  ConnectionStatus,
  Empty,
  ErrorState,
  Loading,
} from '../components/feedback'
import { StageProgress } from '../components/progress/StageProgress'
import type {
  BugStatus,
  ProjectStatus,
  StageStatus,
  TaskStatus,
} from '../domain/status'
import {
  BUG_STATUS_LABELS,
  PROJECT_STATUS_LABELS,
  STAGE_STATUS_LABELS,
  TASK_STATUS_LABELS,
} from '../domain/status'
import {
  BUG_STATUS_TONES,
  PROJECT_STATUS_TONES,
  STAGE_STATUS_TONES,
  TASK_STATUS_TONES,
} from '../domain/statusTone'
import styles from './ComponentsDemoView.module.scss'

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>{title}</h2>
      {children}
    </section>
  )
}

function StatusLozenges<T extends string>({
  labels,
  tones,
}: {
  labels: Record<T, string>
  tones: Record<T, BadgeTone>
}) {
  return (
    <div className={styles.badges}>
      {(Object.keys(labels) as T[]).map((status) => (
        <Badge key={status} label={labels[status]} tone={tones[status]} />
      ))}
    </div>
  )
}

export function ComponentsDemoView() {
  const [detailOpen, setDetailOpen] = useState(false)
  return (
    <main className={styles.page}>
      <h1>Componentes</h1>
      <p>Vitrina de desarrollo para verificación en pantalla.</p>

      <Section title="Carga, vacío y error">
        <Loading />
        <Loading label="Cargando etapas…" />
        <Loading label="Cargando proyectos…" variant="list" />
        <Loading label="Cargando tarea…" variant="detail" />
        <Empty message="No hay tareas en esta columna" />
        <ErrorState message="Falló la conexión con el servidor" onRetry={() => {}} />
        <ConnectionStatus status="connected" lastSyncedAt="2026-11-07T10:30:00Z" />
        <ConnectionStatus status="reconnecting" />
        <ConnectionStatus status="stale" lastSyncedAt="2026-11-07T10:30:00Z" />
      </Section>

      <Section title="Badges">
        <div className={styles.badges}>
          <Badge label="Neutral" />
          <Badge label="Info" tone="info" />
          <Badge label="Éxito" tone="success" />
          <Badge label="Advertencia" tone="warning" />
          <Badge label="Peligro" tone="danger" />
        </div>
        <h3>Proyecto</h3>
        <StatusLozenges<ProjectStatus>
          labels={PROJECT_STATUS_LABELS}
          tones={PROJECT_STATUS_TONES}
        />
        <h3>Etapa</h3>
        <StatusLozenges<StageStatus>
          labels={STAGE_STATUS_LABELS}
          tones={STAGE_STATUS_TONES}
        />
        <h3>Tarea</h3>
        <StatusLozenges<TaskStatus>
          labels={TASK_STATUS_LABELS}
          tones={TASK_STATUS_TONES}
        />
        <h3>Bug</h3>
        <StatusLozenges<BugStatus>
          labels={BUG_STATUS_LABELS}
          tones={BUG_STATUS_TONES}
        />
      </Section>

      <Section title="Barra de avance">
        <StageProgress done={0} total={0} cancelled={0} />
        <StageProgress done={1} total={2} cancelled={0} />
        <StageProgress done={2} total={3} cancelled={0} />
        <StageProgress done={3} total={5} cancelled={2} />
      </Section>

      <Section title="Diálogo">
        <button
          type="button"
          className={styles.demoButton}
          onClick={() => setDetailOpen(true)}
        >
          Abrir detalle
        </button>
        <DetailDialog
          open={detailOpen}
          title="Frontend — 10 — Diálogo de detalle accesible con título larguísimo para comprobar el ajuste: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
          onClose={() => setDetailOpen(false)}
        >
          <p>
            Etapa 1 · in_review · asignada a pmt-frontend-devfront. Texto de
            ejemplo para comprobar el contenido del diálogo y su scroll interno
            con contenido largo.
          </p>
        </DetailDialog>
      </Section>
    </main>
  )
}
