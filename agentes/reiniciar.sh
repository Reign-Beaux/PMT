#!/bin/bash
# uso: reiniciar.sh <agente>
# Deja al agente con una sesión vacía: /new en Claude y mcode; en Codex y Qwen relanza el proceso.
. "$(dirname "$(realpath "$0")")/comun.sh"
A="$1"; P=$(pane "$A")
[ -n "$P" ] || { echo "agente no encontrado: $A" >&2; exit 1; }
# Con un error de mcode, herdr sigue mostrando working: el título dice Failed.
S=$("$HERDR" pane get "$P" | python3 -c "import json,sys;p=json.load(sys.stdin)['result']['pane'];print(p.get('agent_status'),p.get('terminal_title_stripped',''))")
[ "${S%% *}" = working ] && [[ "$S" != *Failed* ]] && { echo "$A sigue trabajando: espera su aviso" >&2; exit 1; }
if [ "$(herramienta "$A")" = codex ] || [ "$(herramienta "$A")" = qwen ]; then
  # Tras /new, Codex descarta el primer mensaje.
  relanzar "$A" || exit 1
else
  "$HERDR" pane send-keys "$P" ctrl+u >/dev/null
  "$HERDR" pane run "$P" /new >/dev/null
  espera 4
  nombrar "$A"
fi
echo "$A reiniciado"
