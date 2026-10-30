# Glosario

Lenguaje ubicuo de PMT. La documentación usa el término en español y el código usa su nombre en inglés.

## Términos de diseño

| Término | Significado | En Go | Dónde vive |
| --- | --- | --- | --- |
| **Entidad** | Concepto del negocio con identidad propia, que perdura aunque cambien sus datos. | `struct` con ID | `domain/` |
| **Objeto de valor** | Concepto definido solo por su valor, sin identidad propia. | Tipo o `struct` sin ID | `domain/` o `kernel/` |
| **Agregado** | Grupo de entidades y objetos de valor que cambia como una unidad y protege sus reglas. | `struct` y sus métodos | `domain/` |
| **Raíz del agregado** | Entidad por la que se accede y se modifica todo el agregado. | `struct` principal | `domain/` |
| **Registro** | Fila de una tabla, representada en Go. No contiene reglas de negocio. | `struct` generado por `sqlc` | `store/sqlcgen/` |

## Términos del negocio

| Término | En código | Significado |
| --- | --- | --- |
| **Proyecto** | `Project` | Trabajo que se gestiona en PMT y se divide en etapas. Es la raíz de su propio agregado; sus etapas son agregados aparte. |
| **Etapa** | `Stage` | Parte de un proyecto que agrupa tareas. |
| **Tarea** | `Task` | Unidad de trabajo de una etapa. |
| **Agente** | `Agent` | Trabajador de IA que opera en herdr y se comunica con PMT por MCP. |
| **Perfil** | `Profile` | Conjunto de responsabilidades de quien actúa: PP, PC, TL, DevFrontend, DevBackend, CR, QAP o QA. |
| **Actor** | `Actor` | Quien realiza una operación, sea agente o usuario humano, con su perfil. |
