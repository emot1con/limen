# Sanedu Backend

## Struktur

```text
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── user/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   └── errors.go
│   │   └── .../
│   ├── usecase/
│   │   ├── user/
│   │   │   ├── create.go
│   │   │   └── create_test.go
│   │   └── .../
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── handler/
│   │   │   ├── middleware/
│   │   │   └── router.go
│   │   └── persistence/
│   │       └── postgres/
│   │           ├── user_repository.go
│   │           └── ...
│   └── platform/
│       ├── config/
│       ├── database/
│       └── logger/
├── migrations/
├── pkg/
├── tests/
├── .env
└── go.mod
```

## Batas dependensi

- `domain`: entity, aturan bisnis, interface repository; tidak bergantung pada framework atau database.
- `usecase`: alur aplikasi; bergantung pada `domain`.
- `adapter`: implementasi HTTP dan persistence; menerjemahkan input/output ke domain/usecase.
- `platform`: konfigurasi serta inisialisasi teknis seperti database dan logger.
- `cmd/api`: merangkai dependency dan menjalankan server.

Untuk 30+ entity, kelompokkan berdasarkan bounded context/fitur (misalnya `user`, `school`, `enrollment`), bukan berdasarkan jenis file global. Setiap fitur punya paket domain dan usecase sendiri; adapter dapat dipisah per fitur juga. Letakkan interface di sisi yang membutuhkan, dan hindari paket `models` atau `repository` besar yang memuat semua entity.
