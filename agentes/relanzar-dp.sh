#!/bin/bash
# uso: relanzar-dp.sh <medium|high>
# Relanza a DP con sesión vacía y el effort indicado; se conserva al reiniciarlo.
. "$(dirname "$(realpath "$0")")/comun.sh"
case "${1:-}" in medium|high) ;; *) echo "uso: relanzar-dp.sh <medium|high>" >&2; exit 1;; esac
[ -n "$(pane dp)" ] || { echo "agente no encontrado: dp" >&2; exit 1; }
[ "$(herramienta dp)" = mcode ] && echo "aviso: el effort de mcode es global, no cambia con esto" >&2
echo "$1" > "$AGENTES_DIR/dp.effort"
relanzar dp || exit 1
echo "dp relanzado en $1"
