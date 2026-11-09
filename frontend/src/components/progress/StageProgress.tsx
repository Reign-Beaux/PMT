import styles from './StageProgress.module.scss'
import {
  progressSegments,
  stageProgressDenominator,
  stageProgressPercent,
} from './stageProgress'

export interface StageProgressProps {
  done: number
  total: number
  cancelled: number
  /** Con counts, la barra se segmenta por estado en vez del fill único. */
  counts?: Record<string, number>
}

export function StageProgress({
  done,
  total,
  cancelled,
  counts,
}: StageProgressProps) {
  const pct = stageProgressPercent({ done, total, cancelled })
  const denom = stageProgressDenominator({ done, total, cancelled })
  return (
    <div className={styles.progress}>
      <div
        role="progressbar"
        aria-label="Avance de la etapa"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        aria-valuetext={`${pct} % (${done} de ${denom} tareas)`}
      >
        <div className={styles.track}>
          {counts === undefined ? (
            <div className={styles.fill} style={{ width: `${pct}%` }} />
          ) : (
            progressSegments(counts).map((segment) => (
              <span
                key={segment.status}
                aria-hidden="true"
                data-status={segment.status}
                className={`${styles.segment} ${styles[segment.tone]}`}
                style={{ width: `${segment.percent}%` }}
              />
            ))
          )}
        </div>
      </div>
      <span className={styles.label}>
        {pct} % · {done}/{denom}
      </span>
    </div>
  )
}
