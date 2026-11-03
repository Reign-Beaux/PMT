#!/bin/bash
# uso: lanzar.sh <agente>
# Corre dentro del pane: arranca al agente con su herramienta, modelo, perfil y token.
. "$(dirname "$(realpath "$0")")/comun.sh"
A="$1"; H=$(herramienta "$A"); M=$(modelo "$A"); E=$(effort "$A"); T=$(token "$A")
F="$AGENTES_DIR/perfiles/$A.md"
[ -f "$F" ] || { echo "falta el perfil generado: $F" >&2; exit 1; }
[ -r "$HOME/.config/pmt/tokens/$T" ] && [ -n "$T" ] || { echo "falta el token de $A: $T" >&2; exit 1; }
cd "$PROYECTO_RUTA" || exit 1
export PMT_TOKEN="$(cat "$HOME/.config/pmt/tokens/$T")"
case "$H" in
  claude)
    # claude-400 es función de shell: aquí se usa el comando que envuelve.
    exec claude --dangerously-skip-permissions --autocompact 400k --model "$M" --effort "$E" \
      --settings '{"enabledPlugins":{"engram@engram":false}}' \
      --mcp-config "$AGENTES_DIR/mcp.json" --strict-mcp-config \
      --append-system-prompt-file "$F" ;;
  codex)
    # ~/.codex tiene un pmt de la v3: se usa ~/.codex-flow.
    # El shell de login no carga nvm desde .bashrc: Codex no veria node ni pnpm.
    # Cerca del límite, Codex abre un menú de cambio de modelo: un aviso con el menú abierto se pierde.
    export CODEX_HOME="$HOME/.codex-flow"
    I=$(python3 -c "import json,sys;print(json.dumps(open(sys.argv[1]).read()))" "$F")
    exec codex -m "$M" -c "developer_instructions=$I" -c "model_reasoning_effort=\"$E\"" -c allow_login_shell=false \
      -c notice.hide_rate_limit_model_nudge=true \
      -c "mcp_servers.pmt.url=\"$PMT_URL\"" -c "projects.\"$PROYECTO_RUTA\".trust_level=\"trusted\"" ;;
  mcode)
    exec mcode --model "$M" --append-system-prompt-file "$F" ;;
  *) echo "herramienta desconocida para $A: $H" >&2; exit 1 ;;
esac
