import type { SVGProps } from 'react'

const paths = {
  grid: <><rect x="3" y="3" width="7" height="7" rx="1.5" /><rect x="14" y="3" width="7" height="7" rx="1.5" /><rect x="3" y="14" width="7" height="7" rx="1.5" /><rect x="14" y="14" width="7" height="7" rx="1.5" /></>,
  layers: <><path d="m12 3 10 5-10 5L2 8l10-5Z" /><path d="m2 12 10 5 10-5M2 16l10 5 10-5" /></>,
  bug: <><path d="M8 8h8v8a4 4 0 0 1-8 0V8ZM9 8V6a3 3 0 0 1 6 0v2M12 9v11M4 9l4 2M20 9l-4 2M3 15h5M21 15h-5M5 21l4-3M19 21l-4-3" /></>,
  chevron: <path d="m9 5 7 7-7 7" />,
  arrow: <path d="M4 12h15m-5-5 5 5-5 5" />,
  check: <><circle cx="12" cy="12" r="9" /><path d="m8 12 3 3 5-6" /></>,
  circle: <><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></>,
  activity: <path d="M3 12h4l3-7 4 14 3-7h4" />,
  file: <><path d="M14 3H5v18h14V8l-5-5Z" /><path d="M14 3v5h5M8 12h8M8 16h6" /></>,
  close: <path d="m6 6 12 12M6 18 18 6" />,
  home: <><path d="m3 10 9-7 9 7v10H3V10Z" /><path d="M9 20v-7h6v7" /></>,
} as const

export function Icon({ name, ...props }: SVGProps<SVGSVGElement> & { name: keyof typeof paths }) {
  return <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" {...props}>{paths[name]}</svg>
}
