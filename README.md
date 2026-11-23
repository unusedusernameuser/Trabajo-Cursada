# Gestor de Tareas

Trabajo de cursada de la materia **Programación Web 2026**.

Esta entrega reemplaza el frontend separado (HTML/CSS/JS + API REST en JSON) por una **aplicación renderizada en el servidor**: el servidor Go genera el HTML con **[templ](https://templ.guide/)**, lee y escribe en PostgreSQL con **sqlc**, y el navegador muestra páginas completas. Desde el TP6 el alta y la baja de usuarios usan **HTMX** para actualizar la lista sin recargar la página. No hay JavaScript propio ni endpoints JSON.

## Funcionalidad

- **Página principal** (`GET /`): formulario para crear usuarios y tabla con todos los usuarios. Cada usuario enlaza a su página de tareas.
- **Alta** (`POST /users`, con HTMX): parsea el formulario, valida, guarda con `CreateUser`, vuelve a consultar la lista con `ListUsers` y devuelve **solo el componente `UserList`**. HTMX lo coloca en `#user-list`, sin recargar la página. No hay redirección.
- **Baja** (`DELETE /users/{id}`, con HTMX): elimina con `DeleteUser` y responde `200` sin contenido para la fila; HTMX reemplaza la fila (`closest tr`) por nada. Pide confirmación con `hx-confirm`.
- **Página de tareas de un usuario** (`GET /users/{user_id}/tasks`): reemplaza a `tasks.html` + `tasks.js` de la entrega anterior. Cada usuario ve y modifica **solo** lo suyo: todas las queries filtran por `user_id`.
  - **Carpetas**: crear, renombrar y borrar. Al borrar una carpeta, sus tareas quedan sin carpeta.
  - **Tags**: crear, renombrar y borrar. Al borrar un tag se quita de todas las tareas.
  - **Tareas**: crear (título, descripción, fecha límite, prioridad, carpeta y tags), cambiar el estado (con el botón rápido *Completar/Reabrir* o con el selector), mover de carpeta, agregar y quitar tags, editar (página `.../edit`) y eliminar. Las vencidas se marcan y las completadas van al final.
  - **Filtros**: por carpeta (incluida *Sin carpeta*), por tag, por estado y por prioridad. Ya no se filtran en el navegador: viajan en la URL (`?folder=3&tag=2&status=pendiente&priority=alta`) y el servidor devuelve la lista filtrada, así que cada combinación se puede enlazar y recargar.
- **Errores**: si el formulario es inválido (`400`) o el nombre/handle/email ya existe (`409`), se vuelve a mostrar la página con el mensaje (y, en el alta de usuarios y de tareas, con lo que se había escrito).
- **404**: cualquier otra ruta (o un usuario o una tarea que no existe) muestra una página 404 hecha con `templ`.
- **Estilos**: [Pico.css](https://picocss.com/) desde CDN, enlazado en el layout, más unas pocas reglas propias en el mismo layout.

### Rutas

| Método | Ruta                                              | Acción                                                         |
|--------|---------------------------------------------------|----------------------------------------------------------------|
| GET    | `/users/{user_id}/tasks`                          | Página de tareas (acepta `folder`, `tag`, `status`, `priority`) |
| POST   | `/users/{user_id}/tasks`                          | Crear tarea (`title`, `task_description`, `due_date`, `task_priority`, `folder_id`, `tag_ids`) |
| GET    | `/users/{user_id}/tasks/{id}/edit`                | Formulario de edición                                          |
| POST   | `/users/{user_id}/tasks/{id}`                     | Guardar la edición (título, descripción, fecha y prioridad)    |
| POST   | `/users/{user_id}/tasks/{id}/delete`              | Eliminar tarea                                                 |
| POST   | `/users/{user_id}/tasks/{id}/status`              | Cambiar estado (`task_status`)                                 |
| POST   | `/users/{user_id}/tasks/{id}/folder`              | Mover de carpeta (`folder_id`, vacío = sin carpeta)            |
| POST   | `/users/{user_id}/tasks/{id}/tags`                | Agregar un tag a la tarea (`tag_id`)                           |
| POST   | `/users/{user_id}/tasks/{id}/tags/{tag_id}/delete`| Quitar un tag de la tarea                                      |
| POST   | `/users/{user_id}/folders`                        | Crear carpeta (`folder_name`)                                  |
| POST   | `/users/{user_id}/folders/{id}`                   | Renombrar carpeta                                              |
| POST   | `/users/{user_id}/folders/{id}/delete`            | Eliminar carpeta                                               |
| POST   | `/users/{user_id}/tags`                           | Crear tag (`tag_name`)                                         |
| POST   | `/users/{user_id}/tags/{id}`                      | Renombrar tag                                                  |
| POST   | `/users/{user_id}/tags/{id}/delete`               | Eliminar tag                                                   |

Todas las acciones que modifican datos son `POST` y terminan en un `303` a la lista de tareas. Los formularios mandan un campo oculto `back` con los filtros que tenía la página, para volver a la misma vista (el servidor solo reutiliza los filtros que conoce).

### Usuarios con HTMX (sin recargar la página)

El script de HTMX se carga en el `<head>` de `layout.templ`. Las dos acciones de usuarios son parciales de HTML:

**Alta**
1. `UserForm` tiene `hx-post="/users" hx-target="#user-list" hx-swap="outerHTML"`.
2. El handler guarda el usuario y devuelve `UserList` con los datos actualizados (sin `<html>`, sin redirect).
3. HTMX reemplaza el `<article id="user-list">` por el fragmento recibido.
4. Junto con la lista viaja un formulario vacío marcado con `hx-swap-oob`, que HTMX intercambia fuera de banda para limpiar los campos (sin JavaScript).

**Errores (400 / 409)**: HTMX no intercambia por defecto las respuestas 4xx. El layout lo habilita para 400 y 409 con `<meta name="htmx-config">` (`responseHandling`), y el handler agrega los headers `HX-Retarget: #user-form` y `HX-Reswap: outerHTML` para que el formulario con el mensaje de error reemplace al formulario, no a la lista.

**Baja**
1. Cada fila tiene un botón con `hx-delete="/users/{id}" hx-target="closest tr" hx-swap="outerHTML" hx-confirm="..."`.
2. El handler llama a `DeleteUser` y responde `200` con el cuerpo vacío; HTMX reemplaza la fila por nada.

La página principal (`GET /`) sigue siendo una página completa. La página de tareas de cada usuario no cambió (formularios `POST` + redirect 303).

## Componentes templ (`views/`)

| Archivo            | Componente                       | Rol                                                                 |
|--------------------|----------------------------------|---------------------------------------------------------------------|
| `layout.templ`     | `Layout(title)`                  | Estructura base (`html`, `head`, `body`), Pico.css y el script de HTMX |
| `user_list.templ`  | `UserList([]db.ListUsersRow)`    | Tabla con los usuarios (`id="user-list"`); Eliminar con `hx-delete`    |
| `user_form.templ`  | `UserForm(UserFormValues)`       | Formulario de alta (`hx-post`), con mensaje de error opcional      |
| `index.templ`      | `IndexPage(users, form)`         | Página principal: usa el layout y compone formulario y lista        |
| `not_found.templ`  | `NotFoundPage()`                 | Página 404                                                          |
| `tasks_page.templ` | `TasksPage(TasksPageData)`       | Página de tareas: compone `FolderPanel`, `TagPanel`, `TaskForm` y `TaskList` |
| `tasks_page.templ` | `TaskItem`, `PriorityOptions`, `FolderOptions` | Una tarea con sus acciones y las `<option>` de los `<select>` |
| `task_edit.templ`  | `TaskEditPage(TaskEditData)`     | Formulario de edición de una tarea (reemplaza al diálogo modal)     |
| `data.go`          | —                                | Modelos de las páginas, filtros (`TaskFilters`), zona horaria y helpers |

templ escapa automáticamente los textos, así que lo que escribe cada persona nunca se interpreta como HTML.

## Estructura del proyecto

```
Trabajo-Cursada-tp6/
├── db/
│   ├── queries/*.sql             # Queries fuente para sqlc
│   ├── schema/schema.sql         # Tablas (también inicializa Docker)
│   └── sqlc/*_test.go            # Tests de integración contra Postgres
├── views/                        # Componentes templ
│   ├── layout.templ, user_list.templ, user_form.templ, index.templ, not_found.templ
│   ├── tasks_page.templ, task_edit.templ
│   ├── data.go, filters_test.go
├── handlers/
│   ├── routes.go                 # Register(): rutas y 404
│   ├── users.go                  # GET /, POST /users (fragmento), DELETE /users/{id}
│   ├── tasks.go                  # Página de tareas, alta/edición/baja, estado, carpeta y tags de la tarea
│   ├── folders.go, tags.go       # Alta, renombrado y baja de carpetas y de tags
│   └── helpers.go                # Lectura de ids, validaciones, fechas y redirect (PRG)
├── main.go                       # Conexión a la DB y servidor HTTP
├── request.hurl                  # Pruebas de las rutas con Hurl
├── Makefile, dockerfile, docker-compose.yml, sqlc.yaml, .env.example
└── go.mod / go.sum
```

> No se versionan los archivos generados: `db/sqlc/*.sql.go`, `db/sqlc/db.go`, `db/sqlc/models.go` (sqlc) ni `views/*_templ.go` (templ). Se crean con `make generate`. Tampoco se versiona `.env`: el Makefile lo crea desde `.env.example`.

## Requisitos previos

- [Go](https://go.dev/dl/) 1.27 o superior
- [Docker](https://www.docker.com/) y Docker Compose
- [Hurl](https://hurl.dev/docs/installation.html) (solo para `make test`; no hace falta para `make run-app`)
- `make`
- Puertos libres: `8080` (aplicación) y `5432` (Postgres; configurable con `DB_PORT` en `.env`)

templ y sqlc no hay que instalarlos: el Makefile los ejecuta con `go tool` / `go run`.

## Cómo ejecutarlo

```bash
git clone <URL-del-repositorio>
cd Trabajo-Cursada-tp6

make run-app
```

Un solo comando. `make run-app` hace, en orden:

1. **Tareas previas**: borra contenedores y volúmenes anteriores, genera el código (`sqlc` y `templ`), compila el proyecto, levanta Postgres con Docker Compose y espera a que esté saludable (`pg_isready`).
2. **Ejecución**: corre el servidor en **http://localhost:8080**.
3. **Tareas posteriores**: al terminar (con `Ctrl+C`, o si algo falla) borra los contenedores y los volúmenes, de modo que la base queda vacía en la próxima ejecución.

Si no existe `.env`, el Makefile lo crea a partir de `.env.example`. No hace falta instalar nada más que los requisitos de abajo. La aplicación carga Pico.css y HTMX desde CDN, así que se necesita conexión a internet para ver los estilos y las acciones de HTMX.

> Tanto el código generado como el `.env` se crean solos y no se versionan. No hay archivos `.js` en el repositorio.

Para levantar todo en contenedores (Postgres + servidor), sin limpieza automática: `make docker-up` (y `make docker-down` para apagar).

### Targets del Makefile

| Comando            | Descripción                                                                         |
|--------------------|-------------------------------------------------------------------------------------|
| `make run-app`     | **Comando principal**: limpia, genera, compila, levanta Postgres, corre la app y limpia al terminar |
| `make deps`        | Opcional: actualiza templ a la última versión (`go.mod` ya lo declara como `tool`)  |
| `make templ`       | `templ generate`: compila `views/*.templ` a `views/*_templ.go`                      |
| `make sqlc`        | Genera el código Go a partir de las queries SQL                                     |
| `make generate`    | `sqlc` + `templ`                                                                    |
| `make build`       | Genera el código y compila el proyecto                                              |
| `make db-up`       | Levanta solo Postgres (Docker) y espera a que esté listo                            |
| `make run`         | Desarrollo: `generate` + `db-up` + `go run .`; no borra nada al terminar            |
| `make docker-up`   | Genera el código y levanta Postgres y la aplicación en contenedores                 |
| `make test`        | Ciclo completo: DB limpia, build, contenedores, `go test`, `hurl` y limpieza        |
| `make hurl`        | Solo `request.hurl` (requiere la aplicación levantada)                              |
| `make docker-down` | Elimina contenedores y volúmenes                                                    |
| `make clean`       | Alias de `docker-down`                                                              |

## Modelo de datos

Definido en `db/schema/schema.sql` (Docker lo aplica al crear el volumen) y consumido por sqlc.

```
users 1───* folders          (folders.user_id, ON DELETE CASCADE)
users 1───* tags             (tags.user_id,    ON DELETE CASCADE)
users 1───* tasks            (tasks.user_id,   ON DELETE CASCADE)
users 1───* tasks            (tasks.assigned_to, ON DELETE SET NULL)
folders 1───* tasks          (tasks.folder_id, ON DELETE SET NULL)
tasks *───* tags             (tabla tasks_tags, ON DELETE CASCADE en ambos lados)
```

| Tabla        | Campos principales                                                                                  | Restricciones                                                              |
|--------------|-----------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------|
| `users`      | `id`, `handle`, `display_name`, `email`, `password_hash`, `created_at`, `updated_at`                | `handle` y `email` únicos, en minúsculas y no vacíos                       |
| `folders`    | `id`, `user_id`, `folder_name`                                                                      | `(user_id, folder_name)` único                                             |
| `tags`       | `id`, `user_id`, `tag_name`                                                                         | `(user_id, tag_name)` único                                                |
| `tasks`      | `id`, `user_id`, `folder_id`, `assigned_to`, `title`, `task_description`, `due_date`, `task_status`, `task_priority`, `attachment_path`, `completed_at` | estado ∈ `pendiente`/`en_progreso`/`completada`; prioridad ∈ `baja`/`media`/`alta`; `completed_at` solo si está completada |
| `tasks_tags` | `task_id`, `tag_id`                                                                                 | clave primaria compuesta                                                   |

Consecuencias: al borrar un usuario se borran sus carpetas, tags y tareas; al borrar una carpeta sus tareas quedan sin carpeta; al borrar un tag se quita de las tareas. `assigned_to` y `attachment_path` existen en el esquema pero la interfaz actual no los usa.

## Pruebas

1. **Tests de integración de las queries** (`db/sqlc/*_test.go`): contra Postgres real.
2. **Test unitario de los filtros** (`views/filters_test.go`): lectura de la query string y URLs de los filtros.
3. **Pruebas HTTP con Hurl** (`request.hurl`): verifican la página principal, el alta y la baja con HTMX (fragmentos de HTML, sin redirect), los errores `400` y `409`, el escape de HTML, el borrado y la página 404; y la página de tareas: carpetas, tags y tareas (alta, estado, mover, tags, edición y baja), filtros por la URL y errores.

`make test` corre todas.

Prueba manual: abrir `http://localhost:8080`, crear un usuario (aparece en la lista y el formulario se limpia, sin recargar la página), repetir el handle (aparece el error), entrar a sus tareas, crear carpetas, tags y tareas, probar los filtros, eliminar un usuario y visitar `/nada` (página 404).

## Tecnologías

Go 1.27 (`net/http`), **templ**, **Pico.css**, PostgreSQL 16, **sqlc**, **pgx**, Hurl, Docker Compose y Make.

## Decisiones de diseño

- **Contraseña**: se guarda tal cual en `password_hash`, como en la entrega anterior; en una aplicación real se hashearía en el servidor (por ejemplo con bcrypt).
- **Borrado con `DELETE`**: HTMX permite enviar `DELETE` desde un botón (`hx-delete`), por eso la baja de usuarios usa `DELETE /users/{id}`. En la página de tareas se mantienen los `POST`.
- **Sin API JSON**: se eliminó la API REST de la entrega anterior (usuarios, carpetas, tags y tareas) y el cliente JS (`static/`). Las queries de sqlc y sus tests se mantienen; la API anterior sigue en el historial de git.
- **Acciones sin JavaScript**: lo que antes eran `PUT`/`DELETE` con `fetch` ahora son formularios `POST` (una ruta por acción). No hay JavaScript propio, ni archivos `.js` ni código inline: las confirmaciones de borrado usan `hx-confirm` (en la página de tareas, los formularios de borrado llevan además `hx-boost="true"` para que HTMX gestione el envío y pida la confirmación).
- **Filtros en la URL**: así la vista es enlazable y no hace falta estado en el cliente. Las tareas se leen una vez con `ListTasksByUser` y se filtran en Go; los tags de cada tarea se arman con una consulta `ListTasksByTag` por tag.
- **Edición en otra página**: el diálogo modal pasó a `GET /users/{user_id}/tasks/{id}/edit`, que conserva los filtros para volver a la misma lista.
- **Zona horaria**: los `<input type="datetime-local">` no llevan zona, así que el servidor interpreta y muestra las fechas en `America/Argentina/Buenos_Aires` (configurable con la variable de entorno `APP_TIMEZONE`). La base de zonas horarias va incluida en el binario (`time/tzdata`).
