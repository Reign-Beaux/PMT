import { describe, expect, it } from 'vitest'
import { connectCommands } from './connectCommands'

describe('connectCommands', () => {
  it('usa el origen y el marcador por omisión', () => {
    expect(connectCommands('https://otro.test')).toEqual({
      claude: 'claude mcp add --transport http pmt https://otro.test/mcp --header "Authorization: Bearer <tu-token>" -s user',
      codex: 'codex mcp add pmt --url https://otro.test/mcp --bearer-token-env-var PMT_TOKEN',
      exportToken: "echo 'export PMT_TOKEN=<tu-token>' >> ~/.bashrc",
      install: 'curl -fsSL https://otro.test/install.sh | sh',
    })
  })
  it('incorpora el token y otro origen', () => {
    const commands = connectCommands('http://localhost:8090', 'pmt_abc')
    expect(commands.claude).toContain('http://localhost:8090/mcp --header "Authorization: Bearer pmt_abc"')
    expect(commands.exportToken).toBe("echo 'export PMT_TOKEN=pmt_abc' >> ~/.bashrc")
    expect(commands.codex).toContain('http://localhost:8090/mcp')
    expect(commands.install).toBe('curl -fsSL http://localhost:8090/install.sh | sh')
  })
})
