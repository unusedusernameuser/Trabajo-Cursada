# my_sqlc_project

Trabajo de cursada de la materia **Programación Web 2026**.

Este repositorio corresponde a la tercera entrega (**TP3**) del proyecto **Gestor de Tareas Personales**: **"Exponiendo la Lógica como API"**. Construye sobre la entrega anterior (TP2), donde se definió el modelo de usuarios en PostgreSQL y se generó la capa de acceso a datos con **sqlc**, y ahora **expone esa lógica como una API REST** con operaciones CRUD sobre la entidad `User`, probada de punta a punta con **Hurl**.

## 📋 Descripción del proyecto

El **Gestor de Tareas Personales** es una aplicación pensada para el día a día del usuario común, que permite crear tareas, agruparlas por categoría, y organizar grupos donde varios usuarios pueden dividirse tareas en común.

### Entidades principales del dominio

- **Usuario (`User`)**: quien crea y administra tareas. Es la entidad implementada hasta el momento, con atributos como `handle` (nombre de usuario), `display_name`, `email` y `password_hash` para autenticación.
- **Tarea (`Task`)**: elemento principal del sistema, con atributos como título, descripción, fecha límite, estado (`pendiente`, `en_progreso`, `completada`), prioridad (`baja`, `media`, `alta`) y la opción de adjuntar un archivo como una imagen.
- **Categoría (`Category`)**: clasificación para las tareas (ej. *NombreMateria*, *Recordatorio*, *Evento*, *Trabajo*).
- **Grupo (`Group`)**: agrupación de tareas que permite colaboración entre varios usuarios (ej. *Vacaciones*, *Proyectos*).

> En esta entrega (TP3) se expone como API REST la entidad `User`. Las entidades `Task`, `Category` y `Group` quedan como trabajo pendiente para próximas entregas (sus tablas y queries están esbozadas pero deshabilitadas).

## Funcionalidad

Actualmente el proyecto implementa:

- **Modelo de datos de usuarios** (`db/schema/schema.sql`): tabla `users` con `handle` (nombre de usuario único), `display_name`, `email` (único) y `password_hash`, con validaciones a nivel de base de datos (formato en minúsculas, campos no vacíos, etc.).
- **Capa de acceso a datos generada con sqlc** (`db/sqlc/`): a partir de las queries escritas en `db/queries/users.sql`, sqlc genera funciones Go type-safe (`GetUserByID`, `ListUsers`, `CreateUser`, `UpdateHandle`, `UpdateEmail`, `UpdateDisplayName`, `DeleteUser`, entre otras).
- **API REST de usuarios** (`handlers.go`): cinco endpoints CRUD que usan directamente las funciones generadas por sqlc, con validación de entrada y traducción de errores de la base de datos a códigos HTTP.
- **Tests de integración** (`db/sqlc/users_test.go`): ejercitan la capa de datos contra una base Postgres real.
- **Pruebas de la API con Hurl** (`request.hurl`): 11 requests que recorren todos los endpoints, incluyendo los casos de error (400, 404, 409).
- **Servidor HTTP** (`main.go`): se conecta a Postgres, registra las rutas de la API y sirve el frontend estático desde `static/` (con página 404 personalizada) en el puerto `8080`.
- **Dockerización**: `docker-compose.yml` levanta Postgres y la aplicación (construida con el `dockerfile`).

## API REST

Base URL: `http://localhost:8080`

| Método   | Ruta          | Descripción                                  | Respuesta exitosa                 |
|----------|---------------|----------------------------------------------|-----------------------------------|
| `POST`   | `/users`      | Crea un usuario                              | `201 Created` + usuario           |
| `GET`    | `/users`      | Lista todos los usuarios                     | `200 OK` + array (`[]` si no hay) |
| `GET`    | `/users/{id}` | Obtiene un usuario por `id`                  | `200 OK` + usuario                |
| `PUT`    | `/users/{id}` | Actualiza `handle`, `email` y `display_name` | `200 OK` + usuario actualizado    |
| `DELETE` | `/users/{id}` | Elimina un usuario por `id`                  | `204 No Content` (sin cuerpo)     |

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
  "created_at": "...",
  "updated_at": "..."
}
```

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

| Código | Cuándo ocurre                                                                                                      |
|--------|--------------------------------------------------------------------------------------------------------------------|
| `400`  | JSON inválido, campos obligatorios vacíos, campos que superan el largo máximo, o `id` que no es un entero positivo |
| `404`  | No existe un usuario con ese `id` (GET y PUT)                                                                      |
| `409`  | Ya existe otro usuario con ese `handle` o `email` (violación de unicidad)                                          |
| `500`  | Error interno inesperado de la base de datos (el detalle se registra en el log, no se expone)                      |

## Estructura del proyecto

```
Trabajo-Cursada-tp3/
├── db/
│   ├── queries/
│   │   ├── users.sql             # Queries SQL fuente para sqlc
│   │   └── future_queries/       # Queries de entidades futuras (aún no activas)
│   ├── schema/
│   │   └── schema.sql            # Definición de tablas (usada también por Docker para inicializar la DB)
│   └── sqlc/
│       └── users_test.go         # Tests de integración contra Postgres
│                                 # (el resto de los archivos de esta carpeta los genera sqlc)
├── static/
│   ├── css/
│   │   ├── 404.css
│   │   └── index.css
│   ├── 404.html
│   └── index.html
├── main.go                       # Arranque: conexión a la DB, rutas y servidor estático
├── handlers.go                   # Handlers CRUD de la API de usuarios y registro de rutas
├── request.hurl                  # Pruebas de la API con Hurl
├── sqlc.yaml                     # Configuración de sqlc (motor, rutas, paquete de salida)
├── dockerfile                    # Imagen de la aplicación (build multi-stage)
├── docker-compose.yml            # Levanta Postgres y la aplicación
├── Makefile                      # Automatiza sqlc, build, tests, Hurl y Docker
├── .env.example                  # Plantilla de variables de entorno
└── go.mod / go.sum               # Dependencias del módulo Go
```

> Los archivos `db/sqlc/db.go`, `models.go` y `users.sql.go` **no se versionan**: son código generado y se crean al ejecutar `make sqlc` (que `make test` ya invoca). Tampoco se versiona el `.env`: el Makefile lo crea automáticamente a partir de `.env.example` si no existe.

## Persistencia

### Motor de base de datos y estrategia de acceso a datos

- **PostgreSQL 16** corriendo en un contenedor Docker (`docker-compose.yml`).
- El acceso a los datos **no usa un ORM**: se escribe SQL plano en `db/queries/users.sql`, y **[sqlc](https://sqlc.dev/)** genera código Go type-safe a partir de esas queries y del esquema. Los handlers de la API llaman directamente a esas funciones generadas.
- El driver utilizado para conectarse a Postgres es **[pgx](https://github.com/jackc/pgx)** (vía su interfaz `database/sql`).

### Modelo de datos

El esquema define por ahora una única entidad, `users`:

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
- Nunca se persiste la contraseña en texto plano: se guarda `password_hash`, y la API nunca lo devuelve en sus respuestas.
- La API valida en `handlers.go` las mismas reglas que el esquema (no vacío y largo máximo de cada columna), de modo que los datos inválidos se rechazan con `400` antes de llegar a la base.
- Los errores de unicidad de Postgres (código `23505`) se traducen a `409 Conflict`, y las filas inexistentes (`sql.ErrNoRows`) a `404 Not Found`.
- El `PUT` usa una transacción para que la actualización de los tres campos sea atómica.
- `updated_at` se actualiza manualmente en cada query de `UPDATE`.

### Inicialización de la base de datos

El esquema (`db/schema/schema.sql`) se monta como script de inicialización de Postgres en `docker-compose.yml`:

```yaml
volumes:
  - ./db/schema/schema.sql:/docker-entrypoint-initdb.d/01-schema.sql:ro
```

Postgres lo ejecuta automáticamente **una sola vez**, la primera vez que se levanta el contenedor con un volumen de datos vacío. `make docker-down` elimina el volumen (`db_data`) para forzar una nueva inicialización.

## Pruebas

El proyecto tiene dos niveles de pruebas:

1. **Tests de integración en Go** (`db/sqlc/users_test.go`): verifican las queries generadas por sqlc contra Postgres real.
2. **Pruebas de la API con [Hurl](https://hurl.dev/)** (`request.hurl`): hacen requests HTTP reales al servidor y verifican códigos de estado y contenido de las respuestas. El archivo cubre, en orden:
   1. Crear usuario (`201`, con normalización a minúsculas y captura del `id`)
   2. Crear con datos inválidos (`400`)
   3. Crear con `handle`/`email` repetido (`409`)
   4. Listar usuarios (`200`)
   5. Obtener usuario por `id` (`200`)
   6. Obtener un `id` inexistente (`404`)
   7. Actualizar usuario (`200`)
   8. Actualizar con datos inválidos (`400`)
   9. Actualizar un `id` inexistente (`404`)
   10. Eliminar usuario (`204`)
   11. Verificar que ya no existe (`404`)

## Tecnologías utilizadas

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
- [Hurl](https://hurl.dev/docs/installation.html)
- `make` (viene preinstalado en Linux/macOS; en Windows se puede usar WSL, Git Bash con make, o ejecutar los comandos del Makefile manualmente)

## Cómo ejecutarlo

### 1. Clonar el repositorio

```bash
git clone <URL-del-repositorio>
```

### 2. Entrar al proyecto

```bash
cd Trabajo-Cursada-tp3
```

### 3. Cambiar de branch

```bash
git checkout <nombre-de-la-branch>
```

### 4. Ejecutar todo con un solo comando

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