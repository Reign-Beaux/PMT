# Importar proyectos desde un fichero

La herramienta MCP `import_project` crea un proyecto nuevo con todas sus etapas y tareas a partir de un fichero YAML. Sustituye las llamadas una por una a `create_project`, `create_stage` y `create_task`.

## Flujo de uso

1. El usuario platica el proyecto con PM.
2. PM escribe el fichero YAML en la ubicación que indique el usuario.
3. El usuario lo revisa.
4. PM llama a `import_project` con la ruta absoluta.

## Formato

```yaml
project:
  name: Tienda en línea
  description: Venta de productos por internet.
  stages:
    - name: Base de datos
      description: Modelo de datos inicial.
      tasks:
        - title: Crear tabla de productos
          description: |
            Debe guardar nombre, precio y existencias.
        - title: Crear tabla de pedidos
          description: Debe relacionarse con productos.
    - name: API
      description: Rutas del backend.
      tasks: []
```

- El fichero contiene un solo documento YAML.
- El orden del fichero es el orden de las etapas y de las tareas.
- `name` o `title` y `description` son obligatorios y no pueden estar vacíos en los tres niveles.
- `stages` y `tasks` pueden faltar o ser listas vacías.
- No hay campos de status ni de versión: el proyecto y las etapas nacen en `planning`, las tareas en `backlog` y `product_version` vacía.
- Una clave desconocida es un error, para detectar erratas.

## Cómo escribir los textos

Todo texto va entre comillas: dobles en general y simples si lleva `\`. Las descripciones largas van con `|`.

| Caso | Mal | Bien |
| --- | --- | --- |
| Dos puntos | `title: Login: pantalla` | `title: "Login: pantalla"` |
| Empieza con `*` | `title: *urgente*` | `title: "*urgente*"` |
| Empieza con `[` | `title: [API] rutas` | `title: "[API] rutas"` |
| Empieza con `#` | `title: #1 prioridad` | `title: "#1 prioridad"` |
| Barra invertida | `title: "C:\qa"` | `title: 'C:\qa'` |
| Espacio seguido de `#` | `title: Revisar API # urgente` | `title: "Revisar API # urgente"` |

Un comentario junto a `name`, `title` o `description` se rechaza, porque YAML lo quitaría del texto sin avisar. Eso incluye el que sigue a un texto entre comillas y el que va en la línea del `|`. Un `#` dentro de las comillas o dentro de un bloque `|` es texto. Los comentarios en su propia línea y los de otras claves, como `tasks: # pendientes`, se permiten.

## Comportamiento

| Aspecto | Regla |
| --- | --- |
| Entrada | `path`, ruta absoluta. La lee `pmt-server`, en la máquina donde corre. |
| Alcance | Solo crea proyectos nuevos. Importar dos veces el mismo fichero crea dos proyectos. |
| Atomicidad | Valida el fichero completo y guarda en una sola transacción. Si algo falla, no se crea nada y el fichero se puede corregir y volver a cargar. |
| Salida | `project_id`, `project_name`, `stage_count` y `task_count`. |
| Interfaz | Solo MCP, sin ruta REST. |

## Importar una etapa

La herramienta MCP `import_stage` crea una etapa con sus tareas en un proyecto en `working`. La etapa nace en su planning propio, como con `add_stage`: sus tareas no generan registros y su registro `add` se escribe al salir de `planning`.

```yaml
stage:
  name: "Pagos"
  description: "Cobro con tarjeta."
  tasks:
    - title: "Crear tabla de pagos"
      description: "Debe relacionarse con pedidos."
```

| Aspecto | Regla |
| --- | --- |
| Entrada | `project_id`, `path`, `reason` y `requested_by`. |
| Proyecto | Debe estar en `working`; si no, responde `el proyecto no está en working`. |
| Formato | Las mismas reglas de escritura y de claves. `name` y `description` son obligatorios; `tasks` puede faltar. |
| Atomicidad | Valida el fichero completo y guarda en una sola transacción. Si algo falla, no se crea nada. Importar dos veces crea dos etapas. |
| Salida | `project_id`, `stage_id`, `stage_name` y `task_count`. |
| Interfaz | Solo MCP, sin ruta REST. |

Los errores indican su ubicación como `etapa: ...` y `tarea 2: ...`.

## Errores

Los errores del fichero son `kernel.Invalid` e indican su ubicación:

| Caso | Ejemplo |
| --- | --- |
| Campo obligatorio | `etapa 3, tarea 5: el título de la tarea es obligatorio` |
| Comentario junto a un texto | `etapa 2, tarea 4: el texto contiene un comentario; ponlo entre comillas` |
| Dos puntos sin comillas | `línea 8: el texto contiene ":"; ponlo entre comillas` |
| Empieza con un carácter especial | `línea 8: el texto empieza con un carácter especial; ponlo entre comillas` |
| Indentación o texto sin comillas | `línea 7: revisa la indentación o pon el texto entre comillas` |
| Barra invertida entre comillas dobles | `línea 8: barra invertida no válida; usa comillas simples` |
| Tabulador o carácter no válido | `línea 2: el fichero tiene un tabulador o carácter no válido; usa espacios` |
| Clave desconocida | `línea 4: clave desconocida "nombre"` |
| Otro error de YAML | `línea 4: el YAML no es válido` |
| Fichero ilegible | `no se pudo leer el fichero: open /ruta/proyecto.yaml: no such file or directory` |

La línea es la que informa `yaml.v3` y no siempre es la del error. En los casos de indentación y de tabulador puede señalar el inicio del bloque que contiene el problema o la línea anterior, así que hay que revisar desde ahí hacia abajo. Si un error no tiene línea, el mensaje la omite.
