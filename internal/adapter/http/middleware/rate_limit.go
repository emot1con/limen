package middleware

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
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

type RateLimiterConfig struct {
	Capacity   int
	RefillRate float64
}

func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{Capacity: 20, RefillRate: 10}
}

func RateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig) gin.HandlerFunc {
	return rateLimiterMiddleware(redisClient, config, func(c *gin.Context) string {
		return IPRateLimiterKey(c.ClientIP())
	})
}

func EmailRateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig) gin.HandlerFunc {
	return rateLimiterMiddleware(redisClient, config, func(c *gin.Context) string {
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
		if err != nil {
			return ""
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		var request struct {
			Email string `json:"email"`
		}
		decoder := json.NewDecoder(bytes.NewReader(body))
		if decoder.Decode(&request) != nil || len(request.Email) > 254 || !utf8.ValidString(request.Email) {
			return ""
		}
		request.Email = strings.ToLower(strings.TrimSpace(request.Email))
		if request.Email == "" {
			return ""
		}
		return "rate_limiter:ip_email:" + IPRateLimiterKey(c.ClientIP()) + ":" + request.Email
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

func IPRateLimiterKey(ip string) string {
	if parsed := net.ParseIP(ip); parsed != nil {
		ip = parsed.String()
	}
	return "rate_limiter:ip:" + strings.ReplaceAll(ip, ":", "_")
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
