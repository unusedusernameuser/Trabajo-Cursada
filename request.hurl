# Pruebas de la API de usuarios.
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
