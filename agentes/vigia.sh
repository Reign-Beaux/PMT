#!/bin/bash
# Vigía: observa a los agentes del proyecto y da aviso al TL si todos llevan
# QUIETO segundos sin trabajar. Tras un aviso no repite hasta que otro agente trabaje.
# Termina si alguien queda blocked o si hay aviso nuevo para el usuario.
# Si mcode falla (título Failed; herdr sigue en working), le envía /retry hasta
# REINTENTOS veces por episodio; si sigue fallando, lo anota para el usuario y termina.
K="$(dirname "$(realpath "$0")")"
. "$K/comun.sh"
PARA_USUARIO="${PARA_USUARIO:-$AGENTES_DIR/para-usuario.md}"
PULSO="${PULSO:-15}"
QUIETO="${QUIETO:-150}"
REINTENTOS="${REINTENTOS:-3}"
W=$(pane _workspace)
[ -n "$W" ] || { echo "sin workspace: corre iniciar.sh" >&2; exit 1; }
# levantar.sh lo usa para no arrancar un segundo vigía.
echo $$ > "$AGENTES_DIR/vigia.pid"; trap 'rm -f "$AGENTES_DIR/vigia.pid"' EXIT

N=$(wc -l < "$PARA_USUARIO"); q=0; armado=1
declare -A fallos
while :; do
  espera "$PULSO"
  S=$("$HERDR" pane list --workspace "$W" | python3 -c "
import json,sys
n={v:k for k,v in (l.strip().split('=') for l in open(sys.argv[1])) if not k.startswith('_')}
# Failed manda sobre el estado: no cuenta como trabajando.
e=lambda p:'failed' if p.get('terminal_title_stripped','').startswith('Failed') else p['agent_status']
print(' '.join(n[p['pane_id']]+':'+e(p) for p in json.load(sys.stdin)['result']['panes'] if p['pane_id'] in n and e(p) not in('idle','done','unknown')))" "$PANES")
  if grep -q blocked <<<"$S"; then echo "BLOQUEADO $S"; break; fi
  [ "$(wc -l < "$PARA_USUARIO")" != "$N" ] && { echo AVISO; tail -3 "$PARA_USUARIO"; break; }
  # El contador de un agente se reinicia cuando deja de estar en Failed.
  for A in "${!fallos[@]}"; do [[ " $S " == *" $A:failed "* ]] || unset "fallos[$A]"; done
  for A in $(grep -oE '[^ ]+:failed' <<<"$S"); do
    A=${A%:failed}; f=${fallos[$A]:-0}
    if [ "$f" -ge "$REINTENTOS" ]; then
      echo "- $(date '+%F %H:%M') vigía: $A sigue en Failed tras $f reintentos con /retry" >> "$PARA_USUARIO"
      echo "FALLA $A"; exit 1
    fi
    fallos[$A]=$((f+1)); echo "$A en Failed: /retry ${fallos[$A]}/$REINTENTOS"
    "$HERDR" pane send-keys "$(pane "$A")" ctrl+u >/dev/null
    "$HERDR" pane run "$(pane "$A")" /retry >/dev/null
  done
  if grep -q working <<<"$S"; then q=0; else q=$((q+PULSO)); fi
  # pp, pc y tl no rearman: hablar con el usuario o atender el aviso no cambia el flujo.
  R=" $S "; for X in pp pc tl; do R=${R// $X:working / }; done
  grep -qE '(^| )[^ ]+:working' <<<"$R" && armado=1
  if [ "$q" -ge "$QUIETO" ] && [ "$armado" = 1 ]; then
    echo "TODOS QUIETOS: aviso al TL"
    "$K/avisar.sh" tl "Vigía: todos llevan ${QUIETO} s quietos. Revisa el avance en PMT y levanta a quien le toque seguir; si nadie puede seguir, anótalo en para-usuario.md."
    armado=0; q=0
  fi
done
