#!/bin/bash
# uso: iniciar.sh <archivo .env> [agente ...]
# Crea las pestañas que falten, guarda los panes, genera los perfiles y lanza a los agentes.
# Sin lista, levanta a los nueve. Un agente que ya corre no se toca.
[ -f "${1:-}" ] || { echo "uso: iniciar.sh <archivo .env> [agente ...]" >&2; exit 1; }
AGENTES_ENV=$1; shift
. "$(dirname "$(realpath "$0")")/comun.sh"
L="${*:-$TODOS}"

for A in $L; do
  case " $TODOS " in *" $A "*) ;; *) echo "agente desconocido: $A" >&2; exit 1;; esac
  case "$(herramienta "$A")" in
    claude|codex|mcode|qwen) ;;
    *) echo "herramienta inválida para $A: $(herramienta "$A")" >&2; exit 1;;
  esac
  [ -n "$(modelo "$A")" ] || { echo "falta el modelo de $A" >&2; exit 1; }
  case "$(dato "$A" EFFORT)" in medium|high) ;; *) echo "effort inválido para $A" >&2; exit 1;; esac
  T=$(token "$A")
  [ -n "$T" ] && [ -r "$HOME/.config/pmt/tokens/$T" ] || { echo "falta el token de $A: ${T:-sin nombre}" >&2; exit 1; }
  for X in $(mcp_extra "$A"); do
    [[ "$X" =~ ^[a-z0-9]+$ ]] && [ "$X" != pmt ] || { echo "MCP extra inválido para $A: $X" >&2; exit 1; }
    [ -n "$(mcp_dato "$X" URL)" ] || { echo "falta MCP_$(echo "$X" | tr a-z A-Z)_URL para $A" >&2; exit 1; }
    T=$(mcp_dato "$X" TOKEN)
    [ -n "$T" ] && [ -r "$HOME/.config/pmt/tokens/$T" ] || { echo "falta el token del MCP $X: ${T:-sin nombre}" >&2; exit 1; }
  done
  case "$(herramienta "$A")" in mcode|qwen) [ -z "$(mcp_extra "$A")" ] ||
    echo "aviso: $A usa $(herramienta "$A"), que no recibe MCP extra por agente" >&2;; esac
done
for V in PROYECTO_RUTA DB_URL DB_CONSULTA PUERTO_BACKEND PUERTO_FRONTEND PMT_URL HERDR_WORKSPACE; do
  [ -n "${!V:-}" ] || { echo "falta $V en $AGENTES_ENV" >&2; exit 1; }
done
[ -d "$PROYECTO_RUTA" ] || { echo "no existe el proyecto: $PROYECTO_RUTA" >&2; exit 1; }

mkdir -p "$AGENTES_DIR/perfiles"
touch "$AGENTES_DIR/para-usuario.md" "$PANES"

# Workspace: por ID o etiqueta; si no existe, se crea.
W=$("$HERDR" workspace list | python3 -c "
import json,sys
for w in json.load(sys.stdin)['result']['workspaces']:
    if sys.argv[1] in (w['workspace_id'],w['label']): print(w['workspace_id']); break
" "$HERDR_WORKSPACE")
if [ -z "$W" ]; then
  J=$("$HERDR" workspace create --cwd "$PROYECTO_RUTA" --label "$HERDR_WORKSPACE" --no-focus)
  read -r W T0 <<<"$(python3 -c "import json,sys;r=json.loads(sys.argv[1])['result'];print(r['workspace']['workspace_id'],r['tab']['tab_id'])" "$J")"
  "$HERDR" tab rename "$T0" "${L%% *}" >/dev/null
fi

# Pane de un agente: el guardado si sigue vivo en el workspace, luego su pestaña, y si no una nueva.
buscar_pane() {
  local P
  P=$(pane "$1")
  if [ -n "$P" ] && "$HERDR" pane get "$P" 2>/dev/null | grep -q "\"workspace_id\":\"$W\""; then echo "$P"; return; fi
  P=$(python3 - "$1" "$("$HERDR" tab list --workspace "$W")" "$("$HERDR" pane list --workspace "$W")" <<'PY'
import json,sys
tabs=[t['tab_id'] for t in json.loads(sys.argv[2])['result']['tabs'] if t['label']==sys.argv[1]]
if tabs:
    print(next(p['pane_id'] for p in json.loads(sys.argv[3])['result']['panes'] if p['tab_id']==tabs[0]))
PY
)
  if [ -n "$P" ]; then echo "$P"; return; fi
  "$HERDR" tab create --workspace "$W" --cwd "$PROYECTO_RUTA" --label "$1" --no-focus |
    python3 -c "import json,sys;print(json.load(sys.stdin)['result']['root_pane']['pane_id'])"
}

N=$(mktemp "$AGENTES_DIR/panes.XXXXXX")
grep -v -E '^_workspace=' "$PANES" > "$N"
for A in $L; do
  P=$(buscar_pane "$A")
  [ -n "$P" ] || { echo "sin pane para $A" >&2; rm -f "$N"; exit 1; }
  sed -i "/^$A=/d" "$N"
  echo "$A=$P" >> "$N"
done
echo "_workspace=$W" >> "$N"
mv "$N" "$PANES"

# mcp.json para Claude: el token sale del entorno, nunca se escribe aquí.
cat > "$AGENTES_DIR/mcp.json" <<JSON
{
  "mcpServers": {
    "pmt": {
      "type": "http",
      "url": "$PMT_URL",
      "headers": {
        "Authorization": "Bearer \${PMT_TOKEN:-}"
      }
    }
  }
}
JSON
# Agente con MCP extra (solo Claude y Codex): mcp-<agente>.json con pmt y sus extra. Cada token sale de MCP_TOKEN_<NOMBRE>.
for A in $TODOS; do
  E=""
  for X in $(mcp_extra "$A"); do E+="$X=$(mcp_dato "$X" URL) "; done
  python3 - "$AGENTES_DIR/mcp.json" "$AGENTES_DIR/mcp-$A.json" $E <<'PY'
import json,os,sys
base,dest,extras=sys.argv[1],sys.argv[2],sys.argv[3:]
if not extras:
    if os.path.exists(dest): os.remove(dest)
    sys.exit()
d=json.load(open(base))
for e in extras:
    n,url=e.split('=',1)
    d['mcpServers'][n]={'type':'http','url':url,'headers':{'Authorization':'Bearer ${MCP_TOKEN_%s:-}'%n.upper()}}
json.dump(d,open(dest,'w'),indent=2)
PY
done
# mcode lo toma de .mcp.json en la raíz del proyecto.
[ -e "$PROYECTO_RUTA/.mcp.json" ] || cp "$AGENTES_DIR/mcp.json" "$PROYECTO_RUTA/.mcp.json"
# Qwen lo toma de .qwen/settings.json del proyecto; se agrega pmt sin tocar lo demás. El token sale del entorno.
python3 - "$PROYECTO_RUTA/.qwen/settings.json" "$PMT_URL" <<'PY'
import json,os,sys
p,url=sys.argv[1],sys.argv[2]
os.makedirs(os.path.dirname(p),exist_ok=True)
d=json.load(open(p)) if os.path.exists(p) else {}
d.setdefault('mcpServers',{})['pmt']={'httpUrl':url,'headers':{'Authorization':'Bearer $PMT_TOKEN'}}
# Herdr sabe si Qwen trabaja por el estado en el título de la terminal.
d.setdefault('ui',{})['showStatusInTitle']=True
json.dump(d,open(p,'w'),indent=2)
PY

# Perfil final: rol + comun.md con los datos del proyecto, para los nueve.
export KIT AGENTES_DIR PROYECTO_RUTA PROYECTO_DESCRIPCION DB_URL DB_CONSULTA PUERTO_BACKEND PUERTO_FRONTEND
TABLA=""
for A in $TODOS; do
  T=$(token "$A"); [ -n "$T" ] || continue
  case $A in
    pp|pc|tl|dp|cr|qap|qa) X=$(echo "$A" | tr a-z A-Z);;
    devback) X=DevBackend;;
    devfront) X=DevFrontend;;
  esac
  TABLA+="| $A | $T | $X |"$'\n'
done
export TABLA="${TABLA%$'\n'}"
# Los textos viven en el backend, que también los sirve: [[local||servido]] marca un tramo con dos versiones
# y aquí se toma la local. PC recibe pc-comun.md en lugar de comun.md: solo lee herdr, sin reglas de PMT ni avisos.
python3 - "$KIT/../backend/internal/features/profiles/catalog" "$AGENTES_DIR/perfiles" $TODOS <<'PY'
import os,re,sys
src,dest,agentes=sys.argv[1],sys.argv[2],sys.argv[3:]
v={k:os.environ.get(k,'') for k in 'KIT AGENTES_DIR PROYECTO_RUTA PROYECTO_DESCRIPCION DB_URL DB_CONSULTA PUERTO_BACKEND PUERTO_FRONTEND'.split()}
v.update(TABLA_AGENTES=os.environ['TABLA'])
for a in agentes:
    t=open(f'{src}/perfiles/{a}.md').read()+open(f'{src}/{"pc-comun" if a=="pc" else "comun"}.md').read()
    t=re.sub(r'\[\[(.*?)\|\|(.*?)\]\]',lambda m:m[1],t,flags=re.S)
    open(f'{dest}/{a}.md','w').write(re.sub(r'\{\{(\w+)\}\}',lambda m:v[m[1]],t))
PY

# El effort de mcode es global: se avisa, no se cambia.
G=$(python3 -c "
import os,re
m=re.search(r'^defaultModelThinking:\n\s+effort: (\w+)',open(os.path.expanduser('~/.minimax/config.yaml')).read(),re.M)
print(m[1] if m else '')" 2>/dev/null)

for A in $L; do
  P=$(pane "$A")
  # Un shell recién abierto tarda en quedar libre; si no se libera, ahí corre un agente.
  espera_shell "$P" || { echo "$A ya corre en $P"; continue; }
  lanzar "$P" "$A"
  [ "$(herramienta "$A")" = mcode ] && [ "$(dato "$A" EFFORT)" != "$G" ] &&
    echo "aviso: $A pide effort $(dato "$A" EFFORT) y mcode tiene ${G:-desconocido}" >&2
  echo "$A lanzado en $P con $(herramienta "$A")"
done
espera 15
for A in $L; do nombrar "$A"; done
echo "listo: workspace $W, panes en $PANES"
