# Se incluye con source. Carga la configuración del proyecto y las funciones comunes.
# La configuración sale de AGENTES_ENV o del .agentes.env más cercano subiendo desde el directorio actual.
KIT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HERDR="${HERDR:-$(command -v herdr || echo "$HOME/.local/bin/herdr")}"
TODOS="pp pc tl dp devback devfront cr qap qa"

if [ -z "${AGENTES_ENV:-}" ]; then
  d=$PWD
  while [ "$d" != / ]; do
    [ -f "$d/.agentes.env" ] && { AGENTES_ENV="$d/.agentes.env"; break; }
    d=$(dirname "$d")
  done
fi
[ -f "${AGENTES_ENV:-}" ] || { echo "no encuentro .agentes.env: define AGENTES_ENV" >&2; exit 1; }
AGENTES_ENV=$(realpath "$AGENTES_ENV")
export AGENTES_ENV
. "$AGENTES_ENV"
PROYECTO_RUTA="${PROYECTO_RUTA%/}"
AGENTES_DIR="${AGENTES_DIR:-$PROYECTO_RUTA/.agentes}"
PANES="$AGENTES_DIR/panes.env"

espera() { timeout "$1" tail -f /dev/null; }
pane() { [ -f "$PANES" ] && grep -E "^$1=" "$PANES" | cut -d= -f2; }
dato() { local v="$(echo "$1" | tr a-z A-Z)_$2"; echo "${!v:-}"; }
herramienta() { dato "$1" HERRAMIENTA; }
token() { dato "$1" TOKEN; }
modelo() { dato "$1" MODELO; }
# El effort guardado por relanzar-dp.sh manda sobre el del .env.
effort() { cat "$AGENTES_DIR/$1.effort" 2>/dev/null || dato "$1" EFFORT; }
nombrar() { "$HERDR" agent rename "$(pane "$1")" "${HERDR_PREFIJO:-}$1" >/dev/null 2>&1; }

# Imprime lo que corre en primer plano en el pane; vacío si está en el shell.
primer_plano() {
  "$HERDR" pane process-info --pane "$1" | python3 -c "
import json,sys
p=json.load(sys.stdin)['result']['process_info']
print(' '.join(x['name'] for x in p['foreground_processes'] if x['pid']!=p.get('shell_pid')))"
}
espera_shell() {
  local i
  for i in 1 2 3 4 5 6 7 8 9 10; do
    [ -z "$(primer_plano "$1")" ] && return 0
    espera 1
  done
  return 1
}

# Comando en el pane, para que el agente herede AGENTES_ENV y vea a sus compañeros.
lanzar() { "$HERDR" pane run "$1" "AGENTES_ENV=$AGENTES_ENV $KIT/lanzar.sh $2" >/dev/null; }

# Cierra el programa del agente y lo vuelve a lanzar, con sesión vacía.
relanzar() {
  local A=$1 P S
  P=$(pane "$A")
  case "$(herramienta "$A")" in codex|qwen) S=/quit;; *) S=/exit;; esac
  "$HERDR" pane send-keys "$P" ctrl+u >/dev/null
  "$HERDR" pane run "$P" "$S" >/dev/null
  espera_shell "$P" || { echo "$A no cerró" >&2; return 1; }
  lanzar "$P" "$A"
  espera 12
  nombrar "$A"
}
