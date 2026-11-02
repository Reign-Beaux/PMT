#!/bin/bash
# uso: levantar.sh <agente> "encargo"
# Reinicia al agente y le da su encargo en una sesión vacía.
K="$(dirname "$(realpath "$0")")"
. "$K/comun.sh"
A="$1"; shift; E="$*"
[ -n "$E" ] || { echo "falta el encargo" >&2; exit 1; }
"$K/reiniciar.sh" "$A" >/dev/null || exit 1
"$K/avisar.sh" "$A" "$E" || exit 1
espera 6
nombrar "$A"
echo "$A levantado"
