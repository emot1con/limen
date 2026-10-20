# API

## Base URL

Local development:

```text
http://localhost:8000
```

Port follows `APP_PORT` in `.env`.

## Health Check

```http
GET /health
```

Response:

```json
{"status":"ok"}
```

Status: `200 OK`.

## Deferred

Business endpoints, request/response schemas, authentication, authorization, pagination, error format, and API versioning are not finalized.
