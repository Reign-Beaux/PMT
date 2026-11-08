import { Link } from 'react-router'

export function NotFoundView() {
  return (
    <>
      <h1>Página no encontrada</h1>
      <p>
        <Link to="/">Ir al inicio</Link>
      </p>
    </>
  )
}
