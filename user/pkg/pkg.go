package pkg

import (
	"errors"
	"time"
	"user/internal/types"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret   []byte
	jwtExpire   time.Duration
	jwtIssuer   string
)

// InitJWT 初始化JWT配置
func InitJWT(secret string, expire string, issuer string) {
	jwtSecret = []byte(secret)
	if expire != "" {
		if d, err := time.ParseDuration(expire); err == nil {
			jwtExpire = d
		} else {
			jwtExpire = 24 * time.Hour
		}
	} else {
		jwtExpire = 24 * time.Hour
	}
	jwtIssuer = issuer
}

// JWTClaims JWT claims
type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT token
func GenerateToken(userID uint, username string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    jwtIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析JWT token
func ParseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Success 成功响应
func Success(data interface{}) types.Response {
	return types.Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}
}

// Error 错误响应
func Error(code int, message string) types.Response {
	return types.Response{
		Code:    code,
		Message: message,
	}
}
