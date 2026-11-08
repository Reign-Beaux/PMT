import { getComments } from '../../api/comments'
import { formatDate } from '../../domain/formatDate'
import { useAsync } from '../../hooks/useAsync'
import { COMMENT_RESOURCES } from '../../realtime/refreshResources'
import { Empty, ErrorState, Loading } from '../feedback'
import styles from './Detail.module.scss'

export interface CommentListProps {
  itemKind: 'task' | 'bug'
  itemId: string
}

export function CommentList({ itemKind, itemId }: CommentListProps) {
  const { status, data, error, retry } = useAsync(
    (signal) => getComments(itemKind, itemId, { signal }),
    [itemKind, itemId],
    { refreshOn: COMMENT_RESOURCES },
  )

  const comments =
    data !== undefined
      ? [...data].sort(
          (a, b) => Date.parse(a.created_at) - Date.parse(b.created_at),
        )
      : []

  return (
    <section aria-label="Comentarios" className={styles.section}>
      <h3 className={styles.sectionTitle}>Comentarios</h3>
      {status === 'loading' && <Loading label="Cargando comentarios…" />}
      {status === 'error' && error !== undefined && (
        <ErrorState message={error.message} onRetry={retry} />
      )}
      {comments.length === 0 && data !== undefined && (
        <Empty message="Sin comentarios" />
      )}
      {comments.length > 0 && (
        <ol className={styles.list}>
          {comments.map((comment) => (
            <li key={comment.id} className={styles.item}>
              <p className={styles.meta}>
                <span>
                  {comment.actor_profile !== ''
                    ? `${comment.actor_name} (${comment.actor_profile})`
                    : comment.actor_name}
                </span>
                <time dateTime={comment.created_at}>
                  {formatDate(comment.created_at)}
                </time>
              </p>
              <p className={styles.preformatted}>{comment.body}</p>
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
