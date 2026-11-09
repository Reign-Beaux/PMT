import s from '../styles/App.module.scss'

export function Filters<T extends string>({ label, value, options, onChange }: {
  label: string
  value: T
  options: readonly { value: T; label: string }[]
  onChange: (value: T) => void
}) {
  return <div className={s['filter-set']} role="group" aria-label={label}>
    {options.map(option => <button key={option.value} type="button" className={s.filterButton} aria-pressed={value === option.value} onClick={() => onChange(option.value)}>{option.label}</button>)}
  </div>
}
