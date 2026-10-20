# Development

## Requirements

- Go 1.25.10
- PostgreSQL 15
- Redis 7

## Configuration

Copy `.env.example` to `.env` and set required values.

`.env` is source of truth for application configuration.

Required variables:

```dotenv
APP_ENV=development
APP_PORT=8000
LOG_LEVEL=info

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=limen
POSTGRES_USER=postgres
POSTGRES_PASSWORD=psql
POSTGRES_SSLMODE=disable

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

Optional retry configuration:

```dotenv
POSTGRES_RETRY_ATTEMPTS=5
POSTGRES_RETRY_INITIAL_BACKOFF=1s
POSTGRES_RETRY_MAX_BACKOFF=10s
REDIS_RETRY_ATTEMPTS=5
REDIS_RETRY_INITIAL_BACKOFF=1s
REDIS_RETRY_MAX_BACKOFF=10s
```

## Run Application

Start PostgreSQL and Redis before starting API.

```sh
set -a; . ./.env; set +a
go run ./cmd/api
```

API health check:

```sh
curl http://localhost:8000/health
```

Stop API with `Ctrl+C`. The server handles `SIGINT` and `SIGTERM`, then closes HTTP server, PostgreSQL, and Redis connections.

## Migration

Migration targets existing PostgreSQL database from `.env`. It does not create a database.

```sh
set -a; . ./.env; set +a
docker exec -i 090fa45cdcc4 psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" < migrations/000001_init.up.sql
```

Current migration is not repeatable. Do not rerun it after full or partial application.

## Testing

Unit and integration testing are deferred until system implementation is complete.
