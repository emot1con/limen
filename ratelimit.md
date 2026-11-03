# Rate Limiter — Token Bucket + Lua

Dokumen ini menjelaskan implementasi rate limiter berbasis **token bucket**
yang dijalankan atomik di Redis via **Lua script**, plus rencana penerapannya
ke endpoint.

Implementasi ada di:

- `internal/adapter/http/middleware/rate_limit.go` — middleware & presets.
- `internal/adapter/http/middleware/rate_limiter.lua` — algoritma token bucket.

Status:

- **Middleware dan Lua script: sudah diimplementasikan** (termasuk limiter per
  IP, per IP+scope, per identifier, per user, dan per user+operasi).
- **Pemasangan ke route: baru `POST /api/v1/auth/login`.** Bagian 6 adalah
  rencana untuk route lainnya.
- Kode belum diuji end-to-end terhadap Redis nyata; lihat bagian 9 untuk asumsi
  yang perlu diverifikasi.

---

## 1. Ringkasan

- Algoritma: **token bucket** — setiap key punya `capacity` token dan diisi
  ulang dengan laju `refill_rate` token/detik.
- Setiap request mengambil 1 token. Jika token cukup → lanjut; jika tidak →
  `429 Too Many Requests`.
- Seluruh operasi (baca state, isi ulang, ambil token, simpan, set TTL)
  dijalankan **atomik per key** di dalam satu Lua script, sehingga tidak ada
  race antar request atau antar instance aplikasi yang berbagi Redis.
- Limiter **berlapis**: tiap lapisan memakai key yang berbeda sesuai tujuannya
  (IP, identifier/akun, user, user+operasi). Tidak ada satu key yang cocok untuk
  semua endpoint.

Keuntungan dibanding sliding window / fixed window:

- Mengizinkan **burst** sampai `capacity`, tapi tetap membatasi laju rata-rata
  `refill_rate`.
- Tidak ada masalah reset window di batas periode.

Persyaratan Go: **1.21 atau lebih baru** (memakai `max` bawaan dan `log/slog`).

---

## 2. Model Token Bucket

State per key disimpan di Redis Hash:

```
limen... :redis> HGETALL rate_limiter:ip:192.168.10.253
1) "tokens"      2) "4.37"
3) "last_refill" 4) "1730289512345"
```

- `capacity` (maksimum token) — ukuran burst.
- `refill_rate` (token/detik) — laju pengisian ulang jangka panjang.
- `tokens` — token saat ini (boleh pecahan).
- `last_refill` — timestamp (ms) terakhir pengisian; dasar perhitungan refill.

Aturan: rata-rata request yang diloloskan ≤ `refill_rate` req/detik, dengan
toleransi burst hingga `capacity` request sekaligus.

---

## 3. Anatomi Lua Script

File: `internal/adapter/http/middleware/rate_limiter.lua` (tidak berubah).

```lua
local capacity = tonumber(ARGV[1])     -- ukuran bucket (burst)
local refill_rate = tonumber(ARGV[2])  -- token per detik
local now = tonumber(ARGV[3])          -- waktu sekarang (unix ms)

local data = redis.call("HMGET", KEYS[1], "tokens", "last_refill")
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

-- Key baru: bucket penuh.
if tokens == nil then
    tokens = capacity
    last_refill = now
end

-- Isi ulang proporsional waktu yang berjalan.
local delta = now - last_refill
if delta > 0 then
    tokens = math.min(capacity, tokens + (delta / 1000) * refill_rate)
    last_refill = now
end

-- Ambil 1 token bila tersedia.
local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

redis.call("HMSET", KEYS[1], "tokens", tokens, "last_refill", last_refill)
redis.call("PEXPIRE", KEYS[1], math.ceil(capacity / refill_rate * 1000))
return allowed
```

Penjelasan baris kunci:

1. `ARGV[1..3]` — parameter dari Go: `capacity`, `refill_rate`, `now`.
   `KEYS[1]` — key limiter.
2. `HMGET` membaca `tokens` dan `last_refill`.
3. Key baru (`tokens == nil`) → bucket dianggap penuh.
4. `delta / 1000 * refill_rate` — konversi ms→detik lalu kali laju refill.
   `math.min(capacity, ...)` membatasi agar tidak melebihi kapasitas.
5. Jika `tokens >= 1`, kurangi 1 dan loloskan.
6. `HMSET` menyimpan state baru.
7. `PEXPIRE` — TTL = waktu untuk mengisi bucket dari kosong sampai penuh
   (`capacity / refill_rate` detik). Key yang idle akan hilang otomatis,
   sehingga Redis tidak menumpuk key limiter.
8. `return allowed` → `1` lolos, `0` ditolak.

> **Atomik per key**: seluruh blok dijalankan sebagai satu perintah Redis.
> Tidak ada request lain yang bisa menyisip di antara `HMGET` dan `HMSET`.
> Atomisitas ini **tidak** mencakup beberapa limiter sekaligus (lihat 6.2) dan
> tidak membuat jam antar instance aplikasi sinkron (lihat bagian 7).

---

## 4. Middleware

File: `internal/adapter/http/middleware/rate_limit.go`

### 4.1 Alur

```
rateLimiterMiddleware(redis, config, keyFunc):
  1. Jika config nil / Capacity<1 / RefillRate<=0 -> lewat (c.Next()).
  2. limiterKey = keyFunc(c). Jika kosong -> lewat.
  3. Jalankan Lua: script.Run(ctx, redis, [limiterKey], capacity, refillRate, nowMs).
  4. Jika error (Redis bermasalah):
       - log error (slog) dengan path, fail_closed, dan error;
       - config.FailClosed == true  -> balas 503 lalu Abort;
       - config.FailClosed == false -> lewat (fail-open, default).
  5. Jika hasil == 0 -> set header Retry-After, balas 429 lalu Abort.
  6. Jika hasil == 1 -> c.Next().
```

### 4.2 Fail-open vs fail-closed

- **Default: fail-open.** Jika Redis error, request dilewatkan. Ini
  memprioritaskan ketersediaan, dengan konsekuensi proteksi hilang selama Redis
  bermasalah. Error tetap di-log.
- **Opt-in: fail-closed** lewat `config.WithFailClosed()`. Saat Redis error,
  request ditolak dengan `503` (`RATE_LIMITER_UNAVAILABLE`).
- Fail-closed hanya dipakai di endpoint yang risikonya lebih besar daripada
  ketidaktersediaan (login, reset password). Untuk endpoint operasional yang
  harus tetap jalan walau Redis bermasalah (mis. identifikasi sidik jari
  untuk absensi), tetap fail-open.
- Jika key tidak bisa dibentuk (mis. email tidak ada di body), limiter
  **dilewatkan** terlepas dari `FailClosed`. Handler wajib memvalidasi input.

### 4.3 Varian middleware dan format key

| Middleware | Key Redis | Dipakai untuk |
|---|---|---|
| `RateLimiterMiddleware(redis, cfg)` | `rate_limiter:ip:<ip>` | Jaring pengaman global per IP (satu bucket per IP) |
| `ScopedIPRateLimiterMiddleware(redis, cfg, scope)` | `rate_limiter:ip_scope:<scope>:<ip>` | Limiter IP per endpoint/tier dengan config berbeda |
| `IdentifierRateLimiterMiddleware(redis, cfg)` | `rate_limiter:email:<sha256(email)>` | Melindungi satu akun dari brute force terdistribusi |
| `EmailRateLimiterMiddleware(redis, cfg)` | `rate_limiter:ip_email:<ip>:<sha256(email)>` | Kombinasi IP+email (tidak melindungi akun dari banyak IP) |
| `UserRateLimiterMiddleware(redis, cfg)` | `rate_limiter:user:<userID>`; fallback `rate_limiter:user_fallback_ip:<ip>` | Batas total per user setelah auth (satu bucket per user) |
| `UserOperationRateLimiterMiddleware(redis, cfg, op)` | `rate_limiter:user_op:<op>:<userID>`; fallback `rate_limiter:user_op_fallback_ip:<op>:<ip>` | Batas per user per operasi/tier setelah auth |

Normalisasi IP (`ipKeyPart`):

- IPv4 dipakai apa adanya.
- IPv6 diringkas ke prefix **/64**, lalu `:` diganti `_`. Berganti alamat di
  dalam blok /64 yang sama tidak memberi bucket baru.

Email di key di-hash (SHA-256 hex) agar email mentah tidak tersimpan di Redis
dan panjang key konstan.

User ID diambil dari `c.Get(UserIDContextKey)` (default `"userID"`), dengan tipe
`string`, `fmt.Stringer`, `int`, `int64`, `uint`, atau `uint64`. Jika tidak
ditemukan atau tipenya tidak dikenali, limiter jatuh ke **fallback IP** (bukan
dilewatkan), dengan prefix berbeda dari limiter IP biasa.

### 4.4 Aturan bucket: satu key = satu config

Key tidak mengandung `capacity`/`refill_rate`. Jika dua limiter dengan config
berbeda memakai key yang sama, keduanya mengonsumsi **bucket yang sama** dan
saling mengganggu (capacity yang lebih kecil memotong token, dan satu endpoint
menghabiskan jatah endpoint lain).

Konsekuensinya:

- `RateLimiterMiddleware` hanya dipakai **sekali**, sebagai limiter global per
  IP. Limiter IP lain dengan config berbeda memakai `ScopedIPRateLimiterMiddleware`
  dengan `scope` unik.
- Beberapa tier per user memakai `UserOperationRateLimiterMiddleware` dengan
  nama operasi/tier yang unik (`"write"`, `"read_sensitive"`, `"read_master"`).
- `UserRateLimiterMiddleware` (key `user:`) berbeda bucket dari
  `UserOperationRateLimiterMiddleware` (key `user_op:`), jadi keduanya aman
  dipakai bersamaan.
- `IdentifierRateLimiterMiddleware` memakai satu bucket per email untuk semua
  route yang dipasangi. Jika kelak dipakai di beberapa endpoint (login, OTP,
  reset) dan jatahnya harus terpisah, perlu varian dengan scope (belum ada).

### 4.5 Ekstraksi email dari body

`IdentifierRateLimiterMiddleware` dan `EmailRateLimiterMiddleware` memakai
`peekEmail`:

- Body dibaca lewat `http.MaxBytesReader` dengan batas **8 KB**
  (`maxLimiterBodyBytes`).
- Jika body melebihi batas atau gagal dibaca, limiter dilewatkan dan body
  dibiarkan dalam keadaan error, sehingga **handler juga gagal membacanya**.
  Padding body tidak bisa dipakai untuk melewati limiter.
- Jika body valid, email diekstrak, di-lowercase dan di-trim, lalu body
  **dipulihkan** via `io.NopCloser(bytes.NewReader(body))` agar handler di
  belakangnya tetap bisa membacanya.
- Email kosong, lebih dari 254 karakter, bukan UTF-8 valid, atau JSON tidak
  valid → key kosong → limiter dilewatkan. Handler harus menolak input seperti
  itu.
- Jangan pasang limiter email di endpoint yang body JSON sahnya bisa melebihi
  8 KB.

### 4.6 Respons

429 (batas terlampaui):

```json
{ "code": "RATE_LIMITED", "message": "too many requests", "details": { "fields": [] } }
```

Ditambah header `Retry-After` (detik), saat ini `max(1, int(1 / RefillRate))`,
yaitu perkiraan waktu untuk 1 token.

503 (hanya jika `FailClosed` dan Redis error):

```json
{ "code": "RATE_LIMITER_UNAVAILABLE", "message": "service temporarily unavailable" }
```

(Bentuk persis `details` mengikuti `response.AbortError` di proyek.)

---

## 5. Presets

| Fungsi | Capacity (burst) | Refill (token/detik) | Perilaku |
|---|---|---|---|
| `StrictRateLimiter()` | 5 | 1/60 | 1 req/menit, burst 5 |
| `LoginEmailRateLimiter()` | 5 | 1/60 | login: 1 percobaan/menit per bucket, burst 5 (dipakai dengan `IdentifierRateLimiterMiddleware`, yaitu per email) |
| `ConservativeRateLimiter()` | 10 | 5 | 5 req/detik, burst 10 |
| `DefaultRateLimiterConfig()` / `ModerateRateLimiter()` | 20 | 10 | 10 req/detik, burst 20 |
| `GenerousRateLimiter()` | 100 | 50 | 50 req/detik, burst 100 |
| `UnlimitedRateLimiter()` | 10000 | 1000 | praktis tanpa batas |
| `CustomRateLimiter(cap, rate)` | bebas | bebas | kustom |

Modifier: `cfg.WithFailClosed()` mengembalikan salinan config dengan fail-closed
aktif, mis. `LoginEmailRateLimiter().WithFailClosed()`.

---

## 6. Strategi Berlapis dan Rencana Penerapan

### 6.1 Prinsip

- **Tidak ada satu key yang cocok untuk semua.** Tiap jenis endpoint memakai
  key yang sesuai:
  - **Publik (belum ada user):** IP, ditambah identifier (email) untuk
    melindungi akun.
  - **Terautentikasi:** user ID sebagai limiter utama, IP longgar di depan
    sebagai jaring pengaman.
  - **Perangkat (mis. sidik jari):** idealnya device ID/API key, IP sebagai
    cadangan.
- **Waspadai NAT sekolah.** Banyak pengguna (satu kelas atau seluruh sekolah)
  bisa berbagi satu IP publik. Limiter IP yang ketat menghukum pengguna sah
  secara bersamaan, mis. satu kelas login serentak di awal jam pelajaran.
  Karena itu limiter IP dibuat **longgar**, dan pembatasan yang lebih ketat
  dipindahkan ke user ID setelah autentikasi.
- **Sensitif tinggi → bucket kecil + refill lambat; tidak sensitif → longgar.**
- **Satu key = satu config** (lihat 4.4). Setiap tier memakai bucket sendiri.

### 6.2 Urutan dan konsumsi token

Middleware berjalan berurutan, dan limiter pertama yang menolak menghentikan
rantai (`Abort`). Urutan pemasangan = urutan pengecekan.

- Urutan umum: **murah dan luas dulu, spesifik belakangan.**
  IP → identifier (perlu membaca body) → handler.
- Limiter yang meloloskan request **tetap mengonsumsi token** walau limiter
  berikutnya menolak. Token tidak dikembalikan. Ini bisa diterima, tapi
  artinya request yang ditolak tetap "membayar" di limiter sebelumnya.
- Limiter IP dipasang **sebelum** identifier. Tujuannya agar satu IP yang
  menyerang terblokir sebelum sempat menguras bucket akun korban.
- Limiter berbasis user (`UserRateLimiterMiddleware`,
  `UserOperationRateLimiterMiddleware`) **wajib dipasang setelah**
  `authMiddleware.Authenticate()`. Jika dipasang sebelumnya, user ID tidak ada
  dan limiter jatuh ke fallback IP tanpa error.
- Setiap limiter = satu round-trip Redis per request. Jaga rantai tetap pendek
  (global IP + satu limiter tier sudah cukup untuk kebanyakan route).

### 6.3 Definisi tier

| Tier | Kategori | Limiter | Key | Preset awal |
|---|---|---|---|---|
| T5 | Global (semua `/api/v1`) | `RateLimiterMiddleware` | IP | `CustomRateLimiter(300, 100)` |
| T0 | Login (publik) | `ScopedIPRateLimiterMiddleware(..., "login")` lalu `IdentifierRateLimiterMiddleware` | IP+scope, lalu email | IP: `CustomRateLimiter(60, 1)`; email: `LoginEmailRateLimiter().WithFailClosed()` |
| T0b | Aksi akun sensitif (terautentikasi) | `UserOperationRateLimiterMiddleware(..., "password_reset")` | user+op | `StrictRateLimiter().WithFailClosed()` |
| T1 | Biometrik / perangkat | `ScopedIPRateLimiterMiddleware(..., "fingerprint_identify"/"fingerprint_enroll")` | IP+scope | identify: `CustomRateLimiter(10, 2)`; enroll: `CustomRateLimiter(5, 1)` |
| T2 | Mutasi data (write) | `UserOperationRateLimiterMiddleware(..., "write")` | user+op | `ConservativeRateLimiter()` |
| T3 | Baca sensitif / berat | `UserOperationRateLimiterMiddleware(..., "read_sensitive")` | user+op | `ModerateRateLimiter()` |
| T4 | Baca master / reference | `UserOperationRateLimiterMiddleware(..., "read_master")` | user+op | `GenerousRateLimiter()` |

Semua angka adalah **titik awal**, bukan hasil pengukuran. Jalankan dulu dalam
mode shadow (hitung dan log tanpa memblokir) dan kalibrasi dengan trafik nyata,
terutama T5, login, dan T1 yang paling terpengaruh NAT sekolah.

### 6.4 Pemetaan route

| Route | Tier | Limiter |
|---|---|---|
| `POST /auth/login` | T0 | Scoped IP `"login"` → Identifier (fail-closed) |
| `POST /auth/logout` | T5 | Cukup limiter global (tidak sensitif secara kredensial) |
| `POST /users/:userId/password-reset` | T0b | User+op `"password_reset"` (setelah auth + CSRF + role), fail-closed |
| `POST /fingerprint/identify` | T1 | Scoped IP `"fingerprint_identify"` |
| `POST /fingerprint/enroll`, `/enroll/raw` | T1 | Scoped IP `"fingerprint_enroll"` (lebih ketat: menulis) |
| `POST/PUT /settings/school`, `/devices`, `/users`, `/classes`, `/subjects`, `/schedules`, `/teaching-assignments`, `/assessments`, `/students/:id` | T2 | User+op `"write"` |
| `POST /students/:id/target-majors` | T2 | User+op `"write"` |
| `POST/PUT /counseling-notes*` | T2 | User+op `"write"` |
| `POST /assignments`, `PUT /assignments/.../submissions/...` | T2 | User+op `"write"` |
| `POST/PUT/DELETE /question-banks*`, `/questions/:id` | T2 | User+op `"write"` |
| `POST/PUT /quizzes*`, `/quiz-attempts*` | T2 | User+op `"write"` |
| `POST /notifications`, `/notifications/:id/read` | T2 | User+op `"write"` |
| `GET /reports/*`, `/counseling-notes`, `/students/:id/report-card` | T3 | User+op `"read_sensitive"` |
| `GET /users`, `/users/:id`, `/students`, `/students/:id` | T3 | User+op `"read_sensitive"` |
| `GET /question-banks/:id/questions`, `/quizzes`, `/quizzes/:id`, `/quiz-attempts*` | T3 | User+op `"read_sensitive"` |
| `GET /auth/me` | T3 | User+op `"read_sensitive"` |
| `GET /settings/school`, `/devices`, `/classes`, `/subjects`, `/schedules`, `/teaching-assignments`, `/passing-grades*`, `/notifications` | T4 | User+op `"read_master"` |

Catatan pemetaan:

- Satu bucket `"write"` per user dipakai bersama oleh semua endpoint write.
  Itu disengaja: yang dibatasi adalah laju tulis total per user.
- Jika endpoint laporan terbukti mahal, pisahkan menjadi operasi tersendiri
  (mis. `"report"`) dengan config lebih ketat daripada `"read_sensitive"`.
- Pada `POST /quiz-attempts*`, periksa apakah klien menyimpan jawaban per
  soal. Jika ya, `ConservativeRateLimiter` (5 req/detik, burst 10) perlu
  diuji agar tidak menghambat siswa yang sah.

### 6.5 Contoh pemasangan

```go
v1 := router.Group("/api/v1")

// T5 — global per IP (satu-satunya pemakaian RateLimiterMiddleware)
v1.Use(RateLimiterMiddleware(redisClient, CustomRateLimiter(300, 100)))

// T0 — login: IP dulu, baru identifier (akun)
loginIP := ScopedIPRateLimiterMiddleware(redisClient, CustomRateLimiter(60, 1), "login")
loginAccount := IdentifierRateLimiterMiddleware(redisClient, LoginEmailRateLimiter().WithFailClosed())
v1.POST("/auth/login", loginIP, loginAccount, authHandler.Login)

// T1 — biometrik (publik/perangkat: per IP+scope)
fpIdentify := ScopedIPRateLimiterMiddleware(redisClient, CustomRateLimiter(10, 2), "fingerprint_identify")
fpEnroll := ScopedIPRateLimiterMiddleware(redisClient, CustomRateLimiter(5, 1), "fingerprint_enroll")
v1.POST("/fingerprint/identify", fpIdentify, fingerprintHandler.Identify)
v1.POST("/fingerprint/enroll", fpEnroll, fingerprintHandler.Enroll)
v1.POST("/fingerprint/enroll/raw", fpEnroll, fingerprintHandler.EnrollRaw)

// Terautentikasi: auth DULU, baru limiter berbasis user
authed := v1.Group("")
authed.Use(authMiddleware.Authenticate())

writeLimiter := UserOperationRateLimiterMiddleware(redisClient, ConservativeRateLimiter(), "write")
sensitiveRead := UserOperationRateLimiterMiddleware(redisClient, ModerateRateLimiter(), "read_sensitive")
masterRead := UserOperationRateLimiterMiddleware(redisClient, GenerousRateLimiter(), "read_master")

// T0b — aksi akun sensitif
authed.POST("/users/:userId/password-reset",
	authMiddleware.RequireCSRF(),
	authMiddleware.RequireRole("admin"),
	UserOperationRateLimiterMiddleware(redisClient, StrictRateLimiter().WithFailClosed(), "password_reset"),
	authHandler.ResetPassword,
)

// T2/T3/T4 — contoh
adminWrite := authed.Group("")
adminWrite.Use(writeLimiter)
authed.GET("/students", sensitiveRead, studentHandler.List)
authed.GET("/classes", masterRead, classHandler.List)
```

### 6.6 Catatan penerapan

- Pastikan `UserIDContextKey` (default `"userID"`) sama dengan key yang diisi
  `authMiddleware.Authenticate()` lewat `c.Set(...)`. Jika berbeda, semua
  limiter user diam-diam jatuh ke fallback IP.
- Endpoint fingerprint yang dipakai perangkat sebaiknya dibatasi per **device
  ID / API key**, bukan hanya IP (banyak perangkat bisa berbagi satu IP/NAT).
  Key function untuk itu **belum ada**; sementara ini T1 memakai IP+scope.
- Fail-closed dipasang hanya di login dan aksi akun sensitif. Biometrik tetap
  fail-open agar absensi tidak berhenti saat Redis bermasalah (keputusan yang
  bisa Anda ubah).
- Jika `IdentifierRateLimiterMiddleware` memakai limit ketat, pihak lain bisa
  menghabiskan bucket akun korban dan mengunci login akun itu sementara
  (lockout DoS). Mitigasi: jangan terlalu ketat, tambahkan CAPTCHA atau delay
  bertahap di handler, dan pantau 429 per akun.
- Pastikan `c.ClientIP()` akurat. Konfigurasi trusted proxies Gin harus sesuai
  topologi (reverse proxy/load balancer). Jika salah, semua pengguna terlihat
  sebagai satu IP, atau `X-Forwarded-For` bisa dipalsukan.

---

## 7. Kasus Tepi dan Perbaikan Opsional

Sudah ditangani:

- **Fail-closed opsional** dan **log error Redis** (lihat 4.1 dan 4.2).
- **IPv6 /64**, **hash email di key**, **batas body 8 KB** (lihat 4.3 dan 4.5).
- **Limiter per akun tanpa IP** (`IdentifierRateLimiterMiddleware`).

Belum ditangani:

- **Clock skew antar instance**: `now` dikirim dari `time.Now()` masing-masing
  aplikasi. Jam yang tidak sinkron bisa membuat `delta` negatif (diabaikan
  karena `delta > 0`) atau refill tidak akurat. Perbaikan: gunakan `TIME` dari
  Redis (`redis.call("TIME")`) agar sumber waktu tunggal. Verifikasi
  kompatibilitasnya dengan versi Redis yang dipakai.
- **Retry-After lebih akurat**: script bisa mengembalikan sisa token / waktu
  tunggu, lalu Go menghitung `ceil((1 - tokens) / refill_rate)` detik, bukan
  `1/refill_rate` yang mengabaikan defisit saat ini.
- **Header kuota**: kembalikan `X-RateLimit-Limit` dan `X-RateLimit-Remaining`
  dengan mengubah return script menjadi array.
- **Bobot per request**: request mahal mengonsumsi lebih dari satu token
  (perlu parameter cost di Lua).
- **Identifier dengan scope**: bucket terpisah per endpoint untuk limiter
  identifier (lihat 4.4).
- **Limiter per device ID / API key** untuk endpoint perangkat (lihat 6.6).
- **Log berlebih saat Redis down**: satu baris log per request. Pertimbangkan
  sampling.
- **Failover Redis**: replikasi asinkron berarti bucket yang baru ditulis bisa
  hilang saat failover dan limit "reset". Biasanya dapat diterima untuk
  limiter.

---

## 8. Observability dan Testing

**Sudah ada:**

- Log error saat script gagal (`slog.Error`, dengan `path`, `fail_closed`,
  `error`). Ini sinyal Redis tidak sehat.

**Belum ada (disarankan):**

- Metrik `rate_limit_allowed_total{route}` dan `rate_limit_blocked_total{route}`.
- Alert pada lonjakan 429 per route dan per tier.
- Mode shadow (hitung tanpa memblokir) untuk kalibrasi sebelum angka dikunci.

**Testing (disarankan):**

- Unit test dengan `miniredis`: bucket mulai penuh, burst diizinkan, lalu
  tertahan sesuai `refill_rate`.
- Uji refill: majukan waktu (mock `now`) dan pastikan token bertambah.
- Uji TTL: setelah `capacity/refill_rate` detik idle, key hilang.
- Uji fail-open: Redis mati → request lolos dan error ter-log.
- Uji fail-closed: Redis mati dengan `WithFailClosed()` → 503.
- Uji `peekEmail`: body tetap utuh untuk handler; body > 8 KB membuat handler
  gagal membaca (tidak lolos tanpa limit).
- Uji `IdentifierRateLimiterMiddleware`: email sama dari IP berbeda memakai
  bucket yang sama; email berbeda dari IP yang sama memakai bucket berbeda.
- Uji key IPv6: dua alamat dalam /64 yang sama → bucket yang sama.
- Uji isolasi bucket: dua user dari IP yang sama memakai bucket terpisah, dan
  tier `"write"` tidak menghabiskan tier `"read_master"`.
- Uji fallback: tanpa user ID di context, limiter jatuh ke key fallback IP.
- Uji rantai: IP menolak → limiter identifier tidak dijalankan.

---

## 9. Asumsi dan Hal yang Perlu Diverifikasi

- **`UserIDContextKey`**: `"userID"` hanya tebakan; harus dicocokkan dengan
  `authMiddleware.Authenticate()`.
- **`loginRateLimiter` saat ini**: dokumen sebelumnya menyebutnya memakai
  `LoginEmailRateLimiter` (IP+email). Perlu diganti ke kombinasi Scoped IP +
  Identifier sesuai 6.5.
- **Angka preset dan tier** adalah titik awal tanpa data trafik. Khususnya
  limit IP login `(60, 1)`, T5 `(300, 100)`, dan T1 perlu dikalibrasi terhadap
  jumlah pengguna per IP sekolah. Hitungan ini tidak berasal dari data Anda.
- **Handler memvalidasi email**: limiter dilewatkan saat email tidak bisa
  diekstrak, jadi keamanan bergantung pada validasi di handler.
- **Topologi proxy dan trusted proxies Gin**: tidak terlihat dari kode.
- **Versi Redis dan perilaku `TIME` di dalam script**: perlu dicek sebelum
  mengubah sumber waktu.
- **Kompilasi dan pengujian**: kode belum dikompilasi atau dijalankan terhadap
  Redis nyata.

---

## 10. Ringkasan

- Token bucket dijalankan atomik per key via Lua di Redis → konsisten lintas
  instance.
- `capacity` = burst, `refill_rate` = laju rata-rata.
- Berlapis: global IP → IP per scope / identifier (publik) → user per
  operasi (terautentikasi).
- **Satu key = satu config.** Tier berbeda wajib memakai scope/operation
  berbeda.
- Fail-open secara default, fail-closed opsional untuk login dan aksi akun
  sensitif, dengan log error Redis.
- Limiter IP dibuat longgar karena NAT sekolah; pembatasan ketat dipindahkan
  ke user ID dan identifier.
- Semua angka perlu dikalibrasi dengan trafik nyata sebelum dikunci.

---

## Lampiran: Perubahan dari Versi Sebelumnya

| Bagian | Sebelumnya | Sekarang | Alasan |
|---|---|---|---|
| Key per tier | Semua limiter IP memakai `rate_limiter:ip:<ip>` dan dokumen menyebut "setiap limiter memakai key sendiri" | `ScopedIPRateLimiterMiddleware` dan `UserOperationRateLimiterMiddleware` dengan scope unik | Satu key dengan config berbeda berarti satu bucket yang saling mengganggu |
| T0 login | `LoginEmailRateLimiter` (IP+email) + `StrictRateLimiter` per IP | Scoped IP longgar + Identifier (email saja), fail-closed | IP+email tidak melindungi akun dari banyak IP; `Strict` per IP menghukum NAT sekolah |
| `POST /auth/logout` | T0, `StrictRateLimiter` | Cukup limiter global | Bukan operasi kredensial; Strict per IP berisiko memblokir pengguna sah |
| `password-reset` | `StrictRateLimiter` per IP, sebelum auth | User+op setelah auth, fail-closed | Terautentikasi, jadi dibatasi per user dan bukan per IP |
| T2/T3/T4 | Per IP | Per user+operasi | Banyak user berbagi satu IP di sekolah |
| Fail-open | Hanya fail-open | Opsional fail-closed + log | Endpoint sensitif butuh pilihan ini |
| Key IPv6 | Alamat penuh | Prefix /64 | Berganti alamat di blok /64 tidak boleh memberi bucket baru |
| Key email | Email mentah | SHA-256 | Hindari PII di Redis, panjang key konstan |
| Batas body | 1 MB | 8 KB + `MaxBytesReader` | Hindari baca body besar sebelum auth; cegah bypass lewat padding |