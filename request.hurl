# Pruebas de la aplicación renderizada en el servidor (HTML, no JSON).
# Requiere el servidor corriendo en http://localhost:8080 (ver README).
# Ejecutar con:  hurl --test request.hurl
# El alta y la baja de usuarios usan HTMX: el servidor devuelve fragmentos de HTML (sin redirect)
# y los requests llevan el header HX-Request que agrega HTMX en el navegador.

# 1. La página principal responde HTML con el formulario
GET http://localhost:8080/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
body contains "Nuevo usuario"
body contains "pico.min.css"
body contains "htmx.org"
body contains "hx-post=\"/users\""
body contains "id=\"user-list\""

# 2. Crear un usuario -> 200 con solo el fragmento de la lista (sin <html>) y el form limpio (OOB)
POST http://localhost:8080/users
HX-Request: true
[Form]
handle: Juan
display_name: Juan Pérez
email: Juan@Example.com
password: hash_de_ejemplo
HTTP 200
[Asserts]
header "Location" not exists
body contains "id=\"user-list\""
body contains "@juan"
body contains "juan@example.com"
body contains "hx-swap-oob"
body not contains "<html"
body not contains "hash_de_ejemplo"

# 3. GET / también muestra el usuario (handle y email en minúsculas)
GET http://localhost:8080/
HTTP 200
[Asserts]
body contains "@juan"
body contains "Juan Pérez"
body contains "juan@example.com"
body not contains "hash_de_ejemplo"

# 4. Datos inválidos (handle vacío) -> 400, el formulario con el error y HX-Retarget al formulario
POST http://localhost:8080/users
HX-Request: true
[Form]
handle:
display_name: Sin Handle
email: sin.handle@example.com
password: hash_de_ejemplo
HTTP 400
[Asserts]
header "HX-Retarget" == "#user-form"
header "HX-Reswap" == "outerHTML"
body contains "obligatorio"
body contains "Sin Handle"

# 5. Handle repetido -> 409 con mensaje de error
POST http://localhost:8080/users
HX-Request: true
[Form]
handle: juan
display_name: Otro Juan
email: otro@example.com
password: hash_de_ejemplo
HTTP 409
[Asserts]
header "HX-Retarget" == "#user-form"
body contains "ya existe un usuario"

# 6. Los datos se escapan (no se interpretan como HTML)
POST http://localhost:8080/users
[Form]
handle: xss
display_name: <script>alert(1)</script>
email: xss@example.com
password: hash_de_ejemplo
HTTP 200
[Asserts]
body not contains "<script>alert(1)</script>"
body contains "&lt;script&gt;"

GET http://localhost:8080/
HTTP 200
[Asserts]
body not contains "<script>alert(1)</script>"
body contains "&lt;script&gt;"

# 7. Eliminar un id que no existe -> 200 (DELETE idempotente); solo viaja el contador OOB
DELETE http://localhost:8080/users/2147483647
HX-Request: true
HTTP 200
[Asserts]
body not contains "<tr"
body contains "user-count"

# 8. Id inválido al eliminar -> 400
DELETE http://localhost:8080/users/abc
HTTP 400

# 8b. La vieja ruta POST /users/{id}/delete ya no existe (ahora es 404)
POST http://localhost:8080/users/2147483647/delete
HTTP 404

# 9. Una ruta inexistente muestra la página 404
GET http://localhost:8080/nada
HTTP 404
[Asserts]
body contains "Página no encontrada"

# ---------------------------------------------------------------------------
# Página de tareas de un usuario (/users/{id}/tasks): carpetas, tags y tareas.
# Todas las acciones son POST + redirect 303 a la lista (patrón PRG).
# ---------------------------------------------------------------------------

# 10. Se crea un usuario para las pruebas y se captura su id desde el link de la lista
POST http://localhost:8080/users
[Form]
handle: tareas
display_name: Usuario de Tareas
email: tareas@example.com
password: hash_de_ejemplo
HTTP 200

GET http://localhost:8080/
HTTP 200
[Captures]
user_id: regex "href=\"/users/(\\d+)/tasks\">@tareas</a>"

# 11. La página de tareas responde HTML con los formularios
GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
body contains "Tareas de Usuario de Tareas"
body contains "Nueva tarea"
body contains "Todavía no hay tareas"

# 12. Un usuario que no existe -> 404; un id inválido -> 400
GET http://localhost:8080/users/2147483647/tasks
HTTP 404
[Asserts]
body contains "Página no encontrada"

GET http://localhost:8080/users/abc/tasks
HTTP 400

# 13. Crear una carpeta -> 303 y vuelve a la lista de tareas
POST http://localhost:8080/users/{{user_id}}/folders
[Form]
folder_name: Trabajo
HTTP 303
[Asserts]
header "Location" == "/users/{{user_id}}/tasks"

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Captures]
folder_id: regex "action=\"/users/\\d+/folders/(\\d+)/delete\""
[Asserts]
body contains "Trabajo (0)"

# 14. Carpeta repetida -> 409; nombre vacío -> 400
POST http://localhost:8080/users/{{user_id}}/folders
[Form]
folder_name: Trabajo
HTTP 409
[Asserts]
body contains "ya existe una carpeta"

POST http://localhost:8080/users/{{user_id}}/folders
[Form]
folder_name:
HTTP 400
[Asserts]
body contains "obligatorio"

# 15. Renombrar la carpeta
POST http://localhost:8080/users/{{user_id}}/folders/{{folder_id}}
[Form]
folder_name: Oficina
HTTP 303

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Asserts]
body contains "Oficina (0)"
body not contains "Trabajo (0)"

# 16. Crear un tag; tag repetido -> 409
POST http://localhost:8080/users/{{user_id}}/tags
[Form]
tag_name: urgente
HTTP 303

POST http://localhost:8080/users/{{user_id}}/tags
[Form]
tag_name: urgente
HTTP 409
[Asserts]
body contains "ya existe un tag"

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Captures]
tag_id: regex "action=\"/users/\\d+/tags/(\\d+)/delete\""
[Asserts]
body contains "urgente (0)"

# 17. Crear una tarea con carpeta, prioridad, fecha y un tag
POST http://localhost:8080/users/{{user_id}}/tasks
[Form]
title: Preparar informe
task_description: Revisar los números del trimestre
due_date: 2030-12-01T18:00
task_priority: alta
folder_id: {{folder_id}}
tag_ids: {{tag_id}}
HTTP 303
[Asserts]
header "Location" == "/users/{{user_id}}/tasks"

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Captures]
task_id: regex "action=\"/users/\\d+/tasks/(\\d+)/delete\""
[Asserts]
body contains "Preparar informe"
body contains "Revisar los números del trimestre"
body contains "01/12/2030 18:00"
body contains "Oficina (1)"
body contains "urgente (1)"
body contains "Pendiente"

# 18. Título vacío -> 400 y se vuelve a mostrar el formulario; prioridad inválida -> 400
POST http://localhost:8080/users/{{user_id}}/tasks
[Form]
title:
task_priority: media
HTTP 400
[Asserts]
body contains "el título es obligatorio"

POST http://localhost:8080/users/{{user_id}}/tasks
[Form]
title: Prioridad rara
task_priority: urgentisima
HTTP 400
[Asserts]
body contains "la prioridad debe ser"
body contains "Prioridad rara"

# 19. Una carpeta que no es del usuario no se acepta
POST http://localhost:8080/users/{{user_id}}/tasks
[Form]
title: Carpeta ajena
task_priority: media
folder_id: 2147483647
HTTP 400
[Asserts]
body contains "la carpeta debe ser del usuario"

# 20. Los textos se escapan
POST http://localhost:8080/users/{{user_id}}/tasks
[Form]
title: <script>alert(2)</script>
task_priority: baja
HTTP 303

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Asserts]
body not contains "<script>alert(2)</script>"
body contains "&lt;script&gt;alert(2)"

# 21. Filtros por la URL: carpeta, tag y estado
GET http://localhost:8080/users/{{user_id}}/tasks?folder={{folder_id}}
HTTP 200
[Asserts]
body contains "Preparar informe"
body not contains "alert(2)"
body contains "Filtrando por carpeta"

GET http://localhost:8080/users/{{user_id}}/tasks?folder=none
HTTP 200
[Asserts]
body not contains "Preparar informe"
body contains "alert(2)"

GET http://localhost:8080/users/{{user_id}}/tasks?tag={{tag_id}}
HTTP 200
[Asserts]
body contains "Preparar informe"
body not contains "alert(2)"

GET http://localhost:8080/users/{{user_id}}/tasks?status=completada
HTTP 200
[Asserts]
body contains "Ninguna tarea coincide"

# 22. Cambiar el estado conserva los filtros (campo oculto "back") y rechaza valores inválidos
POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/status
[Form]
task_status: completada
back: folder={{folder_id}}
HTTP 303
[Asserts]
header "Location" == "/users/{{user_id}}/tasks?folder={{folder_id}}"

GET http://localhost:8080/users/{{user_id}}/tasks?status=completada
HTTP 200
[Asserts]
body contains "Preparar informe"
body contains "Reabrir"

POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/status
[Form]
task_status: terminada
HTTP 400
[Asserts]
body contains "el estado debe ser"

# 23. Mover la tarea a "sin carpeta" y volver a ponerla en la carpeta
POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/folder
[Form]
folder_id:
HTTP 303

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Asserts]
body contains "Oficina (0)"
body contains "Sin carpeta (2)"

POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/folder
[Form]
folder_id: {{folder_id}}
HTTP 303

# 24. Quitar y volver a agregar el tag de la tarea
POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/tags/{{tag_id}}/delete
HTTP 303

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Asserts]
body contains "urgente (0)"

POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/tags
[Form]
tag_id: {{tag_id}}
HTTP 303

POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/tags
[Form]
tag_id: 2147483647
HTTP 404
[Asserts]
body contains "no existen o no son del usuario"

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Asserts]
body contains "urgente (1)"

# 25. Editar la tarea: página con el formulario y envío
GET http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/edit
HTTP 200
[Asserts]
body contains "Editar tarea"
body contains "Preparar informe"
body contains "2030-12-01T18:00"

POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}
[Form]
title: Informe final
task_description:
due_date:
task_priority: media
HTTP 303
[Asserts]
header "Location" == "/users/{{user_id}}/tasks"

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Asserts]
body contains "Informe final"
body not contains "Preparar informe"

POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}
[Form]
title:
task_priority: media
HTTP 400
[Asserts]
body contains "el título es obligatorio"

GET http://localhost:8080/users/{{user_id}}/tasks/2147483647/edit
HTTP 404

# 26. Eliminar la tarea, el tag y la carpeta
POST http://localhost:8080/users/{{user_id}}/tasks/{{task_id}}/delete
HTTP 303

POST http://localhost:8080/users/{{user_id}}/tags/{{tag_id}}/delete
HTTP 303

POST http://localhost:8080/users/{{user_id}}/folders/{{folder_id}}/delete
HTTP 303

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 200
[Asserts]
body not contains "Informe final"
body not contains "Oficina"
body not contains "urgente"

# 27. Se elimina el usuario de prueba (también se borran sus tareas, carpetas y tags)
DELETE http://localhost:8080/users/{{user_id}}
HX-Request: true
HTTP 200
[Asserts]
body not contains "<tr"

GET http://localhost:8080/
HTTP 200
[Asserts]
body not contains "@tareas"

GET http://localhost:8080/users/{{user_id}}/tasks
HTTP 404

# 28. Limpieza: se eliminan los usuarios creados en los tests 2 y 6 para poder volver a correr este archivo
GET http://localhost:8080/
HTTP 200
[Captures]
juan_id: regex "href=\"/users/(\\d+)/tasks\">@juan</a>"
xss_id: regex "href=\"/users/(\\d+)/tasks\">@xss</a>"

DELETE http://localhost:8080/users/{{juan_id}}
HX-Request: true
HTTP 200

DELETE http://localhost:8080/users/{{xss_id}}
HX-Request: true
HTTP 200

GET http://localhost:8080/
HTTP 200
[Asserts]
body not contains "@juan"
body not contains "@xss"
