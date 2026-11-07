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
    claude|codex|mcode) ;;
    *) echo "herramienta inválida para $A: $(herramienta "$A")" >&2; exit 1;;
  esac
  [ -n "$(modelo "$A")" ] || { echo "falta el modelo de $A" >&2; exit 1; }
  case "$(dato "$A" EFFORT)" in medium|high) ;; *) echo "effort inválido para $A" >&2; exit 1;; esac
  T=$(token "$A")
  [ -n "$T" ] && [ -r "$HOME/.config/pmt/tokens/$T" ] || { echo "falta el token de $A: ${T:-sin nombre}" >&2; exit 1; }
done
for V in PROYECTO_RUTA DB_URL DB_CONTENEDOR PUERTO_BACKEND PUERTO_FRONTEND PMT_URL HERDR_WORKSPACE; do
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
# mcode lo toma de .mcp.json en la raíz del proyecto.
[ -e "$PROYECTO_RUTA/.mcp.json" ] || cp "$AGENTES_DIR/mcp.json" "$PROYECTO_RUTA/.mcp.json"

# Perfil final: rol + comun.md con los datos del proyecto, para los nueve.
export KIT AGENTES_DIR PROYECTO_RUTA PROYECTO_DESCRIPCION DB_URL DB_CONTENEDOR PUERTO_BACKEND PUERTO_FRONTEND
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
python3 - "$KIT" "$AGENTES_DIR/perfiles" $TODOS <<'PY'
import os,re,sys
from urllib.parse import urlparse
kit,dest,agentes=sys.argv[1],sys.argv[2],sys.argv[3:]
u=urlparse(os.environ['DB_URL'])
v={k:os.environ.get(k,'') for k in 'KIT AGENTES_DIR PROYECTO_RUTA PROYECTO_DESCRIPCION DB_URL DB_CONTENEDOR PUERTO_BACKEND PUERTO_FRONTEND'.split()}
v.update(DB_USUARIO=u.username or '',DB_NOMBRE=u.path.lstrip('/'),TABLA_AGENTES=os.environ['TABLA'])
comun=open(f'{kit}/comun.md').read()
for a in agentes:
    t=open(f'{kit}/perfiles/{a}.md').read()
    # PC solo lee herdr: sin reglas de PMT ni de avisos.
    t+=(f"\nProyecto: `{{{{PROYECTO_RUTA}}}}`.\n\nAgentes:\n| Agente | PMT | Perfil |\n|---|---|---|\n{{{{TABLA_AGENTES}}}}\n" if a=='pc' else comun)
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
