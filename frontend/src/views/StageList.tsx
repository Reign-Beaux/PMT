import { useState } from 'react'
import { Link } from 'react-router'
import { useProjectContext } from '../shell/useProjectContext'
import { sortStagesByPosition, stageProgressFor } from './stageListProgress'
import { stageProgressPercent, stageProgressDenominator } from '../components/progress/stageProgress'
import { STAGE_STATUS_LABELS } from '../domain/status'
import { STAGE_STATUS_TONES } from '../domain/statusTone'
import { Badge } from '../components/card/Badge'
import { Avatar } from '../components/avatar/Avatar'
import { Filters } from '../components/Filters'
import { Empty } from '../components/feedback'
import { Icon } from '../components/Icon'
import s from '../styles/App.module.scss'

export function StageList() {
  const { project, stages } = useProjectContext()
  const [filter, setFilter] = useState<'all' | 'active'>('all')
  const data = stages.data
  const items = sortStagesByPosition(data?.stages ?? []).filter(stage => filter === 'all' || stage.status === 'in_dev')
  return <section aria-label="Etapas del proyecto">
    <div className={s['section-toolbar']}><h2>Etapas del proyecto {data && <span className={s.count}>{data.stages.length}</span>}</h2><Filters label="Filtrar etapas" value={filter} onChange={setFilter} options={[{ value: 'all', label: 'Todas' }, { value: 'active', label: 'En curso' }]} /></div>
    {data && (items.length === 0 ? <Empty message={data.stages.length === 0 ? 'Este proyecto no tiene etapas' : 'No hay etapas en curso'} /> : <div className={s['stage-list']}>
      {items.map(stage => {
        const numbers = stageProgressFor(stage.id, data.progress)
        const percent = stageProgressPercent(numbers)
        return <Link key={stage.id} className={`${s['stage-row']} ${stage.status === 'in_dev' ? s.current : ''}`} to={`/projects/${project.id}/stages/${stage.id}`}>
          <span className={s['stage-number']}>{String(stage.position).padStart(2, '0')}</span>
          <span className={s['stage-name']}><strong>{stage.name}</strong></span>
          <Badge label={STAGE_STATUS_LABELS[stage.status]} tone={STAGE_STATUS_TONES[stage.status]} />
          <span className={s.stageLead}><Avatar name={stage.lead} /></span>
          <span className={s['stage-progress']}><span className={s.numbers}><span>{numbers.done}/{stageProgressDenominator(numbers)}</span><span>{percent}%</span></span><span className={`${s.track} ${stage.status === 'in_dev' ? s.purple : ''}`} role="progressbar" aria-label={`Avance de ${stage.name}`} aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent}><span style={{ width: `${percent}%` }} /></span></span>
          <Icon name="chevron" />
        </Link>
      })}
    </div>)}
  </section>
}
