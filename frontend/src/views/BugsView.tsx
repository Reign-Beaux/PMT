import { useState } from 'react'
import { useParams } from 'react-router'
import { getProjectBugs } from '../api/bugs'
import { Card } from '../components/card/Card'
import { DetailDialog } from '../components/dialog/DetailDialog'
import { Empty, ErrorState, Loading } from '../components/feedback'
import { KanbanBoard, KanbanColumn } from '../components/kanban'
import { groupBugsByStatus } from '../domain/groupBugsByStatus'
import { useAsync } from '../hooks/useAsync'
import { BUG_RESOURCES } from '../realtime/refreshResources'
import { useDetailSelection } from '../routes/useDetailSelection'
import { BugDetail } from './BugDetail'
import styles from './BugsView.module.scss'

export function BugsView() {
  const { projectId } = useParams()
  if (projectId === undefined) return null
  return <BugsBoard projectId={projectId} />
}

function BugsBoard({ projectId }: { projectId: string }) {
  const { selection, select, clear } = useDetailSelection()
  const { status, data, error, retry } = useAsync(
    (signal) => getProjectBugs(projectId, { signal }),
    [projectId],
    { refreshOn: BUG_RESOURCES },
  )

  const selectedBug =
    selection?.kind === 'bug' && data !== undefined
      ? data.find((bug) => bug.id === selection.id)
      : undefined

  // Ids del bug seleccionado que ya se encontraron en la lista: si un refresco
  // lo quita, el diálogo muestra "Ya no existe" en vez de "Bug no encontrado".
  // Estado derivado durante el render (como useAsync con sus deps).
  const [seenIds, setSeenIds] = useState<readonly string[]>([])
  if (selectedBug !== undefined && !seenIds.includes(selectedBug.id)) {
    setSeenIds((previous) =>
      previous.includes(selectedBug.id)
        ? previous
        : [...previous, selectedBug.id],
    )
  }

  const dialogOpen = selection?.kind === 'bug' && data !== undefined

  return (
    <section className={styles.view}>
      <h2 className={styles.name}>Bugs</h2>
      {status === 'loading' && <Loading label="Cargando bugs…" />}
      {status === 'error' && error !== undefined && (
        <ErrorState message={error.message} onRetry={retry} />
      )}
      {data !== undefined && (
        <>
          {data.length === 0 && (
            <Empty message="Este proyecto no tiene bugs" />
          )}
          <KanbanBoard label="Tablero de bugs">
            {groupBugsByStatus(data).map((column) => (
              <KanbanColumn
                key={column.status}
                id={column.status}
                label={column.label}
                count={column.bugs.length}
              >
                <ul className={styles.cards}>
                  {column.bugs.map((bug) => (
                    <li key={bug.id}>
                      <Card
                        title={bug.title}
                        id={bug.id}
                        subtitle={
                          bug.product_version !== ''
                            ? `Versión ${bug.product_version}`
                            : 'Versión —'
                        }
                        onOpen={() => select({ kind: 'bug', id: bug.id })}
                      />
                    </li>
                  ))}
                </ul>
              </KanbanColumn>
            ))}
          </KanbanBoard>
        </>
      )}
      {dialogOpen &&
        selection?.kind === 'bug' &&
        (selectedBug !== undefined ? (
          <DetailDialog open title={selectedBug.title} onClose={clear}>
            <BugDetail bug={selectedBug} />
          </DetailDialog>
        ) : seenIds.includes(selection.id) ? (
          <DetailDialog open title="Ya no existe" onClose={clear}>
            <p>Ya no existe</p>
          </DetailDialog>
        ) : (
          <DetailDialog open title="Bug no encontrado" onClose={clear}>
            <p>Bug no encontrado</p>
          </DetailDialog>
        ))}
    </section>
  )
}
