#!/bin/bash
# uso: avisar.sh <agente> "mensaje"
# Envía el aviso y comprueba que llegó. Si no se confirma, reintenta una vez sin duplicarlo;
# si sigue sin confirmarse, deja una línea en para-usuario.md y sale con error.
. "$(dirname "$(realpath "$0")")/comun.sh"
A="$1"; shift; M="${*//$'\n'/ }"; P=$(pane "$A")
[ -n "$P" ] || { echo "agente no encontrado: $A" >&2; exit 1; }

fallo() {
  echo "$(date '+%F %T') aviso sin confirmar a $A: $M" >> "$AGENTES_DIR/para-usuario.md"
  echo "aviso sin confirmar a $A ($1)" >&2
  exit 1
}

# En Codex, un $ puede abrir el selector de menciones, donde Enter elige una y cambia el mensaje.
# Esc solo si está abierto: si no, interrumpiría el turno.
cerrar_selector() {
  "$HERDR" pane read "$P" --source visible --lines 12 | grep -q 'esc close' || return 0
  "$HERDR" pane send-keys "$P" esc >/dev/null
  espera 1
}

enviar() {
  "$HERDR" pane get "$P" | grep -q '"agent_status":"blocked"' && return 1
  case "$(herramienta "$A")" in
    # herdr agent prompt no admite mcode.
    mcode) "$HERDR" pane run "$P" "$M" >/dev/null ;;
    codex)
      case "$M" in
        *'$'*)
          # agent prompt manda el Enter junto con el texto: aquí van por separado.
          "$HERDR" pane send-text "$P" "$M" >/dev/null || return 1
          espera 1
          cerrar_selector
          "$HERDR" pane send-keys "$P" enter >/dev/null ;;
        *) "$HERDR" agent prompt "$P" "$M" >/dev/null 2>&1 ;;
      esac ;;
    *) "$HERDR" agent prompt "$P" "$M" >/dev/null 2>&1 ;;
  esac
}

# Imprime: <estado> <ocupado> <veces>. El compositor es la última línea con el prompt del agente:
# lo que hay de ahí para abajo no se ha enviado; lo de arriba, enviado o en cola, ya lo tiene.
# Estados: pendiente (sigue en el compositor) o nada.
# herdr muestra a mcode como unknown al estar quieto, y como working tras un error: manda el título.
mirar() {
  python3 -c '
import json,re,sys
p=json.loads(sys.argv[1])["result"]["pane"]
st,ti=p.get("agent_status"),p.get("terminal_title_stripped") or ""
ocupado=st in("working","blocked") or ti.startswith("Working")
if "Failed" in ti: ocupado=False
norm=lambda s:" ".join(s.split())
m=norm(sys.argv[2]); frag,cola=m[:24],m[-24:]
L=[norm(l) for l in sys.stdin.read().splitlines()]
# Qwen muestra * o > como prompt del compositor.
c=[i for i,l in enumerate(L) if re.match(r"[›❯*>]( |$)",l)]
c=c[-1] if c else None
if c is None: print("nada",int(ocupado),-1); sys.exit()
abajo=" ".join(L[c:])
e="nada"
if frag in abajo or "[Pasted" in abajo:
    e="pendiente"
# Un mensaje largo en cola puede ocupar toda la pantalla: de él puede verse solo el final.
arriba=" ".join(L[:c])
print(e,int(ocupado),arriba.count(frag)+arriba.count(cola))
' "$("$HERDR" pane get "$P")" "$M" < <("$HERDR" pane read "$P" --source visible --lines 80)
}

# Espera la confirmación, unos 4 s. Deja en E y O lo último visto.
confirmar() {
  local N FIN=$((SECONDS+4))
  while [ "$SECONDS" -lt "$FIN" ]; do
    read -r E O N <<<"$(mirar)"
    [ "$E" = nada ] && [ "${N:--1}" -gt "$N0" ] && return 0
    [ "$E" = nada ] && [ "$O" = 1 ] && [ "$O0" = 0 ] && return 0
    espera 0.5
  done
  return 1
}

# Sin programa en primer plano, el mensaje correría en el shell.
[ -n "$(primer_plano "$P" 2>/dev/null)" ] || fallo "sin agente en $P"
read -r _ O0 N0 <<<"$(mirar)"
[ "${N0:--1}" -ge 0 ] || fallo "sin agente listo en $P"
enviar || fallo "no se pudo enviar"
confirmar && exit 0

case "$E" in
  pendiente) cerrar_selector; "$HERDR" pane send-keys "$P" enter >/dev/null ;;
  # Ocupado y sin rastro: pudo entrar. Reenviar podría duplicarlo.
  *) [ "$O" = 0 ] || fallo "ocupado y sin rastro del mensaje"; enviar || fallo "no se pudo reenviar" ;;
esac
confirmar && exit 0
# Lo que quede en el compositor se pegaría al siguiente aviso.
[ "$E" = nada ] || "$HERDR" pane send-keys "$P" ctrl+u >/dev/null
fallo "no confirmado tras reintentar"
