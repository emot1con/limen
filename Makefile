.PHONY: help build up down restart logs ps clean migrate-up migrate-down migrate-create db-backup db-restore redis-backup redis-restore swag-up swag-down seed-admin seed-quiz

include .env
export

SWAG_CONTAINER ?= limen-swagger-ui
SWAG_IMAGE ?= docker.swagger.io/swaggerapi/swagger-ui
SWAG_PORT ?= 8090
SWAG_SPEC ?= $(CURDIR)/openapi.yaml
SWAG_CONTAINER_SPEC ?= /usr/share/nginx/html/spec/openapi.yaml
SWAG_SPEC_URL ?= /spec/openapi.yaml
SWAG_SELINUX ?= Z
SWAG_RUN_FLAGS = --rm -d --name $(SWAG_CONTAINER) -p $(SWAG_PORT):8080 -e SWAGGER_JSON_URL=$(SWAG_SPEC_URL) -v "$(SWAG_SPEC):$(SWAG_CONTAINER_SPEC):ro,$(SWAG_SELINUX)" $(SWAG_IMAGE)

POSTGRES_CONTAINER ?= postgres
POSTGRES_USER ?= $(shell grep POSTGRES_USER .env | cut -d '=' -f2)
POSTGRES_DB ?= $(shell grep POSTGRES_DB .env | cut -d '=' -f2)
SEED_POSTGRES_HOST ?= localhost
BACKUP_DIR ?= backups
TIMESTAMP := $(shell date +%Y%m%d_%H%M%S)
MAX_RETRIES ?= 5
RETRY_DELAY ?= 2

help:
	@echo "Available targets:"
	@echo "  build          - Build docker images"
	@echo "  up             - Start containers"
	@echo "  down           - Stop and remove containers"
	@echo "  restart        - Restart containers"
	@echo "  logs           - Show logs (tail -f)"
	@echo "  ps             - Show running containers"
	@echo "  clean          - Stop containers and remove volumes"
	@echo "  migrate-up     - Run all pending migrations up"
	@echo "  migrate-down   - Roll back all applied migrations"
	@echo "  migrate-create - Create new migration (NAME=name)"
	@echo "  db-backup      - Backup PostgreSQL database"
	@echo "  db-restore     - Restore PostgreSQL database (FILE=path)"
	@echo "  redis-backup   - Backup Redis data"
	@echo "  redis-restore  - Restore Redis data (FILE=path)"
	@echo "  swag-up        - Start Swagger UI"
	@echo "  swag-down      - Stop Swagger UI"
	@echo "  seed-admin     - Create or update first admin (ADMIN_EMAIL, ADMIN_FULL_NAME, ADMIN_PASSWORD)"
 
swag-up:
	docker run $(SWAG_RUN_FLAGS)

swag-down:
	docker stop $(SWAG_CONTAINER)

up-build:
	docker compose down
	docker compose up -d --build

app: 
	docker compose down app
	docker compose up -d --build app

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

ps:
	docker compose ps

clean:
	docker compose down -v

migrate-up:
	@set -eu; \
	for file in migrations/*.up.sql; do \
		version=$$(basename "$$file" .up.sql); \
		applied=$$(docker exec -i $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -tAc "SELECT to_regclass('public.schema_migrations') IS NOT NULL"); \
		if [ "$$applied" != "t" ]; then \
			docker exec -i $(POSTGRES_CONTAINER) psql -v ON_ERROR_STOP=1 -U $(POSTGRES_USER) -d $(POSTGRES_DB) -c 'CREATE TABLE public.schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())'; \
		fi; \
		applied=$$(docker exec -i $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -tAc "SELECT EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '$${version}')"); \
		if [ "$$applied" = "f" ]; then \
			echo "Applying $${version}"; \
			{ printf '%s\n' 'BEGIN;'; cat "$$file"; printf '\nINSERT INTO public.schema_migrations (version) VALUES (%s);\nCOMMIT;\n' "'$${version}'"; } | docker exec -i $(POSTGRES_CONTAINER) psql -v ON_ERROR_STOP=1 -U $(POSTGRES_USER) -d $(POSTGRES_DB); \
		else \
			echo "Skipping $${version} (already applied)"; \
		fi; \
	done

migrate-down:
	@set -eu; \
	for file in $$(ls migrations/*.down.sql 2>/dev/null | sort -r); do \
		version=$$(basename "$$file" .down.sql); \
		applied=$$(docker exec -i $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -tAc "SELECT to_regclass('public.schema_migrations') IS NOT NULL"); \
		if [ "$$applied" != "t" ]; then echo "No migrations applied"; exit 0; fi; \
		applied=$$(docker exec -i $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -tAc "SELECT EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '$${version}')"); \
		if [ "$$applied" = "t" ]; then \
			echo "Rolling back $${version}"; \
			{ printf '%s\n' 'BEGIN;'; cat "$$file"; printf '\nDELETE FROM public.schema_migrations WHERE version = %s;\nCOMMIT;\n' "'$${version}'"; } | docker exec -i $(POSTGRES_CONTAINER) psql -v ON_ERROR_STOP=1 -U $(POSTGRES_USER) -d $(POSTGRES_DB); \
		fi; \
	done

migrate-create:
	@test -n "$(NAME)" || (echo "NAME required. Usage: make migrate-create NAME=add_users"; exit 1)
	@NUMBER=$$(printf "%06d" $$(ls migrations/*.up.sql 2>/dev/null | wc -l | xargs expr 1 +)); \
	touch migrations/$${NUMBER}_$(NAME).up.sql migrations/$${NUMBER}_$(NAME).down.sql; \
	echo "Created migrations/$${NUMBER}_$(NAME).{up,down}.sql"

db-backup:
	@mkdir -p $(BACKUP_DIR)
	@echo "Backing up PostgreSQL database..."
	@for i in $$(seq 1 $(MAX_RETRIES)); do \
		if docker compose exec -T postgres pg_dump -U $(POSTGRES_USER) -d $(POSTGRES_DB) --clean --if-exists > $(BACKUP_DIR)/postgres_$(TIMESTAMP).sql; then \
			echo "Backup successful: $(BACKUP_DIR)/postgres_$(TIMESTAMP).sql"; \
			gzip $(BACKUP_DIR)/postgres_$(TIMESTAMP).sql; \
			echo "Compressed: $(BACKUP_DIR)/postgres_$(TIMESTAMP).sql.gz"; \
			exit 0; \
		else \
			echo "Backup attempt $$i failed, retrying in $(RETRY_DELAY)s..."; \
			sleep $(RETRY_DELAY); \
		fi; \
	done; \
	echo "Backup failed after $(MAX_RETRIES) attempts"; \
	exit 1

db-restore:
	@test -n "$(FILE)" || (echo "FILE required. Usage: make db-restore FILE=backups/postgres_20261019_132805.sql.gz"; exit 1)
	@test -f "$(FILE)" || (echo "File $(FILE) not found"; exit 1)
	@echo "Restoring PostgreSQL database from $(FILE)..."
	@if [ "$${FILE##*.}" = "gz" ]; then \
		for i in $$(seq 1 $(MAX_RETRIES)); do \
			if gunzip -c $(FILE) | docker compose exec -T postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB); then \
				echo "Restore successful from $(FILE)"; \
				exit 0; \
			else \
				echo "Restore attempt $$i failed, retrying in $(RETRY_DELAY)s..."; \
				sleep $(RETRY_DELAY); \
			fi; \
		done; \
	else \
		for i in $$(seq 1 $(MAX_RETRIES)); do \
			if docker compose exec -T postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) < $(FILE); then \
				echo "Restore successful from $(FILE)"; \
				exit 0; \
			else \
				echo "Restore attempt $$i failed, retrying in $(RETRY_DELAY)s..."; \
				sleep $(RETRY_DELAY); \
			fi; \
		done; \
	fi; \
	echo "Restore failed after $(MAX_RETRIES) attempts"; \
	exit 1

redis-backup:
	@mkdir -p $(BACKUP_DIR)
	@echo "Backing up Redis data..."
	@for i in $$(seq 1 $(MAX_RETRIES)); do \
		if docker compose exec -T redis redis-cli SAVE > /dev/null 2>&1 && \
		   docker compose exec -T redis cat /data/dump.rdb > $(BACKUP_DIR)/redis_$(TIMESTAMP).rdb; then \
			echo "Backup successful: $(BACKUP_DIR)/redis_$(TIMESTAMP).rdb"; \
			gzip $(BACKUP_DIR)/redis_$(TIMESTAMP).rdb; \
			echo "Compressed: $(BACKUP_DIR)/redis_$(TIMESTAMP).rdb.gz"; \
			exit 0; \
		else \
			echo "Backup attempt $$i failed, retrying in $(RETRY_DELAY)s..."; \
			sleep $(RETRY_DELAY); \
		fi; \
	done; \
	echo "Backup failed after $(MAX_RETRIES) attempts"; \
	exit 1

seed-admin:
	@test -n "$(ADMIN_EMAIL)" || (echo "ADMIN_EMAIL required"; exit 1)
	@test -n "$(ADMIN_FULL_NAME)" || (echo "ADMIN_FULL_NAME required"; exit 1)
	@test -n "$(ADMIN_PASSWORD)" || (echo "ADMIN_PASSWORD required"; exit 1)
	POSTGRES_HOST="$(SEED_POSTGRES_HOST)" ADMIN_EMAIL="$(ADMIN_EMAIL)" ADMIN_FULL_NAME="$(ADMIN_FULL_NAME)" ADMIN_PASSWORD="$(ADMIN_PASSWORD)" go run ./cmd/seed-admin

seed-quiz:
	POSTGRES_HOST="$(SEED_POSTGRES_HOST)" go run ./cmd/seed-quiz

redis-restore:
	@test -n "$(FILE)" || (echo "FILE required. Usage: make redis-restore FILE=backups/redis_20261019_132805.rdb.gz"; exit 1)
	@test -f "$(FILE)" || (echo "File $(FILE) not found"; exit 1)
	@echo "Restoring Redis data from $(FILE)..."
	@if [ "$${FILE##*.}" = "gz" ]; then \
		for i in $$(seq 1 $(MAX_RETRIES)); do \
			if gunzip -c $(FILE) | docker compose exec -T redis sh -c 'cat > /data/dump.rdb' && \
			   docker compose restart redis && \
			   sleep 3 && \
			   docker compose exec -T redis redis-cli PING > /dev/null 2>&1; then \
				echo "Restore successful from $(FILE)"; \
				exit 0; \
			else \
				echo "Restore attempt $$i failed, retrying in $(RETRY_DELAY)s..."; \
				sleep $(RETRY_DELAY); \
			fi; \
		done; \
	else \
		for i in $$(seq 1 $(MAX_RETRIES)); do \
			if cat $(FILE) | docker compose exec -T redis sh -c 'cat > /data/dump.rdb' && \
			   docker compose restart redis && \
			   sleep 3 && \
			   docker compose exec -T redis redis-cli PING > /dev/null 2>&1; then \
				echo "Restore successful from $(FILE)"; \
				exit 0; \
			else \
				echo "Restore attempt $$i failed, retrying in $(RETRY_DELAY)s..."; \
				sleep $(RETRY_DELAY); \
			fi; \
		done; \
	fi; \
	echo "Restore failed after $(MAX_RETRIES) attempts"; \
	exit 1
