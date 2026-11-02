#!/bin/bash
# uso: relanzar-mcode.sh <agente|todos>
# Cierra mcode y lo vuelve a lanzar, con sesión vacía y la configuración actual.
. "$(dirname "$(realpath "$0")")/comun.sh"
L="${1:-}"
if [ "$L" = todos ]; then L=""; for A in $TODOS; do [ "$(herramienta "$A")" = mcode ] && [ -n "$(pane "$A")" ] && L+=" $A"; done; fi
[ -n "$L" ] || { echo "uso: relanzar-mcode.sh <agente|todos>" >&2; exit 1; }
for A in $L; do
  P=$(pane "$A")
  [ -n "$P" ] && [ "$(herramienta "$A")" = mcode ] || { echo "agente mcode no encontrado: $A" >&2; exit 1; }
  relanzar "$A" || exit 1
  echo "$A: $("$HERDR" pane read "$P" --source visible --lines 40 | grep -o -E 'Full access|Effort [a-z]+|[Ll]ogin' | sort -u | tr '\n' ' ')"
done
