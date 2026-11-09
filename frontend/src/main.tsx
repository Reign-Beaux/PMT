import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router'
import './styles/global.scss'
import { routes } from './routes/routes'
import { EventsProvider } from './realtime/EventsProvider'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <EventsProvider><RouterProvider router={createBrowserRouter(routes)} /></EventsProvider>
  </StrictMode>,
)
