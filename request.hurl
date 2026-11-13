# Pruebas de la API (usuarios, carpetas, tags, tareas).
# Requiere el servidor corriendo en http://localhost:8080 (ver README).
# Ejecutar con:  hurl --test request.hurl

# 1. Crear un usuario -> 201 y devuelve el nuevo objeto
POST http://localhost:8080/users
Content-Type: application/json
{
  "handle": "Juan",
  "display_name": "Juan Pérez",
  "email": "Juan@Example.com",
  "password_hash": "hash_de_ejemplo"
}
HTTP 201
[Captures]
user_id: jsonpath "$.id"
[Asserts]
jsonpath "$.handle" == "juan"
jsonpath "$.display_name" == "Juan Pérez"
jsonpath "$.email" == "juan@example.com"

# 2. Crear con datos inválidos (handle vacío) -> 400
POST http://localhost:8080/users
Content-Type: application/json
{
  "handle": "",
  "display_name": "Sin Handle",
  "email": "sin.handle@example.com",
  "password_hash": "hash_de_ejemplo"
}
HTTP 400

# 3. Crear un usuario con handle o email repetido -> 409
POST http://localhost:8080/users
Content-Type: application/json
{
  "handle": "juan",
  "display_name": "Otro Juan",
  "email": "otro@example.com",
  "password_hash": "hash_de_ejemplo"
}
HTTP 409

# 4. Listar usuarios -> 200 y un array JSON que incluye al usuario creado
GET http://localhost:8080/users
HTTP 200
[Asserts]
jsonpath "$" isCollection
jsonpath "$" count >= 1

# 5. Obtener el usuario por id -> 200
GET http://localhost:8080/users/{{user_id}}
HTTP 200
[Asserts]
jsonpath "$.id" == {{user_id}}
jsonpath "$.handle" == "juan"

# 6. Obtener un id que no existe -> 404
GET http://localhost:8080/users/2147483647
HTTP 404

# 7. Actualizar el usuario -> 200 con los datos nuevos
PUT http://localhost:8080/users/{{user_id}}
Content-Type: application/json
{
  "handle": "juanp",
  "display_name": "Juan P.",
  "email": "juanp@example.com"
}
HTTP 200
[Asserts]
jsonpath "$.id" == {{user_id}}
jsonpath "$.handle" == "juanp"
jsonpath "$.display_name" == "Juan P."
jsonpath "$.email" == "juanp@example.com"

# 8. Actualizar con datos inválidos (display_name vacío) -> 400
PUT http://localhost:8080/users/{{user_id}}
Content-Type: application/json
{
  "handle": "juanp",
  "display_name": "",
  "email": "juanp@example.com"
}
HTTP 400

# 9. Actualizar un id que no existe -> 404
PUT http://localhost:8080/users/2147483647
Content-Type: application/json
{
  "handle": "fantasma",
  "display_name": "Fantasma",
  "email": "fantasma@example.com"
}
HTTP 404

# 10. Eliminar el usuario -> 204 sin cuerpo
DELETE http://localhost:8080/users/{{user_id}}
HTTP 204

# 11. Verificar que ya no existe -> 404
GET http://localhost:8080/users/{{user_id}}
HTTP 404


# ======================================================================
# Carpetas, tags y tareas
# Todas las rutas cuelgan de /users/{user_id} (quien consulta o modifica).
# ======================================================================

# 12. Crear dos usuarios: ana y beto
POST http://localhost:8080/users
Content-Type: application/json
{ "handle": "ana", "display_name": "Ana", "email": "ana@example.com", "password_hash": "h" }
HTTP 201
[Captures]
ana_id: jsonpath "$.id"

POST http://localhost:8080/users
Content-Type: application/json
{ "handle": "beto", "display_name": "Beto", "email": "beto@example.com", "password_hash": "h" }
HTTP 201
[Captures]
beto_id: jsonpath "$.id"

# ---- Carpetas ----

# 13. Crear carpeta -> 201
POST http://localhost:8080/users/{{ana_id}}/folders
Content-Type: application/json
{ "folder_name": "Facultad" }
HTTP 201
[Captures]
folder_id: jsonpath "$.id"
[Asserts]
jsonpath "$.folder_name" == "Facultad"
jsonpath "$.user_id" == {{ana_id}}

# 14. Carpeta repetida -> 409
POST http://localhost:8080/users/{{ana_id}}/folders
Content-Type: application/json
{ "folder_name": "Facultad" }
HTTP 409

# 15. Carpeta con nombre vacío -> 400
POST http://localhost:8080/users/{{ana_id}}/folders
Content-Type: application/json
{ "folder_name": "  " }
HTTP 400

# 16. Carpeta para un usuario que no existe -> 404
POST http://localhost:8080/users/2147483647/folders
Content-Type: application/json
{ "folder_name": "Fantasma" }
HTTP 404

# 17. Listar y obtener carpetas
GET http://localhost:8080/users/{{ana_id}}/folders
HTTP 200
[Asserts]
jsonpath "$" count == 1

GET http://localhost:8080/users/{{ana_id}}/folders/{{folder_id}}
HTTP 200
[Asserts]
jsonpath "$.id" == {{folder_id}}

# 18. Otro usuario no ve la carpeta -> 404
GET http://localhost:8080/users/{{beto_id}}/folders/{{folder_id}}
HTTP 404

# 19. Renombrar la carpeta -> 200
PUT http://localhost:8080/users/{{ana_id}}/folders/{{folder_id}}
Content-Type: application/json
{ "folder_name": "Cursada" }
HTTP 200
[Asserts]
jsonpath "$.folder_name" == "Cursada"

# ---- Tags ----

# 20. Crear tag -> 201, repetido -> 409
POST http://localhost:8080/users/{{ana_id}}/tags
Content-Type: application/json
{ "tag_name": "urgente" }
HTTP 201
[Captures]
tag_id: jsonpath "$.id"
[Asserts]
jsonpath "$.tag_name" == "urgente"

POST http://localhost:8080/users/{{ana_id}}/tags
Content-Type: application/json
{ "tag_name": "urgente" }
HTTP 409

# 21. Listar, obtener y renombrar tags
GET http://localhost:8080/users/{{ana_id}}/tags
HTTP 200
[Asserts]
jsonpath "$" count == 1

GET http://localhost:8080/users/{{ana_id}}/tags/{{tag_id}}
HTTP 200

PUT http://localhost:8080/users/{{ana_id}}/tags/{{tag_id}}
Content-Type: application/json
{ "tag_name": "importante" }
HTTP 200
[Asserts]
jsonpath "$.tag_name" == "importante"

# 22. Otro usuario no ve el tag -> 404
GET http://localhost:8080/users/{{beto_id}}/tags/{{tag_id}}
HTTP 404

# ---- Tareas ----

# 23. Crear tarea con carpeta -> 201
POST http://localhost:8080/users/{{ana_id}}/tasks
Content-Type: application/json
{
  "title": "Armar informe",
  "task_description": "Primera versión",
  "folder_id": {{folder_id}},
  "due_date": "2030-01-01T00:00:00Z",
  "task_priority": "alta"
}
HTTP 201
[Captures]
task_id: jsonpath "$.id"
[Asserts]
jsonpath "$.title" == "Armar informe"
jsonpath "$.task_status" == "pendiente"
jsonpath "$.task_priority" == "alta"
jsonpath "$.folder_id" == {{folder_id}}
jsonpath "$.completed_at" == null

# 24. Crear con datos inválidos -> 400
POST http://localhost:8080/users/{{ana_id}}/tasks
Content-Type: application/json
{ "title": "" }
HTTP 400

POST http://localhost:8080/users/{{ana_id}}/tasks
Content-Type: application/json
{ "title": "Prioridad rara", "task_priority": "urgentisima" }
HTTP 400

# 25. Carpeta de otro usuario -> 400
POST http://localhost:8080/users/{{beto_id}}/tasks
Content-Type: application/json
{ "title": "Carpeta ajena", "folder_id": {{folder_id}} }
HTTP 400

# 26. Ana ve su tarea; otro usuario (o uno inexistente) no -> 404
GET http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}
HTTP 200
[Asserts]
jsonpath "$.id" == {{task_id}}

GET http://localhost:8080/users/{{beto_id}}/tasks/{{task_id}}
HTTP 404

GET http://localhost:8080/users/2147483647/tasks/{{task_id}}
HTTP 404

# 27. Ana completa la tarea: completed_at se completa; al reabrir vuelve a null; otro usuario no puede -> 404
PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/status
Content-Type: application/json
{ "task_status": "completada" }
HTTP 200
[Asserts]
jsonpath "$.task_status" == "completada"
jsonpath "$.completed_at" != null

PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/status
Content-Type: application/json
{ "task_status": "en_progreso" }
HTTP 200
[Asserts]
jsonpath "$.completed_at" == null

PUT http://localhost:8080/users/{{beto_id}}/tasks/{{task_id}}/status
Content-Type: application/json
{ "task_status": "completada" }
HTTP 404

PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/status
Content-Type: application/json
{ "task_status": "terminada" }
HTTP 400

# 28. Actualizar la tarea (solo el dueño)
PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}
Content-Type: application/json
{ "title": "Armar informe final", "task_description": null, "due_date": "2030-02-01T00:00:00Z", "task_priority": "baja" }
HTTP 200
[Asserts]
jsonpath "$.title" == "Armar informe final"
jsonpath "$.task_priority" == "baja"
jsonpath "$.task_description" == null

PUT http://localhost:8080/users/{{beto_id}}/tasks/{{task_id}}
Content-Type: application/json
{ "title": "Hackeada", "task_priority": "baja" }
HTTP 404

# 29. Listados: por usuario (con filtros) y por carpeta; beto no ve nada de ana
GET http://localhost:8080/users/{{ana_id}}/tasks
HTTP 200
[Asserts]
jsonpath "$" count == 1

GET http://localhost:8080/users/{{ana_id}}/tasks?status=en_progreso&priority=baja&folder_id={{folder_id}}
HTTP 200
[Asserts]
jsonpath "$" count == 1

GET http://localhost:8080/users/{{ana_id}}/tasks?status=completada
HTTP 200
[Asserts]
jsonpath "$" count == 0

GET http://localhost:8080/users/{{ana_id}}/tasks?status=terminada
HTTP 400

GET http://localhost:8080/users/{{ana_id}}/folders/{{folder_id}}/tasks
HTTP 200
[Asserts]
jsonpath "$" count == 1

GET http://localhost:8080/users/{{beto_id}}/tasks
HTTP 200
[Asserts]
jsonpath "$" count == 0

# 30. Tareas vencidas: una con fecha pasada aparece, la futura no
POST http://localhost:8080/users/{{ana_id}}/tasks
Content-Type: application/json
{ "title": "Vencida", "due_date": "2020-01-01T00:00:00Z" }
HTTP 201
[Captures]
overdue_id: jsonpath "$.id"

GET http://localhost:8080/users/{{ana_id}}/tasks/overdue
HTTP 200
[Asserts]
jsonpath "$" count == 1
jsonpath "$[0].id" == {{overdue_id}}

# 31. Mover a carpeta (null la saca) y adjuntar archivo
PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/folder
Content-Type: application/json
{ "folder_id": null }
HTTP 200
[Asserts]
jsonpath "$.folder_id" == null

PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/folder
Content-Type: application/json
{ "folder_id": {{folder_id}} }
HTTP 200
[Asserts]
jsonpath "$.folder_id" == {{folder_id}}

PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/attachment
Content-Type: application/json
{ "attachment_path": "/uploads/informe.pdf" }
HTTP 200
[Asserts]
jsonpath "$.attachment_path" == "/uploads/informe.pdf"

# ---- Tags de una tarea ----

# 32. Asociar el tag a la tarea, listar y buscar tareas por tag
PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/tags/{{tag_id}}
HTTP 200
[Asserts]
jsonpath "$" count == 1
jsonpath "$[0].tag_name" == "importante"

GET http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/tags
HTTP 200
[Asserts]
jsonpath "$" count == 1

GET http://localhost:8080/users/{{ana_id}}/tags/{{tag_id}}/tasks
HTTP 200
[Asserts]
jsonpath "$" count == 1
jsonpath "$[0].id" == {{task_id}}

# 33. Un tag de otro usuario no se puede asociar -> 404
PUT http://localhost:8080/users/{{beto_id}}/tasks/{{task_id}}/tags/{{tag_id}}
HTTP 404

# 34. Quitar el tag y limpiar tags
DELETE http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/tags/{{tag_id}}
HTTP 204

GET http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/tags
HTTP 200
[Asserts]
jsonpath "$" count == 0

PUT http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/tags/{{tag_id}}
HTTP 200

DELETE http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/tags
HTTP 204

GET http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}/tags
HTTP 200
[Asserts]
jsonpath "$" count == 0

# ---- Bajas ----

# 35. Borrar tarea, tag y carpeta
DELETE http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}
HTTP 204

GET http://localhost:8080/users/{{ana_id}}/tasks/{{task_id}}
HTTP 404

DELETE http://localhost:8080/users/{{ana_id}}/tags/{{tag_id}}
HTTP 204

DELETE http://localhost:8080/users/{{ana_id}}/folders/{{folder_id}}
HTTP 204

# 36. Borrar los usuarios de prueba (también borra en cascada sus datos restantes)
DELETE http://localhost:8080/users/{{ana_id}}
HTTP 204

DELETE http://localhost:8080/users/{{beto_id}}
HTTP 204
