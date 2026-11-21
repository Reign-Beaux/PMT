# Para retomar (estado al 3 de noviembre de 2026)

## Qué estamos haciendo

Primer proyecto real con el kit de agentes: el frontend de PMT v4.

- Proyecto PMT `854005e9-713f-405e-b15a-52bb31d3d6e2`, versión 0.4.0, 5 etapas. TL de todas: `pmt-frontend-tl`.
- Workspace herdr `pmt-frontend`. Solo queda abierta la pestaña PP (Codex con su perfil, por `agentes/lanzar.sh pp`). El resto del equipo se levanta con `agentes/iniciar.sh`.
- `.agentes.env` y `.agentes/` en la raíz de v4. `.agentes/panes.env` apunta a pestañas cerradas: `iniciar.sh` lo regenera.
- Pruebas solo en el 8090 con la base `pmt_frontend_test`. El despliegue al 8080 lo hace el usuario.

## Herramientas (7 de noviembre)

- Ya no se usa Codex ni mcode. PP, DP, TL y CR: Claude Opus `medium` (ya en `.agentes.env`). PC, Devs, QAP y QA: Qwen (`qwen3.8-max`, effort medium por la config del modelo en ~/.qwen). El kit ya soporta qwen (sin commit): MCP en `.qwen/settings.json`, enlace `.agentes/bin/qwen` para que herdr lo reconozca, `*` como prompt en avisar.sh, reinicio relanzando. Probado con PC (aviso, estados, reinicio y MCP).
- PP relanzado con Claude en `pmt-frontend` (`wE:p2`, pestaña `pp`).

## Estado de PMT (limpiado el 7 de noviembre)

- Proyecto `854005e9…` (PMT, 0.4.0) y sus 5 etapas en `planning`, cada etapa con su TL `pmt-frontend-tl`.
- Etapa 1 con las 22 tareas atómicas en `backlog`; etapas 2–5 sin tareas.
- Sin historial de ningún tipo, sin tareas canceladas, planes, hallazgos, comentarios, bugs ni equipos. La etapa 1 conserva la descripción del TL (seis grupos).
- Respaldo previo: `pg_dump` en el scratchpad de la sesión (`pmt-antes-de-limpiar.dump`, puede no sobrevivir).
- `pmt_frontend_test` vacía. Se conservan los 22 agentes con tokens.

## Lo que se aprendió en los dos intentos

1. Tareas grandes se ciclan en CR: el primer intento (2 tareas) no cerró nada en horas. Con tareas atómicas, la 01 cerró en 19 min y el grupo 02–04 en hora y media.
2. CR ejecutaba pruebas y revisaba evidencia: corregido en su perfil.
3. Devfront escribía scripts propios de Chromium (CDP) para evidencia porque el plan de DP le pedía criterios de pantalla: corregido.
4. QA armaba su propia herramienta de navegador porque Playwright quedó en la tarea 21: nueva regla, QA solo prueba frontend con Playwright. **Falta decidir quién instala Playwright y cuándo** (antes del primer grupo con pantalla).
5. PP dejaba instrucciones en comentarios de PMT ("una tarea por encargo"): corregido, la comunicación es al momento con herdr.

## Reglas decididas por el usuario

- **CR:** solo lee código del producto, no ejecuta pruebas, builds ni servidores, no revisa evidencia ni scripts de Dev. En cada hallazgo anota la corrección exacta para que Dev la aplique. Si CR se porta mal: detener todo y avisar al usuario.
- **DP:** cada criterio de aceptación dice a qué tarea pertenece y quién lo comprueba (Dev: unitarias, build, lint; QA: pantalla, API real, datos). Dev nunca comprueba en navegador. "Contra la API real" solo si la tarea consume la API. No agrega exigencias que la tarea no pide ni garantías de infraestructura. Breve.
- **Dev:** su evidencia son sus pruebas, build y lint, solo de los criterios que le tocan. Sin scripts de navegador.
- **QAP:** incluye en su plan los criterios que el plan de DP asigna a QA.
- **QA:** en frontend solo Playwright; si no está instalado, no prueba y avisa al TL.
- **PP:** planea el qué y **planea todas las tareas de forma atómica desde el inicio**, en todas las etapas. No prepara herramientas de prueba y no deja instrucciones en comentarios.
- **PC:** solo lee las pestañas de herdr para responder al usuario. No usa PMT, no avisa, no afecta el proyecto. Conserva su token (sin token PMT lo trataría como `User`).
- **Vigía:** avisa al TL, no a PC. El TL revisa el avance y levanta a quien toque.
- **TL** agrupa tareas relacionadas.

## Repaso de perfiles uno por uno (en curso)

El usuario pide la responsabilidad de cada perfil de uno en uno y solo da el siguiente cuando lo pide.

- Hecho: PP y PC (definición, documentación y código).
- TL corregido el 6 de noviembre (sin commit): una etapa por vez, el usuario o PP levantan cada sesión de TL, el TL solo cambia el status de su etapa; tarea cero de cada lado instala dependencias y herramientas de prueba y, si es solo configuración, termina en CR; nadie crea herramientas propias de prueba. Cuándo entra QAP se define al llegar a QAP.
- DP revisado el 7 de noviembre (sin commit): descripciones MCP/OpenAPI de `acceptance_criteria` alineadas ("contra la API real solo si la tarea la consume"), también en `dp.md` y `.docs/perfiles.md`; servidor reiniciado y DummyAgent lo confirmó.
- DevBackend revisado el 7 de noviembre (sin commit): sin tecnología fija (se define en la planeación); al terminar ejecuta el linter y corrige lo que marque.
- DevFrontend revisado el 7 de noviembre (sin commit): igual que DevBackend; linter "si la tecnología tiene uno". Playwright se queda para QA (sirve con cualquier frontend web, también Blazor).
- QAP revisado el 7 de noviembre (sin commit): entra si la tarea crea una funcionalidad nueva o cambia una existente, y afecta frontend y backend; casos de error de absurdos a comunes, también rechazos y fallas del backend.
- QA revisado el 7 de noviembre (sin commit): prueba según lo afectado (frontend: Playwright y regresión; MCP: solo comandos afectados; worker: agrega datos para que corra; backend: según afecte frontend, MCP o worker); siempre revisa la base con todos los accesos; limpia sus datos al terminar cada ronda; sin psql ni frontend/e2e fijos en su perfil. `psql` ya no está fijo: `.agentes.env` lleva `DB_CONSULTA`.
- Revisión final de sincronía (7 de noviembre, sin commit): descripciones MCP dicen quién usa cada comando de escritura; TL detiene/reanuda su etapa y cancela/reabre tareas; el TL es responsable de su etapa y solo pasa a PP lo que va más allá; QA lee los criterios del plan de DP.
- Hecho por Dev (sin commit): solo QA rechaza bugs y solo en `reported`; 403 documentado en el OpenAPI de planeación. Orden de ejecución: DP, DevFrontend/DevBackend, CR, QAP, QA.
- Corregido el 6 de noviembre: PP planea todas las tareas de forma atómica (`agentes/perfiles/pp.md` y `.docs/perfiles.md`). Falta relanzar a PP, mejor al terminar el repaso de todos los perfiles.
- **También se debe hacer:** las etapas 2–5 no tienen tareas activas (sus 8 tareas están canceladas). PP tiene que planearlas de nuevo, de forma atómica, como hizo con la etapa 1.

## Commits

- Todo el repaso está en `main` con push, hasta `8931e4a` (7 de noviembre). Solo queda sin commit `frontend/`.

## Hueco de permisos en el backend (cerrado el 6 de noviembre)

- Commit `f213403` (sin push): todo comando de escritura exige perfil. PP y `User` planean; `revise_stage` también el TL de su etapa; el resto `Forbidden` antes de leer la base. Prueba `TestWriteToolsRejectPC` recorre las 40 herramientas de escritura. DummyAgent con `dummy-pc` confirmó los rechazos.
- Con esto PC queda terminado en el repaso.

## Deuda técnica anotada

- Levantar agentes es complejo: decisión de un agente por perfil reutilizado en todos los proyectos (`pp`, `pc`, `tl`, `dp`, `devback`, `devfront`, `cr`, `qap`, `qa`) y cada perfil como skill. Falta decidir si se quitan los tokens.
- Falta herramienta para cargar varias etapas con sus tareas en un proyecto existente.
- `pmt-server` no arranca solo al reiniciar la máquina: propuesta de servicio de usuario de systemd (sin hacer).

## Para arrancar el viernes

1. Revisar que `pmt-server` corra en el 8080; si no: `cd backend; setsid nohup ./bin/pmt-server >> bin/pmt-server.log 2>&1 &`.
2. Seguir el repaso con el TL.
5. Commit de lo pendiente cuando el usuario lo pida.
6. Levantar al equipo con `agentes/iniciar.sh` y que PP avise al TL que inicie la etapa 1.
