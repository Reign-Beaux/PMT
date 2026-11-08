import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'

function Greeter() {
  const [clicked, setClicked] = useState(false)
  return (
    <div>
      <p>hola vitest</p>
      <button type="button" onClick={() => setClicked(true)}>
        saludar
      </button>
      {clicked && <p>botón pulsado</p>}
    </div>
  )
}

describe('smoke', () => {
  it('renderiza el texto de la prueba', () => {
    render(<Greeter />)
    expect(screen.getByText('hola vitest')).toBeTruthy()
  })

  it('cambia el texto al pulsar y no arrastra el DOM de la prueba anterior', async () => {
    const user = userEvent.setup()
    render(<Greeter />)
    expect(screen.getByText('hola vitest')).toBeTruthy()
    await user.click(screen.getByText('saludar'))
    expect(screen.getByText('botón pulsado')).toBeTruthy()
  })
})
