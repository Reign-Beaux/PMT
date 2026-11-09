import { useId, useRef, useState } from 'react'
import { CommentList } from './CommentList'
import { StatusHistory } from './StatusHistory'
import s from '../../styles/App.module.scss'

/** Paneles persistentes: cambiar de pestaña no descarta los datos ni la conexión. */
export function ActivityTabs({ itemKind, itemId }: { itemKind: 'task' | 'bug'; itemId: string }) {
  const [tab, setTab] = useState(0)
  const id = useId()
  const refs = useRef<(HTMLButtonElement | null)[]>([])
  const labels = ['Comentarios', 'Historial']
  return <div>
    <div className={s.activityTabs} role="tablist" aria-label="Actividad">
      {labels.map((label, index) => <button key={label} ref={element => { refs.current[index] = element }} type="button" className={s.activityTab} role="tab" id={`${id}-tab-${index}`} aria-controls={`${id}-panel-${index}`} aria-selected={tab === index} tabIndex={tab === index ? 0 : -1}
        onClick={() => setTab(index)} onKeyDown={event => {
          let next: number
          if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') next = (tab + 1) % 2
          else if (event.key === 'Home') next = 0
          else if (event.key === 'End') next = 1
          else return
          event.preventDefault()
          setTab(next)
          refs.current[next]?.focus({ preventScroll: true })
        }}>{label}</button>)}
    </div>
    <div role="tabpanel" id={`${id}-panel-0`} aria-labelledby={`${id}-tab-0`} hidden={tab !== 0} tabIndex={0}><CommentList itemKind={itemKind} itemId={itemId} /></div>
    <div role="tabpanel" id={`${id}-panel-1`} aria-labelledby={`${id}-tab-1`} hidden={tab !== 1} tabIndex={0}><StatusHistory itemKind={itemKind} itemId={itemId} /></div>
  </div>
}
