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
  qwen)
    # El MCP sale de .qwen/settings.json del proyecto, que genera iniciar.sh. Qwen pide aprobar
    # el servidor y la aprobación queda ligada a su configuración: se aprueba en cada arranque.
    # El effort no se pasa por línea de comandos: lo fija la configuración del modelo en ~/.qwen.
    # Herdr reconoce a Qwen por el nombre del proceso, y el lanzador de Qwen termina en node:
    # se arranca node desde un enlace llamado qwen. QWEN_RAIZ es la instalación de Qwen Code.
    R="${QWEN_RAIZ:-$HOME/.local/lib/qwen-code}"
    mkdir -p "$AGENTES_DIR/bin" && ln -sf "$R/node/bin/node" "$AGENTES_DIR/bin/qwen"
    "$R/bin/qwen" mcp approve pmt >/dev/null 2>&1
    export QWEN_CODE_LAUNCHER_PATH="$R/bin/qwen"
    exec "$AGENTES_DIR/bin/qwen" "$R/lib/cli-entry.js" --yolo --model "$M" --append-system-prompt "$(cat "$F")" ;;
  *) echo "herramienta desconocida para $A: $H" >&2; exit 1 ;;
esac
