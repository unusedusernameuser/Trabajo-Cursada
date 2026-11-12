# Gestor de Tareas

Trabajo de cursada de la materia **Programación Web 2026**.

Esta entrega reemplaza el frontend separado (HTML/CSS/JS + API REST en JSON) por una **aplicación renderizada en el servidor**: el servidor Go genera el HTML con **[templ](https://templ.guide/)**, lee y escribe en PostgreSQL con **sqlc**, y el navegador solo muestra páginas completas. No hay JavaScript propio ni endpoints JSON.

## Funcionalidad

- **Página principal** (`GET /`): formulario para crear usuarios y tabla con todos los usuarios.
- **Alta** (`POST /users`): parsea el formulario, valida, guarda con `CreateUser` y redirige a `/`.
- **Baja** (`POST /users/{id}/delete`): elimina con `DeleteUser` y redirige a `/` (los formularios HTML solo soportan GET y POST).
- **Errores**: si el formulario es inválido (`400`) o el handle/email ya existe (`409`), se vuelve a mostrar la página con el mensaje y lo que se había escrito (salvo la contraseña).
- **404**: cualquier otra ruta muestra una página 404 hecha con `templ`.
- **Estilos**: [Pico.css](https://picocss.com/) desde CDN, enlazado en el layout.

### Patrón Post/Redirect/Get (PRG)

1. El navegador envía el formulario con `POST /users`.
2. El handler guarda el usuario y responde `303 See Other` con `Location: /`.
3. El navegador pide `GET /` y el servidor muestra la lista ya actualizada.

Así, recargar la página después de crear un usuario no reenvía el formulario.

## Componentes templ (`views/`)

| Archivo            | Componente                       | Rol                                                                 |
|--------------------|----------------------------------|---------------------------------------------------------------------|
| `layout.templ`     | `Layout(title)`                  | Estructura base (`html`, `head`, `body`) y link a Pico.css          |
| `user_list.templ`  | `UserList([]db.ListUsersRow)`    | Tabla con los usuarios y un botón Eliminar por fila                 |
| `user_form.templ`  | `UserForm(UserFormValues)`       | Formulario de alta (`POST /users`), con mensaje de error opcional   |
| `index.templ`      | `IndexPage(users, form)`         | Página principal: usa el layout y compone formulario y lista        |
| `not_found.templ`  | `NotFoundPage()`                 | Página 404                                                          |
| `data.go`          | —                                | Tipo `UserFormValues` y helpers de formato de fecha                 |

templ escapa automáticamente los textos, así que lo que escribe cada persona nunca se interpreta como HTML.

## Estructura del proyecto

```
Trabajo-Cursada-tp4/
├── db/
│   ├── queries/*.sql             # Queries fuente para sqlc
│   ├── schema/schema.sql         # Tablas (también inicializa Docker)
│   └── sqlc/*_test.go            # Tests de integración contra Postgres
├── views/                        # Componentes templ
│   ├── layout.templ, user_list.templ, user_form.templ, index.templ, not_found.templ
│   └── data.go
├── handlers/
│   ├── routes.go                 # Register(): rutas y 404
│   └── users.go                  # GET /, POST /users, POST /users/{id}/delete
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
cd Trabajo-Cursada-tp4

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
2. **Pruebas HTTP con Hurl** (`request.hurl`): verifican la página principal, el alta con redirect `303`, los errores `400` y `409`, el escape de HTML, el borrado y la página 404.

`make test` corre ambas.

Prueba manual: abrir `http://localhost:8080`, crear un usuario (la página se recarga completa y aparece en la lista), repetir el handle (aparece el error), eliminar un usuario y visitar `/nada` (página 404).

## Tecnologías

Go 1.27 (`net/http`), **templ**, **Pico.css**, PostgreSQL 16, **sqlc**, **pgx**, Hurl, Docker Compose y Make.

## Decisiones de diseño

- **Contraseña**: se guarda tal cual en `password_hash`, como en la entrega anterior; en una aplicación real se hashearía en el servidor (por ejemplo con bcrypt).
- **Borrado con POST**: un `<form>` no puede enviar `DELETE`, por eso se usa `POST /users/{id}/delete`.
- **Sin API JSON**: se eliminó la API REST de la entrega anterior (usuarios, carpetas, tags, grupos y tareas). Las queries de sqlc y sus tests se mantienen; la API anterior sigue en el historial de git.
