# Deuda técnica

| Prioridad | Deuda | Riesgo | Cuándo resolverla |
| --- | --- | --- | --- |
| Media | Levantar un agente es complejo. Cada proyecto crea su propio juego de agentes con su token, que se carga antes de lanzar, y el kit lanza cada agente con muchos parámetros para meterle su perfil. Un agente levantado a mano no sabe su perfil. Decisión del usuario: un solo agente por perfil (`pp`, `pc`, `tl`, `dp`, `devback`, `devfront`, `cr`, `qap`, `qa`), reutilizado en todos los proyectos, aunque haya dos devs del mismo perfil en paralelo; y cada perfil como una skill que el agente carga al decirle quién es. Falta decidir si se quitan los tokens. | Registros nuevos en `agents` por cada proyecto, arranques que fallan por un parámetro o un token faltante, y agentes que trabajan sin conocer su perfil. | La semana del 16 de noviembre de 2026. |
