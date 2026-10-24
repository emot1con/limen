.PHONY: help build up down restart logs ps clean migrate-up migrate-down migrate-create db-backup db-restore redis-backup redis-restore swag-up swag-down

SWAG_CONTAINER ?= limen-swagger-ui
SWAG_IMAGE ?= docker.swagger.io/swaggerapi/swagger-ui
SWAG_PORT ?= 8090
SWAG_SPEC ?= $(CURDIR)/openapi.yaml
SWAG_CONTAINER_SPEC ?= /usr/share/nginx/html/spec/openapi.yaml
SWAG_SPEC_URL ?= /spec/openapi.yaml
SWAG_SELINUX ?= Z
SWAG_RUN_FLAGS = --rm -d --name $(SWAG_CONTAINER) -p $(SWAG_PORT):8080 -e SWAGGER_JSON_URL=$(SWAG_SPEC_URL) -v "$(SWAG_SPEC):$(SWAG_CONTAINER_SPEC):ro,$(SWAG_SELINUX)" $(SWAG_IMAGE)

POSTGRES_USER ?= $(shell grep POSTGRES_USER .env | cut -d '=' -f2)
POSTGRES_DB ?= $(shell grep POSTGRES_DB .env | cut -d '=' -f2)
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
	@echo "  migrate-up     - Run migrations up"
	@echo "  migrate-down   - Run migrations down"
	@echo "  migrate-create - Create new migration (NAME=name)"
	@echo "  db-backup      - Backup PostgreSQL database"
	@echo "  db-restore     - Restore PostgreSQL database (FILE=path)"
	@echo "  redis-backup   - Backup Redis data"
	@echo "  redis-restore  - Restore Redis data (FILE=path)"
	@echo "  swag-up        - Start Swagger UI"
	@echo "  swag-down      - Stop Swagger UI"
 
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
	docker compose exec -it postgres psql -U postgres $(shell grep POSTGRES_USER .env | cut -d '=' -f2) -d $(shell grep POSTGRES_DB .env | cut -d '=' -f2) < migrations/000001_init.up.sql

migrate-down:
	docker compose exec -it postgres psql -U postgres $(shell grep POSTGRES_USER .env | cut -d '=' -f2) -d $(shell grep POSTGRES_DB .env | cut -d '=' -f2) < migrations/000001_init.down.sql

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
