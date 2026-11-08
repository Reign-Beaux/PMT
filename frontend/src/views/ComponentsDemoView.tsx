import type { ReactNode } from 'react'
import { useState } from 'react'
import { Badge } from '../components/card/Badge'
import { Card } from '../components/card/Card'
import { DetailDialog } from '../components/dialog/DetailDialog'
import {
  ConnectionStatus,
  Empty,
  ErrorState,
  Loading,
} from '../components/feedback'
import { KanbanBoard, KanbanColumn } from '../components/kanban'
import { StageProgress } from '../components/progress/StageProgress'
import styles from './ComponentsDemoView.module.scss'

const KANBAN_COLUMNS: Array<{ id: string; label: string; cards: number }> = [
  { id: 'backlog', label: 'Backlog', cards: 2 },
  { id: 'ready', label: 'Ready', cards: 1 },
  { id: 'in_dev', label: 'En desarrollo', cards: 2 },
  { id: 'in_review', label: 'En revisión', cards: 0 },
  { id: 'in_qa', label: 'En QA', cards: 1 },
  { id: 'changes_requested', label: 'Cambios pedidos', cards: 0 },
  { id: 'pending_close', label: 'Pending close', cards: 1 },
  { id: 'done', label: 'Done', cards: 2 },
  { id: 'cancelled', label: 'Cancelled', cards: 0 },
]

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>{title}</h2>
      {children}
    </section>
  )
}

export function ComponentsDemoView() {
  const [openCount, setOpenCount] = useState(0)
  const [detailOpen, setDetailOpen] = useState(false)
  return (
    <main className={styles.page}>
      <h1>Componentes</h1>
      <p>Vitrina de desarrollo para verificación en pantalla.</p>

      <Section title="Carga, vacío y error">
        <Loading />
        <Loading label="Cargando etapas…" />
        <Empty message="No hay tareas en esta columna" />
        <ErrorState message="Falló la conexión con el servidor" onRetry={() => {}} />
        <ConnectionStatus status="connected" lastSyncedAt="2026-11-07T10:30:00Z" />
        <ConnectionStatus status="reconnecting" />
        <ConnectionStatus status="stale" lastSyncedAt="2026-11-07T10:30:00Z" />
      </Section>

      <Section title="Tarjetas">
        <Card
          title="Tarea interactiva"
          id="t-0001"
          subtitle="Etapa 1 · En desarrollo"
          onOpen={() => setOpenCount((n) => n + 1)}
        />
        <p>Aperturas de la tarjeta interactiva: {openCount}</p>
        <Card
          title="Título larguísimo sin espacios para comprobar el ajuste en pantallas estrechas: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
          id="t-0002"
        />
        <Card title="Tarjeta no interactiva" id="t-0003" subtitle="Sin onOpen" />
      </Section>

      <Section title="Badges">
        <div className={styles.badges}>
          <Badge label="Neutral" />
          <Badge label="Info" tone="info" />
          <Badge label="Éxito" tone="success" />
          <Badge label="Advertencia" tone="warning" />
          <Badge label="Peligro" tone="danger" />
        </div>
      </Section>

      <Section title="Barra de avance">
        <StageProgress done={0} total={0} cancelled={0} />
        <StageProgress done={1} total={2} cancelled={0} />
        <StageProgress done={2} total={3} cancelled={0} />
        <StageProgress done={3} total={5} cancelled={2} />
      </Section>

      <Section title="Kanban">
        <KanbanBoard label="Tablero de ejemplo">
          {KANBAN_COLUMNS.map((column) => (
            <KanbanColumn
              key={column.id}
              id={column.id}
              label={column.label}
              count={column.cards}
            >
              {Array.from({ length: column.cards }, (_, i) => (
                <Card
                  key={i}
                  title={`Tarea de ejemplo ${i + 1}`}
                  id={`${column.id}-000${i + 1}`}
                  subtitle={`${column.label} · asignada a dev`}
                />
              ))}
            </KanbanColumn>
          ))}
        </KanbanBoard>
      </Section>

      <Section title="Diálogo">
        <button type="button" onClick={() => setDetailOpen(true)}>
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
