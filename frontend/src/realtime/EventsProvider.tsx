import type { ReactNode } from 'react'
import { useEffect, useState } from 'react'
import { createEventsConnection } from './eventsConnection'
import type { EventsConnection } from './eventsConnection'
import { EventsContext } from './eventsContext'

interface EventsProviderProps {
  /** Conexión inyectada (pruebas); si falta, se crea una real. */
  connection?: EventsConnection
  children: ReactNode
}

export function EventsProvider({ connection, children }: EventsProviderProps) {
  const [created] = useState(() => connection ?? createEventsConnection())
  const conn = connection ?? created
  useEffect(() => {
    conn.start()
    return () => {
      conn.close()
    }
  }, [conn])
  return (
    <EventsContext.Provider value={conn}>{children}</EventsContext.Provider>
  )
}
