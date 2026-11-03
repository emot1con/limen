package middleware

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"limen/internal/adapter/http/response"
)

//go:embed rate_limiter.lua
var luaScript string

// UserIDContextKey adalah key yang dipakai middleware auth via c.Set(...).
// SESUAIKAN dengan kode auth Anda.
const UserIDContextKey = "userID"

// maxLimiterBodyBytes membatasi body yang dibaca untuk mengekstrak identifier.
// Body yang lebih besar ditolak oleh http.MaxBytesReader, dan handler juga
// akan gagal membacanya, jadi padding body tidak bisa dipakai untuk melewati limiter.
const maxLimiterBodyBytes = 8 << 10

type RateLimiterConfig struct {
	Capacity   int
	RefillRate float64
	// FailClosed: jika true, request ditolak (503) saat Redis error.
	// Default false (fail-open: request diloloskan, error tetap di-log).
	FailClosed bool
}

// WithFailClosed mengembalikan salinan config dengan FailClosed aktif.
// Pakai untuk endpoint sensitif (login, OTP, reset password).
func (c *RateLimiterConfig) WithFailClosed() *RateLimiterConfig {
	if c == nil {
		return nil
	}
	cp := *c
	cp.FailClosed = true
	return &cp
}

func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{Capacity: 20, RefillRate: 10}
}

// RateLimiterMiddleware: key per IP (IPv6 dinormalisasi ke /64).
func RateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig) gin.HandlerFunc {
	return rateLimiterMiddleware(redisClient, config, func(c *gin.Context) string {
		return IPRateLimiterKey(c.ClientIP())
	})
}

// ScopedIPRateLimiterMiddleware: key per IP + scope (nama endpoint/tier,
// mis. "login", "fingerprint_identify"). Wajib dipakai (bukan RateLimiterMiddleware)
// bila beberapa limiter IP dengan config BERBEDA dipasang di route berbeda,
// karena RateLimiterMiddleware selalu memakai satu key per IP.
func ScopedIPRateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig, scope string) gin.HandlerFunc {
	return rateLimiterMiddleware(redisClient, config, func(c *gin.Context) string {
		return "rate_limiter:ip_scope:" + scope + ":" + ipKeyPart(c.ClientIP())
	})
}

// EmailRateLimiterMiddleware: key per kombinasi IP + email.
// CATATAN: tidak melindungi satu akun dari serangan terdistribusi
// (banyak IP = banyak bucket). Untuk itu pakai IdentifierRateLimiterMiddleware.
func EmailRateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig) gin.HandlerFunc {
	return rateLimiterMiddleware(redisClient, config, func(c *gin.Context) string {
		email := peekEmail(c)
		if email == "" {
			return ""
		}
		return "rate_limiter:ip_email:" + ipKeyPart(c.ClientIP()) + ":" + hashKeyPart(email)
	})
}

// IdentifierRateLimiterMiddleware: key per email saja (tanpa IP).
// Melindungi satu akun dari brute force terdistribusi. Tradeoff: pihak lain
// bisa menghabiskan bucket akun korban (lockout DoS), jadi pasangkan dengan
// limiter IP yang dipasang LEBIH DULU, dan jangan set limit terlalu ketat.
func IdentifierRateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig) gin.HandlerFunc {
	return rateLimiterMiddleware(redisClient, config, func(c *gin.Context) string {
		email := peekEmail(c)
		if email == "" {
			return ""
		}
		return "rate_limiter:email:" + hashKeyPart(email)
	})
}

// UserRateLimiterMiddleware membatasi request per user ID.
// Harus dipasang SETELAH middleware auth. Jika user ID tidak ditemukan,
// fallback ke IP (bucket terpisah dari RateLimiterMiddleware).
func UserRateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig) gin.HandlerFunc {
	return rateLimiterMiddleware(redisClient, config, func(c *gin.Context) string {
		if id := userIDFromContext(c); id != "" {
			return UserRateLimiterKey(id)
		}
		return "rate_limiter:user_fallback_ip:" + ipKeyPart(c.ClientIP())
	})
}

// UserOperationRateLimiterMiddleware membatasi operasi tertentu per user ID
// (mis. "export", "send_email", "change_password"), dengan bucket terpisah
// dari limiter umum. Pasang setelah auth.
func UserOperationRateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig, operation string) gin.HandlerFunc {
	return rateLimiterMiddleware(redisClient, config, func(c *gin.Context) string {
		if id := userIDFromContext(c); id != "" {
			return "rate_limiter:user_op:" + operation + ":" + id
		}
		return "rate_limiter:user_op_fallback_ip:" + operation + ":" + ipKeyPart(c.ClientIP())
	})
}

func rateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig, key func(*gin.Context) string) gin.HandlerFunc {
	script := redis.NewScript(luaScript)
	return func(c *gin.Context) {
		if config == nil || config.Capacity < 1 || config.RefillRate <= 0 {
			c.Next()
			return
		}
		limiterKey := key(c)
		if limiterKey == "" {
			c.Next()
			return
		}
		result, err := script.Run(c.Request.Context(), redisClient, []string{limiterKey}, config.Capacity, config.RefillRate, time.Now().UnixMilli()).Int64()
		if err != nil {
			slog.Error("rate limiter: redis script failed",
				"path", c.FullPath(),
				"fail_closed", config.FailClosed,
				"error", err,
			)
			if config.FailClosed {
				response.AbortError(c, http.StatusServiceUnavailable, "RATE_LIMITER_UNAVAILABLE", "service temporarily unavailable")
				return
			}
			c.Next()
			return
		}
		if result == 0 {
			c.Header("Retry-After", strconv.Itoa(max(1, int(1/config.RefillRate))))
			response.AbortError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
			return
		}
		c.Next()
	}
}

// peekEmail membaca field "email" dari body JSON lalu mengembalikan body
// agar handler tetap bisa membacanya. Mengembalikan "" jika email tidak
// ada/tidak valid (limiter dilewati; handler tetap bertanggung jawab validasi).
func peekEmail(c *gin.Context) string {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLimiterBodyBytes)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		// Body terlalu besar atau error baca. Body dibiarkan dalam keadaan error
		// sehingga handler juga gagal membacanya.
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var request struct {
		Email string `json:"email"`
	}
	if json.NewDecoder(bytes.NewReader(body)).Decode(&request) != nil ||
		len(request.Email) > 254 || !utf8.ValidString(request.Email) {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(request.Email))
}

func userIDFromContext(c *gin.Context) string {
	v, ok := c.Get(UserIDContextKey)
	if !ok {
		return ""
	}
	return userIDToString(v)
}

// hashKeyPart menghindari menyimpan email mentah di key Redis dan
// menjaga panjang key tetap konstan.
func hashKeyPart(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ipKeyPart menormalisasi IP: IPv4 apa adanya, IPv6 diringkas ke prefix /64
// (satu pengguna umumnya mendapat blok /64, jadi berganti alamat di dalam
// blok tidak memberi bucket baru).
func ipKeyPart(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return strings.ReplaceAll(ip, ":", "_")
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	return strings.ReplaceAll(parsed.Mask(net.CIDRMask(64, 128)).String(), ":", "_")
}

func IPRateLimiterKey(ip string) string {
	return "rate_limiter:ip:" + ipKeyPart(ip)
}

func UserRateLimiterKey(userID string) string {
	return "rate_limiter:user:" + userID
}

func userIDToString(v any) string {
	switch id := v.(type) {
	case string:
		return id
	case fmt.Stringer: // uuid.UUID dll.
		return id.String()
	case int:
		return strconv.Itoa(id)
	case int64:
		return strconv.FormatInt(id, 10)
	case uint:
		return strconv.FormatUint(uint64(id), 10)
	case uint64:
		return strconv.FormatUint(id, 10)
	default:
		return ""
	}
}

func StrictRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{Capacity: 5, RefillRate: 1.0 / 60}
}
func LoginEmailRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{Capacity: 5, RefillRate: 1.0 / 60}
}
func ConservativeRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{Capacity: 10, RefillRate: 5}
}
func ModerateRateLimiter() *RateLimiterConfig { return DefaultRateLimiterConfig() }
func GenerousRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{Capacity: 100, RefillRate: 50}
}
func UnlimitedRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{Capacity: 10000, RefillRate: 1000}
}
func CustomRateLimiter(capacity int, refillRate float64) *RateLimiterConfig {
	return &RateLimiterConfig{Capacity: capacity, RefillRate: refillRate}
}