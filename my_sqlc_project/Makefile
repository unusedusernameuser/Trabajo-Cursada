include .env
export

COMPOSE      := docker compose
WAIT_TIMEOUT := 60

.PHONY: all test sqlc build docker-down docker-up clean

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

test: docker-down build docker-up
	@echo "==> Ejecutando tests..."
	@DATABASE_URL="$(DATABASE_URL)" go test ./... -v; \
	status=$$?; \
	$(MAKE) docker-down; \
	exit $$status

## ---- 3) Tareas posteriores ---------------------------------------------
## (docker-down ya se usa arriba como paso final, sea cual sea el resultado
##  de los tests, y también queda disponible como target independiente)

clean: docker-down
