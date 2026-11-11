# Gestor de Tareas

Trabajo de cursada de la materia **Programación Web 2026**.

Este repositorio corresponde a la cuarta entrega (**TP4**) del proyecto **Gestor de Tareas Personales**: **"Construyendo la Interfaz de Usuario"**. Construye sobre la entrega anterior (TP3), que expuso la lógica de negocio como una API REST en Go sobre PostgreSQL, y ahora le suma un **frontend en HTML, CSS y JavaScript** que consume esa API: desde el navegador se pueden **listar, crear y eliminar usuarios**.

## Descripción del proyecto

El **Gestor de Tareas Personales** es una aplicación pensada para el día a día del usuario común: permite crear tareas, agruparlas por carpetas y tags, y organizar grupos donde varios usuarios pueden dividirse tareas en común.

### Entidades principales del dominio

- **Usuario (`User`)**: quien crea y administra tareas. Tiene `handle` (nombre de usuario único), `display_name`, `email` y `password_hash`.
- **Tarea (`Task`)**: elemento principal del sistema, con título, descripción, fecha límite, estado (`pendiente`, `en_progreso`, `completada`), prioridad (`baja`, `media`, `alta`) y un archivo adjunto opcional.
- **Carpeta (`Folder`) y Tag (`Tag`)**: dos formas de clasificar las tareas de un usuario.
- **Grupo (`Group`)**: agrupación de tareas que permite colaborar entre varios usuarios (ej. *Vacaciones*, *Proyectos*).

## Funcionalidad

- **Interfaz de usuario** (`static/`): página única que muestra la lista de usuarios, un formulario para crear uno nuevo y un botón para eliminar cada uno. Es la novedad de esta entrega y se detalla en la sección [Frontend](#frontend).
- **API REST** (`handlers/`): CRUD sobre usuarios, carpetas, tags, grupos (con sus miembros) y tareas (con sus tags). Cada entidad tiene su archivo, con validación de entrada y traducción de errores de la base de datos a códigos HTTP.
- **Capa de acceso a datos con sqlc** (`db/sqlc/`): funciones Go type-safe generadas a partir del SQL escrito en `db/queries/`.
- **Tests de integración** (`db/sqlc/*_test.go`): un archivo por cada `.sql` de `db/queries/`, contra una base Postgres real.
- **Pruebas de la API con Hurl** (`request.hurl`): recorren todos los endpoints, incluyendo los casos de error (400, 404, 409).
- **Servidor HTTP** (`main.go`): se conecta a Postgres, registra las rutas de la API y sirve el frontend desde `static/` en el puerto `8080`, con una página 404 personalizada.
- **Dockerización**: `docker-compose.yml` levanta Postgres y la aplicación (construida con el `dockerfile`).

## Frontend

El cliente está formado por tres archivos dentro de `static/`:

| Archivo          | Rol                                                                                             |
|------------------|-------------------------------------------------------------------------------------------------|
| `index.html`     | Estructura: título principal, formulario de alta (`#user-form`) y contenedor de la lista (`#user-list`) |
| `js/app.js`      | Comunicación con la API (`fetch`), armado dinámico del DOM y manejo de eventos                  |
| `css/styles.css` | Estilos con CSS Grid y Flexbox; lo usan tanto `index.html` como `404.html`                      |

### Cómo funciona

1. **Al cargar la página** se hace `GET /users` y se genera un `<li>` por cada usuario, con su nombre, `@handle`, email, fecha de alta y un botón **Eliminar**.
2. **Al enviar el formulario** se previene el comportamiento por defecto, se toman los valores de los inputs y se hace `POST /users` con un cuerpo JSON. Si sale bien, se limpia el formulario y se vuelve a pedir la lista completa para mostrar el nuevo usuario.
3. **Al hacer clic en Eliminar** se pide confirmación, se hace `DELETE /users/{id}` y se vuelve a pedir la lista. Como los botones se crean dinámicamente, se usa **delegación de eventos**: hay un único listener en `#user-list` que detecta el clic con `closest('[data-action="delete"]')` y toma el `id` desde `data-id` del `<li>`.
4. **Los errores** de la API (por ejemplo `409` por `handle` o email repetido, o `400` por validación) se muestran en pantalla con el mensaje que devuelve el servidor (`{"error": "..."}`). Si el servidor no responde, se muestra un aviso de conexión.

### Decisiones de diseño

- **Mismo origen, sin CORS**: el servidor Go sirve la API y los archivos estáticos en el mismo puerto, por eso el cliente usa rutas relativas (`/users`). Por la misma razón, la página debe abrirse desde `http://localhost:8080` y **no** directamente como archivo (`file://`).
- **`textContent` en lugar de `innerHTML`**: los datos de los usuarios se insertan como texto, así que nunca se interpretan como HTML (evita inyección de código).
- **Campo contraseña**: la API espera un `password_hash`, pero no hashea nada por su cuenta. Este cliente de práctica envía el valor tal cual; en una aplicación real el hash se calcularía en el servidor (por ejemplo con bcrypt) antes de guardarlo.

## API REST

Base URL: `http://localhost:8080`

| Método   | Ruta          | Descripción                                  | Respuesta exitosa                 |
|----------|---------------|----------------------------------------------|-----------------------------------|
| `POST`   | `/users`      | Crea un usuario                              | `201 Created` + usuario           |
| `GET`    | `/users`      | Lista todos los usuarios                     | `200 OK` + array (`[]` si no hay) |
| `GET`    | `/users/{id}` | Obtiene un usuario por `id`                  | `200 OK` + usuario                |
| `PUT`    | `/users/{id}` | Actualiza `handle`, `email` y `display_name` | `200 OK` + usuario actualizado    |
| `DELETE` | `/users/{id}` | Elimina un usuario por `id`                  | `204 No Content` (sin cuerpo)     |

El frontend utiliza `GET`, `POST` y `DELETE` sobre `/users`.

### Carpetas, tags, grupos y tareas

Como la API no tiene autenticación, el usuario que consulta o modifica va en la URL (`/users/{user_id}/...`) y las queries lo usan para validar permisos (por ejemplo, una carpeta solo la ve su dueño).

| Método | Ruta | Descripción |
|--------|------|-------------|
| `POST` `GET` | `/users/{user_id}/folders` | Crear / listar carpetas del usuario |
| `GET` `PUT` `DELETE` | `/users/{user_id}/folders/{id}` | Obtener / renombrar / borrar (`{"folder_name": ...}`) |
| `POST` `GET` | `/users/{user_id}/tags` | Crear / listar tags |
| `GET` `PUT` `DELETE` | `/users/{user_id}/tags/{id}` | Obtener / renombrar / borrar (`{"tag_name": ...}`) |
| `POST` `GET` | `/users/{user_id}/groups` | Crear grupo (el usuario queda como dueño y miembro) / listar los grupos de los que es miembro (`?owner=true`: solo los que creó) |
| `GET` | `/groups/{id}` | Obtener un grupo |
| `PUT` `DELETE` | `/users/{user_id}/groups/{id}` | Modificar / borrar (solo el dueño) |
| `POST` `GET` | `/groups/{id}/members` | Agregar miembro (`{"user_id": 3}`) / listar miembros |
| `GET` | `/groups/{id}/members/count` | Cantidad de miembros |
| `GET` `DELETE` | `/groups/{id}/members/{user_id}` | Consultar pertenencia (`{"is_member": true}`) / quitar miembro (el dueño no se puede quitar) |
| `POST` `GET` | `/users/{user_id}/tasks` | Crear tarea / listar (`?status=&priority=&folder_id=`) |
| `GET` | `/users/{user_id}/tasks/overdue` | Tareas vencidas y sin completar |
| `GET` | `/users/{user_id}/assigned-tasks` | Tareas asignadas al usuario |
| `GET` | `/users/{user_id}/folders/{folder_id}/tasks` | Tareas de una carpeta |
| `GET` | `/users/{user_id}/groups/{group_id}/tasks` | Tareas de un grupo (solo si es miembro) |
| `GET` `PUT` `DELETE` | `/users/{user_id}/tasks/{id}` | Obtener (creador, asignado o miembro del grupo) / modificar y borrar (solo el creador) |
| `PUT` | `/users/{user_id}/tasks/{id}/status` | Cambiar estado (`{"task_status": "completada"}`); completa o limpia `completed_at` |
| `PUT` | `/users/{user_id}/tasks/{id}/folder` | Mover a una carpeta (`{"folder_id": 3}`; `null` la saca) |
| `PUT` | `/users/{user_id}/tasks/{id}/assignee` | Asignar (`{"assigned_to": 3}`; `null` desasigna) |
| `PUT` | `/users/{user_id}/tasks/{id}/attachment` | Adjuntar archivo (`{"attachment_path": "..."}`; `null` lo quita) |
| `GET` `DELETE` | `/users/{user_id}/tasks/{id}/tags` | Listar los tags de la tarea / quitarlos todos |
| `PUT` `DELETE` | `/users/{user_id}/tasks/{id}/tags/{tag_id}` | Asociar / quitar un tag |
| `GET` | `/users/{user_id}/tags/{tag_id}/tasks` | Tareas que tienen un tag |

Ejemplo de `POST /users/{user_id}/tasks` (solo `title` es obligatorio; `due_date` en formato RFC 3339):

```json
{
  "title": "Armar informe",
  "task_description": "Primera versión",
  "folder_id": 1,
  "group_id": 2,
  "assigned_to": 3,
  "due_date": "2030-01-01T00:00:00Z",
  "task_priority": "alta"
}
```

Los campos opcionales ausentes se devuelven como `null`. Los `DELETE` son idempotentes: responden `204` aunque el recurso no exista o no sea del usuario (en ese caso no borran nada).

Todas las respuestas con cuerpo son JSON. Los errores tienen la forma `{"error": "mensaje"}`.

### Cuerpo de `POST /users`

```json
{
  "handle": "Juan",
  "display_name": "Juan Pérez",
  "email": "Juan@Example.com",
  "password_hash": "hash_de_ejemplo"
}
```

Respuesta (`201`). Notar que `handle` y `email` se normalizan a minúsculas y que `password_hash` nunca se devuelve:

```json
{
  "id": 1,
  "handle": "juan",
  "display_name": "Juan Pérez",
  "email": "juan@example.com",
  "created_at": { "Time": "2026-11-11T10:00:00Z", "Valid": true },
  "updated_at": { "Time": "2026-11-11T10:00:00Z", "Valid": true }
}
```

Las fechas llegan como objeto (`{"Time": ..., "Valid": ...}`) porque las columnas `created_at` y `updated_at` admiten `NULL` y sqlc las mapea a `sql.NullTime`. El frontend lo contempla en `formatDate()`.

### Cuerpo de `PUT /users/{id}`

Se envían `handle`, `display_name` y `email` (`password_hash` se ignora en el PUT). Los tres cambios se aplican dentro de **una única transacción**: si alguno falla (por ejemplo, por un `handle` repetido), no se guarda ninguno.

```json
{
  "handle": "juanp",
  "display_name": "Juan P.",
  "email": "juanp@example.com"
}
```

### Códigos de error

| Código | Cuándo ocurre                                                                                                       |
|--------|---------------------------------------------------------------------------------------------------------------------|
| `400`  | JSON inválido, campos obligatorios vacíos, campos que superan el largo máximo, o `id` que no es un entero positivo  |
| `404`  | No existe el recurso (o no pertenece al usuario de la URL), o una referencia a otro recurso inexistente             |
| `409`  | Violación de unicidad (`handle`/`email` de usuario, nombre de carpeta o tag repetido por usuario, miembro repetido) |
| `500`  | Error interno inesperado de la base de datos (el detalle se registra en el log, no se expone)                       |

## Estructura del proyecto

```
Trabajo-Cursada-tp4/
├── db/
│   ├── queries/
│   │   └── *.sql                 # Queries SQL fuente para sqlc (users, folders, tags, groups, groups_members, tasks, tasks_tags)
│   ├── schema/
│   │   └── schema.sql            # Definición de tablas (usada también por Docker para inicializar la DB)
│   └── sqlc/
│       ├── *_test.go             # Tests de integración contra Postgres (uno por cada .sql)
│       └── helpers_test.go       # Helpers compartidos por los tests
│                                 # (el resto de los archivos de esta carpeta los genera sqlc)
├── static/                       # Frontend (lo sirve main.go)
│   ├── index.html                # Página principal: formulario y lista de usuarios
│   ├── 404.html                  # Página de error 404
│   ├── css/
│   │   └── styles.css            # Estilos compartidos por index.html y 404.html
│   └── js/
│       └── app.js                # Lógica del cliente: fetch, render del DOM y eventos
├── handlers/                     # API REST (paquete handlers)
│   ├── routes.go                 # Register(): registra las rutas de todas las entidades
│   ├── helpers.go                # Lectura de ids, validación y conversiones sql.Null*
│   ├── errors.go                 # Respuestas JSON y traducción de errores de la base a HTTP
│   ├── users.go                  # Handlers de usuarios
│   ├── folders.go                # Handlers de carpetas
│   ├── tags.go                   # Handlers de tags
│   ├── groups.go                 # Handlers de grupos y miembros
│   └── tasks.go                  # Handlers de tareas y de sus tags
├── main.go                       # Arranque: conexión a la DB, handlers.Register y servidor estático
├── request.hurl                  # Pruebas de la API con Hurl
├── sqlc.yaml                     # Configuración de sqlc (motor, rutas, paquete de salida)
├── dockerfile                    # Imagen de la aplicación (build multi-stage)
├── docker-compose.yml            # Levanta Postgres y la aplicación
├── Makefile                      # Automatiza sqlc, build, tests, Hurl y Docker
├── .env.example                  # Plantilla de variables de entorno
└── go.mod / go.sum               # Dependencias del módulo Go
```

> Los archivos `db/sqlc/db.go`, `models.go` y los `*.sql.go` **no se versionan**: son código generado y se crean al ejecutar `make sqlc` (que `make test` ya invoca). Tampoco se versiona el `.env`: el Makefile lo crea automáticamente a partir de `.env.example` si no existe.

## Persistencia

### Motor de base de datos y estrategia de acceso a datos

- **PostgreSQL 16** corriendo en un contenedor Docker (`docker-compose.yml`).
- El acceso a los datos **no usa un ORM**: se escribe SQL plano en `db/queries/`, y **[sqlc](https://sqlc.dev/)** genera código Go type-safe a partir de esas queries y del esquema. Los handlers de la API llaman directamente a esas funciones generadas.
- El driver utilizado para conectarse a Postgres es **[pgx](https://github.com/jackc/pgx)** (vía su interfaz `database/sql`).

### Modelo de datos

El esquema (`db/schema/schema.sql`) define `users`, `folders`, `tags`, `groups`, `groups_members`, `tasks` y `tasks_tags`. A modo de ejemplo, la tabla `users`, que es la que usa el frontend:

| Columna         | Tipo                       | Restricciones                                                         |
|-----------------|----------------------------|-----------------------------------------------------------------------|
| `id`            | `SERIAL`                   | `PRIMARY KEY`                                                         |
| `handle`        | `VARCHAR(31)`              | `UNIQUE`, `NOT NULL`, debe estar en minúsculas y no puede estar vacío |
| `display_name`  | `VARCHAR(63)`              | `NOT NULL`, no puede estar vacío                                      |
| `email`         | `VARCHAR(255)`             | `UNIQUE`, `NOT NULL`, debe estar en minúsculas y no puede estar vacío |
| `password_hash` | `VARCHAR(255)`             | `NOT NULL`, no puede estar vacío                                      |
| `created_at`    | `TIMESTAMP WITH TIME ZONE` | `DEFAULT CURRENT_TIMESTAMP`                                           |
| `updated_at`    | `TIMESTAMP WITH TIME ZONE` | `DEFAULT CURRENT_TIMESTAMP`                                           |

**Decisiones de diseño:**

- `handle` y `email` son `UNIQUE` porque identifican al usuario de forma inequívoca. Se normalizan a minúsculas (a nivel de base con `CHECK` y en las queries con `LOWER()`) para evitar duplicados como `Juan` vs `juan`.
- La API nunca devuelve `password_hash` en sus respuestas.
- La API valida en `handlers/` las mismas reglas que el esquema (no vacío y largo máximo de cada columna), de modo que los datos inválidos se rechazan con `400` antes de llegar a la base.
- Los errores de unicidad de Postgres (código `23505`) se traducen a `409 Conflict`, y las filas inexistentes (`sql.ErrNoRows`) a `404 Not Found`.
- El `PUT` de usuarios usa una transacción para que la actualización de los tres campos sea atómica.
- `updated_at` se actualiza manualmente en cada query de `UPDATE`.

### Inicialización de la base de datos

El esquema (`db/schema/schema.sql`) se monta como script de inicialización de Postgres en `docker-compose.yml`:

```yaml
volumes:
  - ./db/schema/schema.sql:/docker-entrypoint-initdb.d/01-schema.sql:ro
```

Postgres lo ejecuta automáticamente **una sola vez**, la primera vez que se levanta el contenedor con un volumen de datos vacío. `make docker-down` elimina el volumen (`db_data`) para forzar una nueva inicialización.

## Pruebas

El proyecto tiene dos niveles de pruebas automáticas:

1. **Tests de integración en Go** (`db/sqlc/*_test.go`): un archivo por cada `.sql`, que verifica las queries generadas por sqlc contra Postgres real.
2. **Pruebas de la API con [Hurl](https://hurl.dev/)** (`request.hurl`): hacen requests HTTP reales al servidor y verifican códigos de estado y contenido de las respuestas. Empiezan por el CRUD de usuarios (alta, validación, duplicados, listado, consulta, actualización y borrado, con sus casos `400`, `404` y `409`) y siguen con el recorrido completo de carpetas, tags, grupos, miembros, tareas y tags de tareas, incluyendo los casos de permisos.

El frontend **no tiene pruebas automáticas**: se verifica a mano desde el navegador (ver el siguiente apartado).

### Probar el frontend a mano

Con la aplicación levantada (ver [Cómo ejecutarlo](#cómo-ejecutarlo)), abrir `http://localhost:8080` y comprobar:

- La lista carga al abrir la página (o muestra el mensaje de lista vacía).
- Crear un usuario lo agrega a la lista sin recargar la página.
- Crear otro con el mismo `handle` o email muestra el error de duplicado.
- Eliminar un usuario lo quita de la lista.
- Una ruta inexistente (por ejemplo `http://localhost:8080/nada`) muestra la página 404.

## Tecnologías utilizadas

- **HTML, CSS (Flexbox y Grid) y JavaScript** sin frameworks ni librerías, usando la API `fetch` del navegador
- **Go** 1.27 (`net/http` de la biblioteca estándar, sin frameworks)
- **PostgreSQL 16** (vía Docker)
- **[sqlc](https://sqlc.dev/)** para generar código Go a partir de SQL
- **[pgx](https://github.com/jackc/pgx)** como driver de PostgreSQL
- **[Hurl](https://hurl.dev/)** para probar la API REST
- **Docker / Docker Compose** para levantar la base de datos y la aplicación
- **Make** para automatizar el flujo de trabajo

## Requisitos previos

Para clonar y ejecutar el proyecto necesitás tener instalado:

- [Go](https://go.dev/dl/) 1.27 o superior
- [Docker](https://www.docker.com/) y Docker Compose (Docker Desktop los incluye)
- [Hurl](https://hurl.dev/docs/installation.html) (solo para correr las pruebas de la API)
- `make` (viene preinstalado en Linux/macOS; en Windows se puede usar WSL, Git Bash con make, o ejecutar los comandos del Makefile manualmente)

## Cómo ejecutarlo

### 1. Clonar el repositorio y entrar al proyecto

```bash
git clone <URL-del-repositorio>
cd Trabajo-Cursada-tp4
```

### 2. Cambiar de branch (si corresponde)

```bash
git checkout <nombre-de-la-branch>
```

### 3. Levantar la aplicación y usar la interfaz

```bash
make docker-up
```

Esto crea el `.env` a partir de `.env.example` si no existe, construye la imagen y levanta Postgres y el servidor Go, esperando a que estén listos. Después, abrir **http://localhost:8080** en el navegador.

Para apagar todo (eliminando contenedores y volumen, es decir, **los usuarios cargados**):

```bash
make docker-down
```

> Si el volumen de Postgres ya existía de una corrida anterior, el esquema no se vuelve a aplicar. Ante cualquier duda, `make docker-down` y volver a levantar.

### 4. Correr todas las pruebas con un solo comando

```bash
make test
```

Este comando encapsula todo el ciclo:

1. Crea el `.env` a partir de `.env.example` si no existe.
2. Baja cualquier contenedor y volumen previo.
3. Genera el código con **sqlc** y compila el proyecto.
4. Levanta Postgres y la aplicación con Docker, esperando a que estén listos.
5. Corre los tests de integración de Go.
6. Si pasaron, corre `request.hurl` contra la API en `http://localhost:8080`.
7. Baja los contenedores y elimina los volúmenes, **se ejecute lo que se ejecute**, incluso si algún test falla.

El comando termina con un código de salida distinto de cero si falló alguno de los tests.

### Otros targets del Makefile

| Comando            | Descripción                                                                       |
|--------------------|-----------------------------------------------------------------------------------|
| `make sqlc`        | Genera el código Go a partir de las queries SQL                                   |
| `make build`       | Genera el código con sqlc y compila el proyecto                                   |
| `make docker-up`   | Levanta Postgres y la aplicación y espera a que estén listos                      |
| `make hurl`        | Corre solo `request.hurl` (requiere la aplicación levantada con `make docker-up`) |
| `make docker-down` | Elimina contenedores y volúmenes                                                  |
| `make clean`       | Equivale a `make docker-down`                                                     |

Para probar la API manualmente, una vez levantada con `make docker-up`:

```bash
hurl --test request.hurl
```
