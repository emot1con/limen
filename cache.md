# Rencana Implementasi Cache

Dokumen ini menjelaskan rencana penambahan caching pada API Limen, memilih
endpoint yang layak di-cache, dan — yang terpenting — mekanisme **cache
invalidation** yang benar agar tidak terjadi data basi (stale).

Status: **usulan / belum diimplementasikan**.

---

## 1. Tujuan

- Mengurangi beban Postgres untuk endpoint **read-heavy** yang datanya
  **jarang berubah** (master/reference data).
- Menurunkan latency untuk request yang sama berulang kali.
- Menjaga konsistensi: setiap kali data ditulis, cache yang terdampak
  **langsung tidak valid** — bukan menunggu TTL habis.

Non-tujuan:

- Bukan menggantikan pengindeksan database.
- Bukan menyimpan data sensitif/per-user jangka panjang.
- Bukan cache untuk response yang cepat berubah (transaksional/real-time).

---

## 2. Ringkasan Endpoint Kandidat

Kriteria: frekuensi baca tinggi, frekuensi tulis rendah, hasilnya identik untuk
semua pemanggil yang berwenang (atau bisa di-key berdasarkan scope).

| Endpoint | Jenis data | Cakupan key | TTL saran | Trigger invalidasi |
|---|---|---|---|---|
| `GET /settings/school` | setting global | global | 1 jam | `PUT /settings/school` |
| `GET /passing-grades` | referensi | per filter+paginasi | 6–24 jam | *tidak ada write route saat ini → TTL saja* |
| `GET /passing-grades/:passingGradeId` | referensi | per id | 6–24 jam | idem |
| `GET /classes` | master | per filter+paginasi | 15–30 menit | `POST /classes`, `PUT /classes/:classId` |
| `GET /subjects` | master | per filter+paginasi | 15–30 menit | `POST /subjects` |
| `GET /devices` | master | per filter+paginasi | 15–30 menit | `POST /devices`, `PUT /devices/:deviceId` |
| `GET /schedules` | jadwal | per filter+paginasi | 15–30 menit | `POST/PUT/DELETE /schedules` |
| `GET /teaching-assignments` | penugasan | per filter+paginasi | 15–30 menit | `POST /teaching-assignments` |
| `GET /question-banks/:bankId/questions` | konten kuis | per bank | 15–30 menit | `POST/PUT/DELETE` question / question-bank |
| `GET /quizzes` | ringkasan kuis | per filter+paginasi | 5–15 menit | `POST /quizzes`, import/seed |

### 2.1 Wajib diverifikasi sebelum caching

- **`GET /quizzes/:quizId` dan `GET /question-banks/:bankId/questions`**: cek
  apakah payload memuat `answer_key`/kunci jawaban. Jika ya, **jangan** di-cache
  global. Solusi: sertakan `role` (dan bila perlu `user_id`) pada key, atau
  batasi cache hanya untuk role guru/admin.
- **`GET /quizzes`**: pastikan hanya ringkasan (tanpa kunci jawaban).
- Endpoint ini di-gate oleh role di router. Pastikan key memuat scope agar
  tidak bocor lintas role.

### 2.2 Jangan di-cache (non-goals)

- `GET /notifications` — berubah sering, per-user.
- `GET /reports/*` — agregasi yang berubah mengikuti presensi/nilai.
- Assignment submissions, quiz-attempts, attendance.
- `GET /fingerprint/latest`, `GET /fingerprint/stream` (SSE), dan semua endpoint
  fingerprint.
- `GET /auth/me` — sangat sering, tetapi role/sesi mudah berubah. Jika tetap
  ingin di-cache: TTL sangat pendek (≤10 detik), key per `session_id`, dan
  invalidasi saat update user/role. Default: **skip**.
- `GET /students*` — PII, per-konteks. Default: **skip**.

---

## 3. Arsitektur

Tambahkan abstraksi cache di `internal/platform/cache` agar tidak menyebar
pemanggilan Redis mentah ke seluruh usecase.

```
internal/platform/cache/
  cache.go        // interface Cache + key builder + version helpers
  redis.go        // implementasi Redis (JSON), dibuat dari *redis.Client
  noop.go         // implementasi no-op untuk CACHE_ENABLED=false / testing
```

Interface:

```go
package cache

type Cache interface {
    // Get meng-unmarshal ke dest. return value true jika hit.
    Get(ctx context.Context, key string, dest any) (bool, error)
    // Set menyimpan value dengan TTL (sudah termasuk jitter).
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    // Delete menghapus key spesifik (untuk invalidation bertarget).
    Delete(ctx context.Context, keys ...string) error
    // Version membaca generation counter untuk sebuah group (default 0).
    GetVersion(ctx context.Context, group string) (int64, error)
    // BumpVersion menaikkan generation counter secara atomik (INCR).
    BumpVersion(ctx context.Context, group string) (int64, error)
}
```

Wiring:

- `cmd/api/main.go`: buat `cacheClient := cache.NewRedis(redisClient, prefix)`
  dari `redisClient` yang sudah ada, lalu teruskan ke `router.New(...)`.
- `router.New(...)`: teruskan cache ke usecase yang mendukungnya
  (`settings`, `academic`, `aptitude`, `schedule`, `questionbank`, `quiz`).
- Semua usecase yang menerima cache harus **tahan nil**: jika cache `nil`
  (mis. `CACHE_ENABLED=false`), langsung fallback ke database.

### 3.1 Perubahan constructor

Constructor usecase bertambah parameter `cache.Cache`:

```go
func NewUseCase(repo attendancedomain.SchoolSettingsRepository, c cache.Cache, logger *logrus.Logger) *UseCase
```

Di `router.New`, jika cache dimatikan, kirim `cache.NewNoop()`.

---

## 4. Skema Key dan Versioning

Prefix: `limen:cache:{APP_ENV}:` (menghindari tabrakan antar-environment yang
berbagi Redis).

Format key:

```
limen:cache:{env}:{group}:v{version}:{scope}
```

- `group` — nama logis resource, mis. `school-settings`, `passing-grades`,
  `classes`, `devices`, `schedules`, `teaching-assignments`,
  `question-bank-questions`, `quizzes`.
- `version` — generation counter per group (lihat §5.2).
- `scope` — parameter yang memengaruhi hasil: id, filter, paginasi, dan jika
  perlu role. Gunakan hash pendek (mis. SHA-256 8 byte) dari query yang sudah
  dinormalisasi untuk list berfilter.

Contoh:

```
limen:cache:production:school-settings:v3:singleton
limen:cache:production:passing-grades:v1:id:9f2c...
limen:cache:production:classes:v7:list:level=10&q=&page=1&pageSize=20
```

Version counter disimpan di:

```
limen:cache:{env}:{group}:ver
```

Aturan key:
- Key **wajib memuat semua parameter** yang mengubah hasil (filter, sort,
  paginasi, role/scope). Jangan pernah memakai satu key untuk response yang
  berbeda antar pemanggil.
- Jangan menyertakan data yang tidak stabil (timestamp request, request-id).

---

## 5. Mekanisme Cache Invalidation (inti)

Pola yang dipakai: **cache-aside (look-aside) + versioned keys**.

Kunci kebenarannya ada pada **generation counter per group**, bukan
`DEL`/`SCAN` per key. Ini yang membuat invalidation aman dari race.

### 5.1 Alur baca (cache-aside)

1. Baca version group: `ver := GetVersion(group)`.
2. Bangun key dengan `ver`. `Get(key)`.
3. **Hit** → unmarshal, return.
4. **Miss** → query database.
5. `Set(key, value, ttl + jitter)`.
6. Jika DB error → return error (cache tidak diisi). Opsional: **negative
   cache** untuk `ErrNotFound` dengan TTL pendek (mis. 30 detik).

Implementasi Redis harus **fail-open**: error Redis saat baca diperlakukan
sebagai miss dan lanjut ke DB; error Redis saat set/delete hanya di-log, tidak
menggagalkan request.

### 5.2 Alur tulis (invalidation)

Setiap operasi tulis mengikuti aturan berikut:

1. **Tulis ke database lebih dulu**, pastikan commit sukses.
2. **Setelah commit**, panggil `BumpVersion(group)` untuk setiap group yang
   datanya berubah (`INCR` atomik — aman lintas instance).
3. Kembalikan response.

Yang **tidak** dilakukan:
- Tidak memakai `KEYS`/`SCAN` untuk menghapus sekumpulan key. `KEYS` memblokir
  Redis; `SCAN` pun tidak menjamin snapshot. Versioning menghindari keduanya.
- Tidak melakukan invalidasi **sebelum** commit (berisiko cache menyimpan
  kondisi pra-commit atau menghapus lalu diisi ulang oleh request lain).

Contoh pemetaan trigger:

| Operasi tulis | Group yang di-`BumpVersion` |
|---|---|
| `PUT /settings/school` | `school-settings` |
| `POST /classes`, `PUT /classes/:id` | `classes` |
| `POST /subjects` | `subjects`, `classes` (bila kelas menampilkan subject), `schedules` (bila relevan) |
| `POST/PUT /devices` | `devices` |
| `POST/PUT/DELETE /schedules` | `schedules` |
| `POST /teaching-assignments` | `teaching-assignments` |
| `POST/PUT/DELETE question / question-bank` | `question-bank-questions`, `quizzes` |
| `POST /quizzes` | `quizzes` |

> Catatan: satu operasi boleh mem-bump lebih dari satu group bila data
> memengaruhi beberapa endpoint. Lebih baik over-invalidate sedikit (cache
> dingin) daripada under-invalidate (data basi).

### 5.3 Mengapa versioning lebih benar daripada delete-on-write

Race klasik pada cache-aside murni dengan delete-on-write:

```
Waktu  Reader R                         Writer W
t0     read DB (nilai lama)
t1                                      commit nilai baru
t2                                      DEL key           -> cache kosong
t3     SET key = nilai lama             -> data basi tersimpan!
```

Setelah `t3`, cache menyimpan nilai lama dan akan terus disajikan sampai TTL
habis. Ini terjadi karena `DEL` setelah read-then-set tidak menjamin urutan.

Dengan versioned keys:

```
t0     R: GetVersion -> v
t1     R: read DB (nilai lama)
t2     W: commit + INCR -> v+1
t3     R: SET ...:v:... = nilai lama
```

`t3` menulis ke key versi **lama**. Setelah `t2`, semua reader baru memakai
`v+1`, sehingga key di `t3` tidak pernah dibaca lagi dan kedaluwarsa sendiri.
**Tidak ada data basi yang bisa bertahan.** Inilah alasan versioning dipakai
untuk list cache.

Untuk **detail per-id** kita bisa memakai keduanya:

- `BumpVersion(group)` menangani seluruh list sekaligus.
- `Delete(detailKey)` untuk operasi bertarget (update satu entitas) agar tidak
  perlu mem-bump seluruh group saat memungkinkan. Namun karena key detail bisa
  berada di banyak versi, cara paling sederhana dan aman tetap: **bump version
  group**. Hapus key detail hanya sebagai optimisasi.

### 5.4 Konsistensi baca-setelah-tulis

Karena key selalu memuat version yang di-`INCR` setelah commit, request pertama
setelah write pasti **miss** dan membaca nilai terbaru dari DB. Tidak diperlukan
read-your-writes khusus.

### 5.5 Multi-instance

Version counter tersimpan di Redis yang dipakai bersama oleh semua replika.
`INCR` bersifat atomik, sehingga replika mana pun melihat version terbaru segera
setelah bump. Tidak ada state lokal yang perlu disinkronkan.

### 5.6 Kegagalan invalidasi

- Jika `BumpVersion` gagal (Redis down), write **tetap sukses** dan di-log
  warning. Konsistensi dijaga oleh **TTL** sebagai jaring pengaman: paling lama
  data basi hidup selama TTL group.
- TTL harus dianggap sebagai *upper bound* staleness, bukan mekanisme utama.

### 5.7 Cache stampede / thundering herd

Saat key panas kadaluwarsa atau version di-bump, banyak request bisa menembus
ke DB bersamaan. Mitigasi (pilih salah satu/keduanya):

- **singleflight** lokal (`golang.org/x/sync/singleflight`) di usecase: kunci
  berdasarkan cache key sehingga hanya satu goroutine query DB per instance.
- **TTL jitter** (mis. `ttl ± 10%` acak) agar tidak semua key kadaluwarsa
  bersamaan.
- Opsional: distributed lock di Redis untuk key yang sangat panas.

### 5.8 Negative caching

`ErrNotFound` boleh di-cache sangat singkat (≤30 detik) untuk meredam probing.
Karena pembuatan entitas baru mem-bump version group, entri negatif otomatis
tidak terpakai lagi.

---

## 6. Contoh Implementasi

### 6.1 Cache Redis (ringkas)

```go
package cache

type redisCache struct {
    client *redis.Client
    prefix string
}

func (c *redisCache) Get(ctx context.Context, key string, dest any) (bool, error) {
    raw, err := c.client.Get(ctx, c.prefix+key).Bytes()
    if err == redis.Nil {
        return false, nil
    }
    if err != nil {
        return false, err // pemanggil perlakukan sebagai miss (fail-open)
    }
    if err := json.Unmarshal(raw, dest); err != nil {
        return false, err
    }
    return true, nil
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
    raw, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, c.prefix+key, raw, jitter(ttl)).Err()
}

func (c *redisCache) GetVersion(ctx context.Context, group string) (int64, error) {
    val, err := c.client.Get(ctx, c.prefix+group+":ver").Int64()
    if err == redis.Nil {
        return 0, nil
    }
    return val, err
}

func (c *redisCache) BumpVersion(ctx context.Context, group string) (int64, error) {
    return c.client.Incr(ctx, c.prefix+group+":ver").Result()
}
```

### 6.2 Penerapan di usecase `settings`

```go
const groupSchoolSettings = "school-settings"

func (uc *UseCase) Get(ctx context.Context) (Output, error) {
    ver, _ := uc.cache.GetVersion(ctx, groupSchoolSettings)
    key := cache.Key(groupSchoolSettings, ver, "singleton")

    var cached Output
    if hit, err := uc.cache.Get(ctx, key, &cached); err == nil && hit {
        return cached, nil
    }

    out, err := uc.getFromDB(ctx)
    if err != nil {
        return Output{}, err
    }
    _ = uc.cache.Set(ctx, key, out, uc.ttl)
    return out, nil
}

func (uc *UseCase) Update(ctx context.Context, input UpdateInput) (Output, error) {
    out, err := uc.updateDB(ctx, input)
    if err != nil {
        return Output{}, err
    }
    if _, err := uc.cache.BumpVersion(ctx, groupSchoolSettings); err != nil {
        uc.logger.WithError(err).Warn("cache invalidation failed")
    }
    return out, nil
}
```

### 6.3 List berfilter (hashing scope)

```go
func listKey(group string, ver int64, params string) string {
    sum := sha256.Sum256([]byte(params))
    return cache.Key(group, ver, hex.EncodeToString(sum[:8]))
}
```

---

## 7. Konfigurasi

Tambahkan di `internal/platform/config` dan `.env`:

```
CACHE_ENABLED=true
CACHE_TTL_SCHOOL_SETTINGS=1h
CACHE_TTL_PASSING_GRADES=12h
CACHE_TTL_ACADEMIC=20m
CACHE_TTL_SCHEDULES=20m
CACHE_TTL_QUIZ=10m
CACHE_KEY_PREFIX=limen:cache
```

- `CACHE_ENABLED=false` → gunakan `cache.NewNoop()` (tanpa perubahan kode
  usecase).
- Bila Redis tidak tersedia, API tetap berjalan (fail-open).

---

## 8. Observability

- Metrik/log hit & miss per group (`cache_hit_total`, `cache_miss_total`).
- Log warning saat `BumpVersion`/`Set` gagal (tanpa menggagalkan request).
- Log warning bila Redis read error (indikasi redis down, fallback ke DB).

---

## 9. Rencana Bertahap

Fase 0 — Fondasi
1. Buat package `internal/platform/cache` + implementasi Redis & Noop.
2. Tambah config `CACHE_*`.
3. Wiring di `main.go` dan `router.New`.

Fase 1 — Low-risk (read global, write tunggal)
4. `school-settings` (1 read, 1 write) sebagai pilot.
5. `devices`, `teaching-assignments`, `subjects`.

Fase 2 — Master data dengan filter
6. `classes`, `schedules` (CRUD lengkap → uji invalidasi tiap write).
7. `passing-grades` (read-only → TTL saja sampai ada write endpoint).

Fase 3 — Konten kuis
8. Verifikasi payload `/quizzes/:id` & `question-banks/:id/questions` (kunci
   jawaban). Jika aman, cache dengan scope role; jika tidak, key per role atau
   skip.
9. `quizzes` (ringkasan).

Fase 4 — Opsional
10. `auth/me` TTL pendek per-session bila terbukti menguntungkan.
11. singleflight pada key panas.

---

## 10. Pengujian

- **Unit**: gunakan `miniredis` untuk menguji `Get/Set/BumpVersion`.
- **Invalidasi**: untuk setiap endpoint tulis → assert version group naik dan
  pembacaan berikutnya mengambil data baru.
- **Race**: uji skenario §5.3 (reader lambat vs writer) → pastikan tidak ada
  stale value yang bertahan.
- **Fail-open**: simulasikan Redis mati → endpoint tetap 200 dari DB.
- **Kebocoran**: pastikan response berbeda antar role tidak bertabrakan key.

---

## 11. Checklist Rollback

- Set `CACHE_ENABLED=false` (instan, tanpa deploy kode) untuk kembali ke DB.
- Karena cache bersifat best-effort, mematikannya tidak mengubah perilaku
  fungsional, hanya performa.
