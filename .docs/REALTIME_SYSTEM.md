# Arquitectura del Sistema de Tiempo Real y Colaboración Híbrida

Este documento describe el funcionamiento técnico del sistema de notificaciones en tiempo real de PMT, el cual permite que las acciones de los asistentes de IA (vía MCP) y las de los usuarios humanos (vía Web) se sincronicen instantáneamente.

## 1. El Desafío de los Dos Mundos

El sistema PMT opera en dos procesos de sistema operativo totalmente independientes:
1.  **Servidor API (`cmd/api`):** Mantiene las conexiones WebSocket activas con los navegadores de los usuarios.
2.  **Servidor MCP (`bin/pmt-mcp`):** Es lanzado por los agentes de IA (Gemini, Claude) cada vez que realizan una tarea.

**El Problema:** El proceso MCP no puede hablar con el Hub de WebSockets porque no comparten memoria. Cuando un asistente crea una tarea, el servidor API no lo sabe de forma natural.

---

## 2. La Solución: El Puente de PostgreSQL (Pub/Sub)

Para unir ambos procesos, se decidió utilizar PostgreSQL como un **Bus de Mensajes** mediante los comandos `LISTEN` y `NOTIFY`.

### Flujo de un Evento:
1.  **Origen (MCP):** El asistente ejecuta una acción (ej. crear tarea).
2.  **Notificación (`PgNotifier`):** El servicio de aplicación llama al `PgNotifier`. Este convierte el evento en un JSON plano y ejecuta `SELECT pg_notify('pmt_events', payload)`.
3.  **Tránsito (Postgres):** La base de datos recibe el mensaje y lo retransmite a todos los clientes que estén escuchando el canal `pmt_events`.
4.  **Recepción (`PgListener`):** El proceso de la API tiene un hilo (goroutine) ejecutando un `LISTEN`. Al recibir el mensaje, lo decodifica.
5.  **Despacho (`Hub`):** El `PgListener` entrega el mensaje al `Hub` de WebSockets.
6.  **Destino (Navegador):** El `Hub` identifica a qué usuario pertenece el mensaje y lo envía por el socket abierto.

---

## 3. Componentes del Backend (Go)

### A. El Emisor: `internal/adapter/driven/postgres/pg_notifier.go`
Utiliza una estructura de datos simplificada para evitar errores de serialización de Go (campos privados):
- **Estrategia:** Convierte los datos a un `map[string]string`. Esto garantiza que los UUIDs y textos viajen íntegros sin que Go los borre al intentar convertirlos a JSON.

### B. El Receptor: `internal/adapter/driven/postgres/pg_listener.go`
Es un componente "todoterreno":
- Corre en segundo plano desde el arranque del servidor API.
- **Resiliencia:** Está programado para intentar decodificar el mensaje en múltiples formatos. Si el formato cambia en el futuro, el receptor intentará encontrar los campos `owner_id` y `event_name` de forma inteligente.

### C. El Gestor: `internal/adapter/driving/httpserver/ws/hub.go`
Administra el ciclo de vida de las conexiones:
- Mantiene un mapa de `ownerID -> conexiones`.
- Implementa un sistema de despacho no bloqueante para evitar que un cliente lento ralentice a los demás (**Deadlock Prevention**).

---

## 4. Componentes del Frontend (React)

### A. Autenticación y Cookies (`ws/handler.go` + Axios)
Para que el WebSocket se conecte, el navegador debe enviar la cookie de sesión.
- **Decisión de Seguridad:** Se cambió la política de cookies a **`SameSite: Lax`**. Esto es necesario porque el frontend y el backend operan en diferentes puertos (`5173` y `8080`), y el modo `Strict` bloqueaba la conexión inicial del WebSocket.

### B. El Proveedor: `src/core/ws/WebSocketProvider.tsx`
Es el corazón de la reactividad en el cliente:
- **Conexión Automática:** Se conecta al cargar la app y se reconecta si el servidor cae.
- **Invalidación Agresiva:** No intenta ser "quirúrgico". Si llega cualquier evento de tipo `issue.*`, marca todas las consultas de la caché que empiecen por `['projects']` como obsoletas. 
- **Resultado:** TanStack Query detecta el cambio y vuelve a pedir los datos al servidor instantáneamente, actualizando la pantalla sin que el usuario presione F5.

---

## 5. Guía de Mantenimiento

### ¿Cómo agregar un nuevo evento?
1.  En el servicio de aplicación correspondiente (ej. `comment/service.go`), llama a `s.notifier.Notify(ownerID, event)`.
2.  Asegúrate de que el `Payload` sea un mapa simple o una estructura con campos públicos (Mayúsculas).
3.  En el frontend, añade el nombre del evento al `switch` en `handleEvent` dentro de `WebSocketProvider.tsx` si requiere una lógica de refresco distinta.

### ¿Cómo depurar si falla?
1.  Verifica la terminal de la API: Busca errores de "PgListener".
2.  Verifica la terminal de Postgres: Puedes ejecutar `LISTEN pmt_events;` en una consola de SQL para ver si los mensajes están viajando.
3.  Verifica la consola F12: Los mensajes que llegan al navegador se pueden ver activando temporalmente los `console.log` en el `WebSocketProvider`.

---
*Documentación generada el 12 de Abril de 2026 tras la validación exitosa del sistema real-time.*
