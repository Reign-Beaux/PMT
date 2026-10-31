# Deuda técnica

| Prioridad | Deuda | Riesgo | Cuándo resolverla |
| --- | --- | --- | --- |
| Media | OpenAPI sin validación contra el código. | Construir el frontend sobre un contrato desalineado. | Antes del frontend, empezando por una prueba que compare las rutas registradas contra el OpenAPI. |
| Media | `store` sin pruebas automáticas contra PostgreSQL. | Errores como el código 23001 solo aparecen en pruebas manuales. | Antes de que `tasks` tenga transiciones. |
| Baja | Las creaciones simultáneas de etapas en un proyecto reciben 409 y deben reintentarse; en una prueba, 13 de 20. | Las creaciones fallan si quien llama no reintenta. | Si varios agentes crean etapas a la vez. |
| Baja | `GET /api/projects/{project_id}/stages` responde `[]` si el proyecto no existe, en vez de 404. | Quien llama no distingue un proyecto sin etapas de uno inexistente. | Con la primera colaboración entre features. |
| Baja | El cierre de `pmt-server` agota 10 segundos y sale con código 1 si hay una petición en curso. Una conexión inactiva no lo provoca. Causa del caso observado no comprobada. | El cierre tarda más y parece un fallo. | Al trabajar más en el MCP. |
| Muy baja | El esquema de `list_projects` permite `null` en `projects`, aunque siempre devuelve un arreglo. | Los clientes del esquema contemplan un caso que no ocurre. | Al tocar esa herramienta. |
