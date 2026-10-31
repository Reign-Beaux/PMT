# Análisis: trabajo nuevo con el proyecto en `working`

Análisis previo. Nada de esto está implementado ni decidido. Parte de la regla de cambios fuera de `planning`, en desarrollo: las etapas y tareas ya no se editan ni eliminan directamente, y cada cambio se guarda como registro inmutable con `add_*`, `revise_*` y `retire_*`.

## 1. Nuevos cambios solicitados

Un proyecto en `working` recibe una petición nueva que requiere una etapa.

### Qué cubre ya la regla de cambios

`add_stage` funciona con cualquier status del proyecto, así que la etapa se puede crear con su motivo y quién la pidió.

### Qué falta definir

La etapa nace en `planning`, pero `create_task` se rechaza porque el proyecto no está en `planning`. Cada tarea tendría que entrar con `add_task` y quedaría registrada como un cambio, aunque la etapa apenas se esté definiendo.

### Dirección propuesta

Tratar la petición como una etapa nueva que se planea por sí sola, sin que el proyecto salga de `working`:

- Mientras la etapa esté en su propio `planning`, se define y edita libremente.
- Al salir de `planning`, queda bajo la misma regla de historial.
- La carga por YAML también podría servir para cargar la etapa completa.

## 2. Corrección de bugs

Los bugs no se agrupan en etapas: son trabajo aislado. Hoy toda tarea exige `stage_id`, así que un bug no puede existir como tarea sin etapa.

### Opciones

| Opción | A favor | En contra |
| --- | --- | --- |
| Tabla `bugs` propia | Pertenece directo al proyecto, tiene sus propios datos y no afecta a `tasks` | Sus status y comandos se definen aparte |
| Tarea con tipo `bug` | Reutiliza los status de la tarea | `stage_id` deja de ser obligatorio y `tasks` mezcla dos conceptos |
| Etapa fija "Bugs" | No cambia el modelo | Agrupa los bugs en una etapa, justo lo que se quiere evitar |

### Dirección propuesta: tabla `bugs`

- **Pertenencia:** `project_id`, sin etapa. `tasks` conserva su `stage_id` obligatorio.
- **Datos propios:** por ejemplo, los pasos para reproducirlo y la `product_version` donde apareció.
- **Separación:** feature propia, con `list_bugs` aparte. Los bugs no se mezclan con las tareas ni afectan al cierre de etapas.

El flujo de tareas no es un argumento para reutilizarlas: hoy tienen status, pero no transiciones ni comando para cambiarlos.

### Qué cuesta

- Definir sus status y comandos aparte.
- Aplicarles las transiciones por perfil (Dev, CR, QA) cuando se definan.
- Unir las dos tablas si se quiere ver tareas y bugs en una sola vista.

## Relación entre ambos casos

En los dos entra trabajo nuevo a un proyecto en `working`, así que conviene definirlos juntos cuando se llegue a ese paso.
