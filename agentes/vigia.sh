#!/bin/bash
# Vigía: observa a los agentes del proyecto y da aviso a pc si todos llevan
# QUIETO segundos sin trabajar. Tras un aviso no repite hasta que otro agente trabaje.
# Termina si alguien queda blocked o si hay aviso nuevo para el usuario.
K="$(dirname "$(realpath "$0")")"
. "$K/comun.sh"
PARA_USUARIO="${PARA_USUARIO:-$AGENTES_DIR/para-usuario.md}"
PULSO="${PULSO:-15}"
QUIETO="${QUIETO:-150}"
W=$(pane _workspace)
[ -n "$W" ] || { echo "sin workspace: corre iniciar.sh" >&2; exit 1; }

N=$(wc -l < "$PARA_USUARIO"); q=0; armado=1
while :; do
  espera "$PULSO"
  S=$("$HERDR" pane list --workspace "$W" | python3 -c "
import json,sys
n={v:k for k,v in (l.strip().split('=') for l in open(sys.argv[1])) if not k.startswith('_')}
print(' '.join(n[p['pane_id']]+':'+p['agent_status'] for p in json.load(sys.stdin)['result']['panes'] if p['pane_id'] in n and p['agent_status'] not in('idle','done','unknown')))" "$PANES")
  if grep -q blocked <<<"$S"; then echo "BLOQUEADO $S"; break; fi
  [ "$(wc -l < "$PARA_USUARIO")" != "$N" ] && { echo AVISO; tail -3 "$PARA_USUARIO"; break; }
  if grep -q working <<<"$S"; then q=0; else q=$((q+PULSO)); fi
  # pc no rearma: si solo trabajó pc por el aviso, nada cambió en el flujo.
  grep -qE '(^| )[^ ]+:working' <<<"${S//pc:working/}" && armado=1
  if [ "$q" -ge "$QUIETO" ] && [ "$armado" = 1 ]; then
    echo "TODOS QUIETOS: aviso a pc"
    "$K/avisar.sh" pc "Vigía: todos llevan ${QUIETO} s quietos. Revisa el avance en PMT y avisa a quien le toque seguir; si nadie puede seguir, informa al usuario."
    armado=0; q=0
  fi
done
