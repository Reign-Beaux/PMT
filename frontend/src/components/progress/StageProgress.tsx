import styles from './StageProgress.module.scss'
import { stageProgressDenominator, stageProgressPercent } from './stageProgress'

export interface StageProgressProps {
  done: number
  total: number
  cancelled: number
}

export function StageProgress({ done, total, cancelled }: StageProgressProps) {
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
          <div className={styles.fill} style={{ width: `${pct}%` }} />
        </div>
      </div>
      <span className={styles.label}>
        {pct} % · {done}/{denom}
      </span>
    </div>
  )
}
