# LIMEN

Backend Sistem Informasi Akademik dan Manajemen Sekolah. Fitur aktif fokus pada autentikasi, pengelolaan pengguna, pengaturan sekolah, perangkat absensi, data akademik, jadwal, dan penugasan mengajar.

> Status: endpoint aktif belum mencakup semua domain pada requirement dan `openapi.yaml`. Fitur absensi, nilai, tugas, kuis, konseling, minat-bakat, dan passing grade masih berupa skema/kontrak atau belum tersedia pada router aktif.

## Stack

- Go 1.25.10
- Gin
- GORM + PostgreSQL 15
- Redis 7
- Docker Compose

## Prasyarat

Siapkan salah satu lingkungan berikut:

- Go 1.25.10, PostgreSQL 15, Redis 7 untuk jalan lokal.
- Docker dan Docker Compose untuk Redis dan aplikasi dalam container.

## Konfigurasi

Salin contoh konfigurasi:

```sh
cp .env.example .env
```

Atur `.env` sesuai lingkungan:

| Variabel | Keterangan |
| --- | --- |
| `APP_ENV` | Lingkungan aplikasi, contoh `development`. |
| `APP_PORT` | Port HTTP aplikasi saat jalan lokal. |
| `LOG_LEVEL` | Level log aplikasi. |
| `POSTGRES_DB` | Nama database PostgreSQL. |
| `POSTGRES_USER` | Pengguna PostgreSQL. |
| `POSTGRES_PASSWORD` | Kata sandi PostgreSQL. |
| `POSTGRES_HOST` / `POSTGRES_PORT` | Host dan port PostgreSQL. |
| `POSTGRES_SSLMODE` | Mode SSL PostgreSQL. |
| `POSTGRES_RETRY_ATTEMPTS` | Batas percobaan koneksi PostgreSQL. |
| `POSTGRES_RETRY_INITIAL_BACKOFF` / `POSTGRES_RETRY_MAX_BACKOFF` | Jeda retry PostgreSQL. |
| `REDIS_HOST` / `REDIS_PORT` | Host dan port Redis. |
| `REDIS_PASSWORD` | Kata sandi Redis, kosong bila tidak dipakai. |
| `REDIS_DB` | Nomor database Redis. |
| `REDIS_RETRY_ATTEMPTS` | Batas percobaan koneksi Redis. |
| `REDIS_RETRY_INITIAL_BACKOFF` / `REDIS_RETRY_MAX_BACKOFF` | Jeda retry Redis. |

Jangan commit `.env`. Ganti kredensial contoh sebelum dipakai di lingkungan bersama atau produksi.

## Database dan migrasi

Buat database PostgreSQL sesuai `POSTGRES_DB`, lalu jalankan migrasi:

```sh
make migrate-up
```

Target migrasi menjalankan `psql` di container bernama `postgres`. PostgreSQL harus berjalan sebagai container dengan nama tersebut.

Buat file migrasi baru:

```sh
make migrate-create NAME=add_table_name
```

Rollback semua migrasi yang tercatat:

```sh
make migrate-down
```

## Admin awal

Buat atau perbarui akun admin setelah migrasi:

```sh
make seed-admin \
  ADMIN_EMAIL=admin@example.com \
  ADMIN_FULL_NAME="Admin Sekolah" \
  ADMIN_PASSWORD='ganti-dengan-password-kuat'
```

Perintah ini membutuhkan PostgreSQL aktif dan konfigurasi `.env` benar.

## Menjalankan aplikasi

### Lokal

Pastikan PostgreSQL dan Redis aktif, lalu ekspor variabel dari `.env`:

```sh
set -a; . ./.env; set +a
go run ./cmd/api
```

Saat ini `cmd/api` belum ada di working tree. Lengkapi entry point ini sebelum menjalankan mode lokal.

### Docker Compose

```sh
make up-build
```

Layanan Compose saat ini hanya menjalankan `app` dan `redis`; layanan PostgreSQL masih dikomentari pada `docker-compose.yml`. Sediakan PostgreSQL terpisah dan set `POSTGRES_HOST` yang dapat dijangkau dari container aplikasi.

Port Docker aplikasi dipetakan ke `8080`, sedangkan contoh `.env` memakai `APP_PORT=8000`. Selaraskan port aplikasi dengan pemetaan Docker sebelum menjalankan container.

Perintah umum:

```sh
make up       # mulai container
make app      # build ulang dan mulai app
make logs     # lihat log berkelanjutan
make ps       # lihat status container
make down     # hentikan container
```

`make clean` menghapus container dan volume Compose. Data Redis dalam volume akan hilang.

## API

Base path API: `/api/v1`.

Autentikasi memakai sesi berbasis cookie. Endpoint tulis yang sudah masuk membutuhkan token CSRF. Endpoint administrasi membutuhkan peran `admin`.

| Area | Endpoint aktif |
| --- | --- |
| Auth | `POST /auth/login`, `POST /auth/logout`, `GET /auth/me` |
| Kata sandi | `POST /users/:userId/password-reset` |
| Pengaturan | `GET`, `PUT /settings/school` |
| Perangkat | `GET`, `POST /devices`, `PUT /devices/:deviceId` |
| Pengguna | `GET`, `POST /users`, `GET`, `PUT /users/:userId` |
| Akademik | `GET`, `POST`, `PUT /classes`; `GET`, `POST /subjects`; `GET /students`; `GET`, `PATCH /students/:studentId` |
| Jadwal | `GET`, `POST /schedules`; `PUT`, `DELETE /schedules/:scheduleId` |
| Penugasan | `GET`, `POST /teaching-assignments` |

Spesifikasi API tersedia di `openapi.yaml`, tetapi sebagian endpoint di sana masih berstatus usulan. Gunakan daftar endpoint aktif di atas sebagai acuan operasional.

Jalankan Swagger UI lokal:

```sh
make swag-up
```

Swagger UI tersedia pada port `8090` secara default. Hentikan dengan:

```sh
make swag-down
```

## Backup dan restore

Backup PostgreSQL:

```sh
make db-backup
```

Restore PostgreSQL:

```sh
make db-restore FILE=backups/postgres_YYYYMMDD_HHMMSS.sql.gz
```

Backup Redis:

```sh
make redis-backup
```

Restore Redis:

```sh
make redis-restore FILE=backups/redis_YYYYMMDD_HHMMSS.rdb.gz
```

Restore mengganti data database atau Redis target. Pastikan file backup benar dan buat backup terbaru sebelum restore.

## Struktur proyek

```text
cmd/                    entry point aplikasi dan utilitas
internal/domain/        model serta aturan domain
internal/usecase/       logika aplikasi
internal/adapter/       HTTP handler, middleware, dan persistence
internal/platform/      konfigurasi serta koneksi infrastruktur
migrations/             migrasi PostgreSQL
openapi.yaml            kontrak API
```

Dependency mengikuti arah Clean Architecture: adapter bergantung pada use case dan domain; domain tidak bergantung pada adapter atau infrastruktur.
