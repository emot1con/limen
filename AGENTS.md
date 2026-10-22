# AGENTS.md

## Scope

Limen backend: Go 1.25.10, Gin, GORM, PostgreSQL 15, Redis 7. Clean architecture.

## Conflict handling

When user instructions, specifications, documentation, or implementation conflict, do not guess which one is authoritative. Point out the conflicting details and ask the user to confirm before proceeding.

## Testing

Unit and integration testing are deferred until system implementation is complete. Do not treat missing tests as a blocker before then.

## Local development

Start application:

```sh
set -a; . ./.env; set +a
go run ./cmd/api
```

Build binary (production):

```sh
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/api ./cmd/api
```

Health check runs at `http://localhost:8000/health` (or `APP_PORT` from `.env`).

## Docker operations

Use Makefile targets:

- `make up-build`: rebuild and restart all containers
- `make app`: rebuild and restart app container only
- `make logs`: tail container logs
- `make down`: stop containers
- `make clean`: stop and remove volumes

Docker uses `.env` for all configuration. PostgreSQL and Redis services currently commented out in `docker-compose.yml`.

## Migration

Manual migration (requires PostgreSQL running):

```sh
set -a; . ./.env; set +a
docker exec -i <container_id> psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" < migrations/000001_init.up.sql
```

Current migration is not idempotent. Do not rerun after full or partial application.

Create new migration:

```sh
make migrate-create NAME=description_here
```

Generates sequential numbered files in `migrations/`.

## Configuration

`.env` is source of truth. Copy `.env.example` to `.env` before running.

Required variables: `APP_ENV`, `APP_PORT`, `LOG_LEVEL`, `POSTGRES_*`, `REDIS_*`.

`POSTGRES_PASSWORD` must not be empty. Retry configuration optional (defaults to 5 attempts, 1s initial backoff, 10s max backoff).

Application uses `set -a; . ./.env; set +a` to load environment for local runs. Docker Compose uses `env_file: - .env`.

## Architecture conventions

Structure:

- `cmd/api/main.go`: entrypoint, HTTP server, wiring
- `internal/domain/<context>/`: entities, repository interfaces, business rules
- `internal/usecase/<context>/`: application workflows
- `internal/adapter/http/`: Gin handlers and routing
- `internal/adapter/persistence/postgres/`: GORM repository implementations
- `internal/platform/config/`: config loader, logger, Redis client setup
- `internal/platform/database/`: PostgreSQL connection with retry
- `migrations/`: SQL schema files

Domain contexts: `user`, `academic`, `attendance`, `audit`, `assessment`, `assignment`, `quiz`, `counseling`.

Dependency rule: domain imports nothing from adapter/platform/cmd. Use cases import domain contracts. Adapters import domain and use cases.

PostgreSQL connection uses GORM with retry and exponential backoff. Redis connection uses go-redis/v9 with retry.

Application lifecycle (in `cmd/api/main.go`): load config, create logger, connect PostgreSQL with retry, connect Redis with retry, start HTTP server in goroutine, wait for `SIGINT`/`SIGTERM`, shutdown server, close connections.

## Code patterns

- User handler at `internal/adapter/http/handler/user_handler.go` uses `net/http`, not Gin. Likely transition pattern.
- Router at `internal/adapter/http/router/routes.go` entirely commented out. Current routes live in `cmd/api/main.go`.
- GORM models not yet implemented. Domain entities define structure; persistence adapters incomplete.
- Existing use case at `internal/usecase/user/create.go` references `userdomain.Repository` interface that does not match actual `UserRepository` in `internal/domain/user/user.go`.

When adding code, check neighboring files for existing conventions first.
