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

.PHONY: all test hurl sqlc build docker-down docker-up clean

# Punto de entrada por defecto: corre todo el ciclo completo.
all: test

## ---- 1) Tareas previas ------------------------------------------------

SQLC_VERSION := v1.31.1

sqlc:
	@echo "==> Generando código con sqlc..."
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

build: sqlc
	@echo "==> Compilando el proyecto..."
	go build ./...

docker-down:
	@echo "==> Eliminando contenedores y volúmenes previos..."
	$(COMPOSE) down -v --remove-orphans

# --wait hace que Compose no devuelva el control hasta que el servicio
# reporte "healthy" según el healthcheck definido en docker-compose.yml
# (pg_isready). Reemplaza a un script de espera separado.
docker-up:
	@echo "==> Levantando contenedores y esperando a que estén listos..."
	$(COMPOSE) up -d --wait --wait-timeout $(WAIT_TIMEOUT)

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