# Kit de agentes

Levanta el equipo de nueve agentes de PMT sobre un proyecto real. Requiere `herdr`, `pmt-server` corriendo, y `claude`, `codex` o `mcode` según lo que elijas. Corre desde un pane de herdr.

## Arrancar un proyecto

1. **Tokens.** Crea un agente por perfil y guarda su token en un archivo con el mismo nombre. El comando escribe solo el token y no se puede volver a obtener:

   ```bash
   cd backend
   umask 077
   ./bin/pmt-server agent create --name mi-proyecto-pp --profile PP > ~/.config/pmt/tokens/mi-proyecto-pp
   ```

   Repite con `PC`, `TL`, `DP`, `DevBackend`, `DevFrontend`, `CR`, `QAP` y `QA`.

2. **Configuración.** Copia `agentes.env.example` a `.agentes.env` en la raíz del proyecto y ajústalo. Añade `.agentes.env` y `.agentes/` al `.gitignore` del proyecto: la cadena de base de datos queda ahí.

3. **Effort de mcode.** Es global: se fija en `defaultModelThinking.effort` de `~/.minimax/config.yaml`. `iniciar.sh` avisa si difiere del `.env`.

4. **Agentes.** Crea las pestañas que falten, guarda `.agentes/panes.env`, genera los perfiles en `.agentes/perfiles/`, crea `.mcp.json` si no existe y lanza a todos:

   ```bash
   /ruta/a/agentes/iniciar.sh /ruta/al/proyecto/.agentes.env
   ```

   Para levantar solo algunos, añade sus nombres al final. Un agente que ya corre no se toca.

5. **PP.** Abre su pestaña en herdr y descríbele el proyecto. Con tu visto bueno libera y avisa al TL.

6. **Vigía (opcional).** Desde la raíz del proyecto, en una pestaña aparte: `/ruta/a/agentes/vigia.sh`.

## Scripts

Toman la configuración de `AGENTES_ENV`, que los agentes heredan, o del `.agentes.env` más cercano.

| Script | Qué hace |
| --- | --- |
| `avisar.sh <agente> "mensaje"` | Envía un aviso y confirma que llegó. Reintenta una vez sin duplicarlo; si no se confirma, deja una línea en `para-usuario.md` y sale con error. Tarda segundos, unos 10 como máximo |
| `reiniciar.sh <agente>` | Sesión vacía: `/new` en Claude y mcode, relanzar en Codex |
| `levantar.sh <agente> "encargo"` | Reinicia y da el encargo |
| `relanzar-dp.sh <medium\|high>` | Relanza a DP; el effort se conserva al reiniciar |
| `relanzar-mcode.sh <agente\|todos>` | Relanza mcode para que lea su configuración |
| `vigia.sh` | Avisa al TL si todos llevan un rato quietos |

Los avisos van a los agentes de este proyecto por su pane. `HERDR_PREFIJO` solo evita choques de nombre si hay otro equipo vivo.

Codex usa `~/.codex-flow`, porque `~/.codex` tiene un `pmt` de la v3. Debe tener `[mcp_servers.pmt]` con `bearer_token_env_var = "PMT_TOKEN"`. `lanzar.sh` usa `allow_login_shell=false`, porque el shell de login no carga `nvm` y Codex no vería `node` ni `pnpm`.

## Tras un reinicio del equipo

PostgreSQL puede volver solo, pero `pmt-server` no. Herdr intenta restaurar Codex con `codex resume` sin `CODEX_HOME`, y falla. Para retomar:

1. Arranca `pmt-server`.
2. Relanza cada agente con su sesión, en su pane y con `AGENTES_ENV` y `PMT_TOKEN` como los pone `lanzar.sh`:
   - Codex: `CODEX_HOME=~/.codex-flow codex resume <id>` con los mismos `-c` de `lanzar.sh` (perfil, effort, shell y menú), con los ID en `~/.codex-flow/sessions`. El MCP lo toma de `~/.codex-flow` y de `PMT_TOKEN`.
   - mcode: `mcode --session <id>`, con los ID en `~/.minimax/v2/sessions`.
3. El que no tenga trabajo abierto puede volver con sesión vacía: `iniciar.sh <.env> <agente>`.
