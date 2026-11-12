package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS membatasi origin yang diizinkan berdasarkan daftar domain dari konfigurasi.
// Mendukung wildcard "*" (origin request akan di-echo agar kompatibel dengan credentials).
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}
	_, wildcard := allowed["*"]

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			_, ok := allowed[origin]
			if wildcard || ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
				c.Header("Access-Control-Max-Age", "86400")
				c.Header("Vary", "Origin")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
