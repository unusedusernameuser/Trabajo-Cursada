# my_sqlc_project

Trabajo de cursada de la materia **Programación Web 2026**.

Este repositorio corresponde a la segunda entrega (**TP2**) del proyecto **Gestor de Tareas Personales**. Construye sobre la definición de dominio de la primera entrega (TP1) y avanza en la infraestructura de datos de la aplicación: define el modelo de usuarios en PostgreSQL, genera código Go type-safe a partir de SQL con **sqlc**, y expone un servidor HTTP mínimo que hoy sirve el frontend estático.

## 📋 Descripción del proyecto

El **Gestor de Tareas Personales** es una aplicación pensada para el día a día del usuario común, que permite crear tareas, agruparlas por categoría, y organizar grupos donde varios usuarios pueden dividirse tareas en común.

### Entidades principales del dominio

- **Usuario (`User`)**: quien crea y administra tareas. Es la entidad implementada hasta el momento en este TP, con atributos como `handle` (nombre de usuario), `display_name`, `email` y `password_hash` para autenticación.
- **Tarea (`Task`)**: elemento principal del sistema, con atributos como título, descripción, fecha límite, estado (`pendiente`, `en_progreso`, `completada`), prioridad (`baja`, `media`, `alta`) y la opción de adjuntar un archivo como una imagen.
- **Categoría (`Category`)**: clasificación para las tareas (ej. *NombreMateria*, *Recordatorio*, *Evento*, *Trabajo*).
- **Grupo (`Group`)**: agrupación de tareas que permite colaboración entre varios usuarios (ej. *Vacaciones*, *Proyectos*).

> En esta entrega (TP2) se modeló e implementó la entidad `User` junto con su capa de acceso a datos. Las entidades `Task`, `Category` y `Group`, definidas conceptualmente en el TP1, quedan como trabajo pendiente para próximas entregas.

## Funcionalidad

Actualmente el proyecto implementa:

- **Modelo de datos de usuarios** (`db/schema/schema.sql`): tabla `users` con `handle` (nombre de usuario único), `display_name`, `email` (único) y `password_hash`, con validaciones a nivel de base de datos (formato en minúsculas, campos no vacíos, etc.).
- **Capa de acceso a datos generada con sqlc** (`db/sqlc/`): a partir de las queries escritas en `db/queries/users.sql`, sqlc genera funciones Go type-safe para:
  - `GetUserByID`, `GetUserByHandle`, `GetUserByEmail`
  - `GetUserAuthByEmail` (para autenticación)
  - `ListUsers`
  - `CreateUser`
  - `UpdateDisplayName`, `UpdateHandle`, `UpdateEmail`, `UpdatePassword`
  - `DeleteUser`
- **Tests de integración** (`db/sqlc/users_test.go`) que ejercitan estas operaciones contra una base de datos Postgres real.
- **Servidor HTTP** (`main.go`): por el momento sirve archivos estáticos desde `static/` (incluyendo una página 404 personalizada) en el puerto `8080`. Todavía no expone endpoints REST que utilicen las queries de sqlc.
- **Frontend estático** (`static/`): una página de presentación del proyecto ("Gestor de Tareas").

> Nota: la conexión de las queries de sqlc a endpoints HTTP reales (para exponerlas como una API) es también trabajo pendiente de próximas entregas.

## Estructura del proyecto

```
my_sqlc_project/
├── db/
│   ├── queries/
│   │   └── users.sql        # Queries SQL fuente para sqlc
│   ├── schema/
│   │   └── schema.sql        # Definición de tablas (usada también por Docker para inicializar la DB)
│   └── sqlc/
│       ├── db.go             # Boilerplate generado por sqlc (conexión/queries)
│       ├── models.go         # Structs Go generados a partir de las tablas
│       ├── users.sql.go      # Funciones Go generadas a partir de users.sql
│       └── users_test.go     # Tests de integración contra Postgres
├── static/
│   ├── css/
│   │   ├── 404.css
│   │   └── index.css
│   ├── 404.html
│   └── index.html
├── main.go                   # Servidor HTTP (sirve archivos estáticos)
├── sqlc.yaml                 # Configuración de sqlc (motor, rutas, paquete de salida)
├── docker-compose.yml        # Levanta Postgres para desarrollo/tests
├── Makefile                  # Automatiza generación de código, build, tests y Docker
├── go.mod / go.sum           # Dependencias del módulo Go
└── .env                      # Variables de entorno (credenciales de la DB local)
```

## Persistencia

Esta sección documenta cómo se diseñó y resolvió la capa de persistencia del proyecto.

### Motor de base de datos y estrategia de acceso a datos

- **PostgreSQL 16** corriendo en un contenedor Docker (`docker-compose.yml`), pensado para entornos de desarrollo y testing local.
- El acceso a los datos **no usa un ORM**: se escribe SQL plano en `db/queries/users.sql`, y **[sqlc](https://sqlc.dev/)** genera automáticamente código Go type-safe a partir de esas queries y del esquema (`db/schema/schema.sql`). Esto da funciones Go con structs y parámetros tipados, evitando el uso de `interface{}` o mapeo manual de columnas.
- El driver utilizado para conectarse a Postgres es **[pgx](https://github.com/jackc/pgx)** (vía su interfaz `database/sql`).

### Modelo de datos

Por el momento el esquema define una única entidad, `users`, que sienta las bases para las entidades de tareas, categorías y grupos que se incorporarán en próximas entregas:

| Columna         | Tipo                        | Restricciones                                                        |
|-----------------|------------------------------|-----------------------------------------------------------------------|
| `id`            | `SERIAL`                     | `PRIMARY KEY`                                                         |
| `handle`        | `VARCHAR(31)`                | `UNIQUE`, `NOT NULL`, debe estar en minúsculas y no puede estar vacío |
| `display_name`  | `VARCHAR(63)`                | `NOT NULL`, no puede estar vacío                                      |
| `email`         | `VARCHAR(255)`                | `UNIQUE`, `NOT NULL`, debe estar en minúsculas y no puede estar vacío |
| `password_hash` | `VARCHAR(255)`                | `NOT NULL`, no puede estar vacío                                      |
| `created_at`    | `TIMESTAMP WITH TIME ZONE`   | `DEFAULT CURRENT_TIMESTAMP`                                           |
| `updated_at`    | `TIMESTAMP WITH TIME ZONE`   | `DEFAULT CURRENT_TIMESTAMP`                                           |

**Decisiones de diseño:**

- `handle` y `email` son `UNIQUE` porque identifican al usuario de forma inequívoca (nombre de usuario y correo, respectivamente).
- Ambos se normalizan a minúsculas a nivel de base (`CHECK (... = lower(...))`) para evitar duplicados como `Juan` vs `juan`, y las queries de inserción/actualización aplican `LOWER()` antes de guardarlos.
- Nunca se persiste la contraseña en texto plano: se guarda `password_hash`, y existe una query específica (`GetUserAuthByEmail`) que solo trae `id` y `password_hash`, para no exponer el resto de los datos del usuario en el flujo de autenticación.
- `created_at` y `updated_at` permiten trazar cuándo se creó y modificó por última vez cada registro; `updated_at` se actualiza manualmente en cada query de `UPDATE`.

### Inicialización de la base de datos

El esquema (`db/schema/schema.sql`) se monta como script de inicialización de Postgres en `docker-compose.yml`:

```yaml
volumes:
  - ./db/schema/schema.sql:/docker-entrypoint-initdb.d/01-schema.sql:ro
```

Esto hace que Postgres ejecute `schema.sql` automáticamente **una sola vez**, la primera vez que se levanta el contenedor con un volumen de datos vacío (`make docker-up`). Si se necesita recrear la base desde cero, `make docker-down` elimina el volumen (`db_data`) para forzar una nueva inicialización.

### Operaciones de persistencia disponibles (CRUD)

Definidas en `db/queries/users.sql` y generadas como funciones Go en `db/sqlc/users.sql.go`:

| Query                  | Tipo   | Descripción                                                   |
|-------------------------|--------|-----------------------------------------------------------------|
| `CreateUser`             | Create | Inserta un nuevo usuario (normalizando `handle` y `email`)      |
| `GetUserByID`            | Read   | Busca un usuario por su `id`                                    |
| `GetUserByHandle`        | Read   | Busca un usuario por su `handle`                                 |
| `GetUserByEmail`         | Read   | Busca un usuario por su `email`                                  |
| `GetUserAuthByEmail`     | Read   | Trae solo `id` y `password_hash`, para autenticación             |
| `ListUsers`              | Read   | Lista todos los usuarios, ordenados por `handle`                 |
| `UpdateDisplayName`      | Update | Actualiza el nombre visible de un usuario                        |
| `UpdateHandle`           | Update | Actualiza el `handle` de un usuario                              |
| `UpdateEmail`            | Update | Actualiza el `email` de un usuario                               |
| `UpdatePassword`         | Update | Actualiza el `password_hash` de un usuario                       |
| `DeleteUser`             | Delete | Elimina un usuario por su `id`                                   |

Estas operaciones se validan mediante tests de integración (`db/sqlc/users_test.go`) que corren contra una instancia real de Postgres levantada con Docker (ver sección "Cómo ejecutarlo").

## Tecnologías utilizadas

- **Go** 1.27
- **PostgreSQL 16** (vía Docker)
- **[sqlc](https://sqlc.dev/)** para generar código Go a partir de SQL
- **[pgx](https://github.com/jackc/pgx)** como driver de PostgreSQL
- **Docker / Docker Compose** para levantar la base de datos
- **Make** para automatizar el flujo de trabajo

## Requisitos previos

Para clonar y ejecutar el proyecto necesitás tener instalado:

- [Go](https://go.dev/dl/) 1.27 o superior
- [Docker](https://www.docker.com/) y Docker Compose (Docker Desktop los incluye)
- `make` (viene preinstalado en Linux/macOS; en Windows se puede usar WSL, Git Bash con make, o ejecutar los comandos del Makefile manualmente)

## Cómo ejecutarlo

### 1. Clonar el repositorio

```bash
git clone <URL-del-repositorio>
```

### 2. Entrar al proyecto

```bash
cd Trabajo-Cursada-tp2/my_sqlc_project
```

### 3. Cambiar de branch

```bash
git checkout <nombre-de-la-branch>
```

### 4. Ejecutar los tests

```bash
make test
```

Este comando encapsula todo el ciclo: baja cualquier contenedor previo, compila el proyecto, levanta la base de datos con Docker, corre los tests de integración contra Postgres, y finalmente baja el contenedor (se ejecute lo que se ejecute, incluso si los tests fallan).
