# Si no existe un .env, make lo crea copiando .env.example
.env:
	@test -f .env.example || { echo "Falta .env.example: no se puede crear .env"; exit 1; }
	@cp .env.example .env
	@echo "==> .env no existe: se creo a partir de .env.exaple"

include .env
export

COMPOSE      := docker compose
WAIT_TIMEOUT := 60
HURL_FILE    := request.hurl
# Reintentos por si la app todavia esta arrancando cuando empieza hurl
HURL_RETRY   := 15

.PHONY: all test hurl deps templ sqlc generate build run db-up docker-down docker-up clean

# Punto de entrada por defecto: corre todo el ciclo completo.
all: test

## ---- 1) Tareas previas ------------------------------------------------

SQLC_VERSION := v1.31.1

# Una sola vez (o al clonar el repo): registra templ como dependencia y herramienta
# del módulo (go.mod / go.sum). Después, `go tool templ` usa esa misma versión.
deps:
	@echo "==> Instalando templ como dependencia del módulo..."
	go get github.com/a-h/templ@latest
	go get -tool github.com/a-h/templ/cmd/templ@latest
	go mod tidy

sqlc:
	@echo "==> Generando código con sqlc..."
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

# Compila los .templ de views/ a código Go (views/*_templ.go).
templ:
	@echo "==> Generando componentes con templ..."
	go tool templ generate

generate: sqlc templ

build: generate
	@echo "==> Compilando el proyecto..."
	go build ./...

# Levanta solo el motor de base de datos (Postgres) para correr el servidor en la máquina local.
db-up:
	@echo "==> Levantando Postgres y esperando a que esté listo..."
	$(COMPOSE) up -d --wait --wait-timeout $(WAIT_TIMEOUT) db

# Flujo de desarrollo: genera código (sqlc + templ), levanta Postgres y corre el servidor
# local en http://localhost:8080. Con Ctrl+C se detiene el servidor (la DB queda arriba;
# apagarla con `make docker-down`).
run: generate db-up
	@echo "==> Iniciando el servidor en http://localhost:8080 ..."
	DATABASE_URL="$(DATABASE_URL)" go run .

docker-down:
	@echo "==> Eliminando contenedores y volúmenes previos..."
	$(COMPOSE) down -v --remove-orphans

# --wait hace que Compose no devuelva el control hasta que el servicio
# reporte "healthy" según el healthcheck definido en docker-compose.yml
# (pg_isready). Reemplaza a un script de espera separado.
docker-up: generate
	@echo "==> Levantando contenedores y esperando a que estén listos..."
	$(COMPOSE) up -d --build --wait --wait-timeout $(WAIT_TIMEOUT)

## ---- 2) Ejecución de tests --------------------------------------------

# Corre los tests de Go y, si pasan, las pruebas de la API con hurl.
test: docker-down build docker-up
	@echo "==> Ejecutando tests..."
	@DATABASE_URL="$(DATABASE_URL)" go test ./... -v; \
	status=$$?; \
	if [ $$status -eq 0 ]; then \
		$(MAKE) hurl; \
		status=$$?; \
	fi; \
	$(MAKE) docker-down; \
	exit $$status

# Ejecuta request.hurl contra el servidor en http://localhost:8080.
# Requiere la app levantada (make docker-up) y hurl instalado.
hurl:
	@command -v hurl >/dev/null 2>&1 || { echo "Falta hurl: https://hurl.dev/docs/installation.html"; exit 1; }
	@echo "==> Ejecutando $(HURL_FILE) con hurl..."
	hurl --test --retry $(HURL_RETRY) --retry-interval 1000 $(HURL_FILE)

## ---- 3) Tareas posteriores ---------------------------------------------
## (docker-down ya se usa arriba como paso final, sea cual sea el resultado
##  de los tests, y también queda disponible como target independiente)

clean: docker-down