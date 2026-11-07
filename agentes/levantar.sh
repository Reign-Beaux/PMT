#!/bin/bash
# uso: levantar.sh <agente> "encargo"
# Reinicia al agente y le da su encargo en una sesión vacía.
K="$(dirname "$(realpath "$0")")"
. "$K/comun.sh"
A="$1"; shift; E="$*"
[ -n "$E" ] || { echo "falta el encargo" >&2; exit 1; }
"$K/reiniciar.sh" "$A" >/dev/null || exit 1
# Con el TL arranca el vigía, si no corre ya: termina solo cuando alguien anota en para-usuario.md.
if [ "$A" = tl ]; then
  if ! kill -0 "$(cat "$AGENTES_DIR/vigia.pid" 2>/dev/null)" 2>/dev/null; then
    touch "$AGENTES_DIR/para-usuario.md"
    setsid nohup "$K/vigia.sh" >> "$AGENTES_DIR/vigia.log" 2>&1 < /dev/null &
  fi
fi
"$K/avisar.sh" "$A" "$E" || exit 1
espera 6
nombrar "$A"
echo "$A levantado"
