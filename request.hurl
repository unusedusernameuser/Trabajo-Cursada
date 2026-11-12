# Pruebas de la aplicación renderizada en el servidor (HTML, no JSON).
# Requiere el servidor corriendo en http://localhost:8080 (ver README).
# Ejecutar con:  hurl --test request.hurl
# Hurl no sigue redirects por defecto, así que se verifica el 303 y el header Location.

# 1. La página principal responde HTML con el formulario
GET http://localhost:8080/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
body contains "Nuevo usuario"
body contains "pico.min.css"

# 2. Crear un usuario -> 303 y redirect a /
POST http://localhost:8080/users
[Form]
handle: Juan
display_name: Juan Pérez
email: Juan@Example.com
password: hash_de_ejemplo
HTTP 303
[Asserts]
header "Location" == "/"

# 3. Tras el redirect, GET / muestra el usuario (handle y email en minúsculas)
GET http://localhost:8080/
HTTP 200
[Asserts]
body contains "@juan"
body contains "Juan Pérez"
body contains "juan@example.com"
body not contains "hash_de_ejemplo"

# 4. Datos inválidos (handle vacío) -> 400 y se vuelve a mostrar el formulario con el error
POST http://localhost:8080/users
[Form]
handle:
display_name: Sin Handle
email: sin.handle@example.com
password: hash_de_ejemplo
HTTP 400
[Asserts]
body contains "obligatorio"
body contains "Sin Handle"

# 5. Handle repetido -> 409 con mensaje de error
POST http://localhost:8080/users
[Form]
handle: juan
display_name: Otro Juan
email: otro@example.com
password: hash_de_ejemplo
HTTP 409
[Asserts]
body contains "ya existe un usuario"

# 6. Los datos se escapan (no se interpretan como HTML)
POST http://localhost:8080/users
[Form]
handle: xss
display_name: <script>alert(1)</script>
email: xss@example.com
password: hash_de_ejemplo
HTTP 303

GET http://localhost:8080/
HTTP 200
[Asserts]
body not contains "<script>alert(1)</script>"
body contains "&lt;script&gt;"

# 7. Eliminar un id que no existe -> también 303 (DELETE idempotente, como en la entrega anterior)
POST http://localhost:8080/users/2147483647/delete
HTTP 303
[Asserts]
header "Location" == "/"

# 8. Id inválido al eliminar -> 400
POST http://localhost:8080/users/abc/delete
HTTP 400

# 9. Una ruta inexistente muestra la página 404
GET http://localhost:8080/nada
HTTP 404
[Asserts]
body contains "Página no encontrada"
