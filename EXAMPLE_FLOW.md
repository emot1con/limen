# Contoh Alur Dua Entity

Entity contoh: `User` dan `School`.

```text
HTTP request
  └─ adapter/http/handler
       └─ usecase/<entity>
            └─ domain/<entity>.Repository (interface)
                 └─ adapter/persistence/postgres (implementasi)
```

```text
cmd/api/main.go                                      composition root
internal/domain/user/entity.go                        User + Repository interface
internal/domain/school/entity.go                      School + Repository interface
internal/usecase/user/create.go                       aturan create User
internal/usecase/school/create.go                     aturan create School
internal/adapter/http/handler/user_handler.go         HTTP controller User
internal/adapter/http/handler/school_handler.go       HTTP controller School
internal/adapter/persistence/postgres/user_repository.go
internal/adapter/persistence/postgres/school_repository.go
```

Jalankan server:

```sh
go run ./cmd/api
```

Coba endpoint:

```sh
curl -X POST localhost:8080/users -H 'Content-Type: application/json' -d '{"ID":"usr-1","Name":"Budi","Email":"budi@example.com"}'
curl -X POST localhost:8080/schools -H 'Content-Type: application/json' -d '{"ID":"sch-1","Name":"SMA Negeri 1"}'
```

Repository saat ini in-memory supaya contoh dapat jalan tanpa database. Ganti isi `internal/adapter/persistence/postgres/*_repository.go` dengan query PostgreSQL tanpa mengubah domain, usecase, atau handler.
