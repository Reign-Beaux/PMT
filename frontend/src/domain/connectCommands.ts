export function connectCommands(origin: string, token = '<tu-token>') {
  return {
    claude: `claude mcp add --transport http pmt ${origin}/mcp --header "Authorization: Bearer ${token}" -s user`,
    codex: `codex mcp add pmt --url ${origin}/mcp --bearer-token-env-var PMT_TOKEN`,
    exportToken: `echo 'export PMT_TOKEN=${token}' >> ~/.bashrc`,
    install: `curl -fsSL ${origin}/install.sh | sh`,
  }
}
