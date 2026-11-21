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
2. El handler llama a `DeleteUser` y responde `200`. Lo único que envía es el contador del título (`hx-swap-oob`) para que el número "Usuarios (N)" no quede desactualizado; la fila se reemplaza por nada.

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
- [Hurl](https://hurl.dev/docs/installation.html) (solo para las pruebas HTTP)
- `make`

templ y sqlc no hay que instalarlos: el Makefile los ejecuta con `go tool` / `go run`.

## Cómo ejecutarlo

```bash
git clone <URL-del-repositorio>
cd Trabajo-Cursada-tp6

make deps   # una sola vez: agrega templ a go.mod / go.sum
make run    # genera código (sqlc + templ), levanta Postgres y corre el servidor
```

Abrir **http://localhost:8080**. Con `Ctrl+C` se detiene el servidor; `make docker-down` apaga Postgres y borra el volumen (los usuarios cargados).

Para levantar todo en contenedores (Postgres + servidor): `make docker-up`.

### Targets del Makefile

| Comando            | Descripción                                                                         |
|--------------------|-------------------------------------------------------------------------------------|
| `make deps`        | Registra templ como dependencia del módulo (una sola vez)                           |
| `make templ`       | `templ generate`: compila `views/*.templ` a `views/*_templ.go`                      |
| `make sqlc`        | Genera el código Go a partir de las queries SQL                                     |
| `make generate`    | `sqlc` + `templ`                                                                    |
| `make build`       | Genera el código y compila el proyecto                                              |
| `make db-up`       | Levanta solo Postgres (Docker) y espera a que esté listo                            |
| `make run`         | `generate` + `db-up` + `go run .` (servidor local en el puerto 8080)                |
| `make docker-up`   | Genera el código y levanta Postgres y la aplicación en contenedores                 |
| `make test`        | Ciclo completo: DB limpia, build, contenedores, `go test`, `hurl` y limpieza        |
| `make hurl`        | Solo `request.hurl` (requiere la aplicación levantada)                              |
| `make docker-down` | Elimina contenedores y volúmenes                                                    |

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
- **Acciones sin JavaScript**: lo que antes eran `PUT`/`DELETE` con `fetch` ahora son formularios `POST` (una ruta por acción). En la página de tareas los únicos fragmentos de JS son los `confirm()` de los botones de borrado; en usuarios la confirmación la hace `hx-confirm`.
- **Filtros en la URL**: así la vista es enlazable y no hace falta estado en el cliente. Las tareas se leen una vez con `ListTasksByUser` y se filtran en Go; los tags de cada tarea se arman con una consulta `ListTasksByTag` por tag.
- **Edición en otra página**: el diálogo modal pasó a `GET /users/{user_id}/tasks/{id}/edit`, que conserva los filtros para volver a la misma lista.
- **Zona horaria**: los `<input type="datetime-local">` no llevan zona, así que el servidor interpreta y muestra las fechas en `America/Argentina/Buenos_Aires` (configurable con la variable de entorno `APP_TIMEZONE`). La base de zonas horarias va incluida en el binario (`time/tzdata`).
