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
