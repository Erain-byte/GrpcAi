package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gateway/internal/config"
	"gateway/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func JWTAuth(jwtCfg config.JWTConfig, publicAPIs []string) gin.HandlerFunc {
	secret := strings.TrimSpace(jwtCfg.Secret)

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if isPublicAPI(path, publicAPIs) {
			c.Next()
			return
		}

		if secret == "" {
			if logger.Logger != nil {
				logger.Logger.Error("JWT secret is not configured", zap.String("path", path))
			}
			AuditReject(c, "jwt", "secret_not_configured", http.StatusInternalServerError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "JWT secret is not configured",
			})
			c.Abort()
			return
		}

		tokenStr := extractToken(c)
		if tokenStr == "" {
			AuditReject(c, "jwt", "missing_token", http.StatusUnauthorized)
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "missing or invalid token",
			})
			c.Abort()
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid {
			if logger.Logger != nil {
				logger.Logger.Warn("JWT validation failed",
					zap.Error(err),
					zap.String("path", path),
					zap.String("client_ip", c.ClientIP()),
				)
			}
			AuditReject(c, "jwt", "invalid_token", http.StatusUnauthorized, zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}

		exp, err := claims.GetExpirationTime()
		if err != nil || exp == nil {
			AuditReject(c, "jwt", "missing_exp_claim", http.StatusUnauthorized, zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "token missing exp claim",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("username", claims["username"])
		c.Set("role", claims["role"])

		c.Next()
	}
}

func isPublicAPI(path string, publicAPIs []string) bool {
	for _, api := range publicAPIs {
		if api == path {
			return true
		}
		if strings.HasSuffix(api, "/*") {
			prefix := strings.TrimSuffix(api, "/*")
			if strings.HasPrefix(path, prefix+"/") || path == prefix {
				return true
			}
		}
	}
	return false
}

func extractToken(c *gin.Context) string {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func GenerateToken(jwtCfg config.JWTConfig, userID interface{}, username string, role string) (string, error) {
	expire, err := time.ParseDuration(jwtCfg.Expire)
	if err != nil {
		expire = 24 * time.Hour
	}

	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(expire).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtCfg.Secret))
}
