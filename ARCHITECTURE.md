# Limen Backend Architecture

## Architecture Style

Clean Architecture with dependency inversion: domain layer defines contracts; adapters implement them.

## Technology Stack

- **Framework**: Gin (HTTP routing and middleware)
- **ORM**: GORM (PostgreSQL persistence)
- **Cache**: go-redis/v9
- **Logger**: Logrus
- **Language**: Go 1.25.10

## Structure

```text
cmd/api/main.go                         Application bootstrap and HTTP server
internal/domain/<context>/              Entity, domain types, repository interfaces, errors
internal/usecase/<context>/             Application workflows
internal/adapter/http/                  HTTP handlers and routing
internal/adapter/persistence/postgres/  PostgreSQL repository adapters
internal/platform/config/               Environment config and logger setup
internal/platform/database/             PostgreSQL and Redis connections
migrations/                             PostgreSQL schema migrations
```

## Domain Contexts

- `user`
- `academic`
- `attendance`
- `audit`
- `assessment`
- `assignment`
- `quiz`
- `counseling`

Entity grouping and cross-domain dependencies live in [DOMAIN_MODEL.md](DOMAIN_MODEL.md).

## Dependency Boundaries

- `domain`: entities, business rules, typed errors, and repository interfaces. No HTTP, database, cache, or framework imports.
- `usecase`: application workflows. Depends on domain contracts.
- `adapter`: translates HTTP and persistence concerns to domain/usecase inputs and outputs.
- `platform`: technical setup for configuration, logging, PostgreSQL, and Redis.
- `cmd/api`: composition root. Creates dependencies and runs the Gin HTTP server.
- PostgreSQL connections use GORM; Redis connections use go-redis/v9.

Dependencies must point inward: adapter and platform code may import domain/usecase; domain must not import adapter, platform, or `cmd`.

## Application Lifecycle

`cmd/api/main.go` performs this sequence:

1. Load configuration from environment.
2. Create Logrus logger.
3. Connect PostgreSQL with retry and exponential backoff.
4. Connect Redis with retry and exponential backoff.
5. Start HTTP server in a goroutine.
6. Wait for `SIGINT` or `SIGTERM`.
7. Gracefully stop HTTP server and close PostgreSQL and Redis connections.

## Configuration

`.env` is source of truth. Variable definitions and local setup are in [DEVELOPMENT.md](DEVELOPMENT.md).

## Current HTTP Surface

- `GET /health`: returns application health response after PostgreSQL and Redis connect during startup.

Business endpoints, authorization, and API contracts are not finalized.
