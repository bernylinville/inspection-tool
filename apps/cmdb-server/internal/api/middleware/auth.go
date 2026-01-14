package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/service/auth"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	ContextKeyUserID    = "user_id"
	ContextKeyUsername  = "username"
	ContextKeyRoles     = "roles"
)

var (
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidAuthFormat = errors.New("invalid authorization format")
)

type AuthMiddleware struct {
	authService *auth.AuthService
}

func NewAuthMiddleware(authService *auth.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := extractToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    40100,
				"message": err.Error(),
			})
			return
		}

		claims, err := m.authService.ValidateToken(token)
		if err != nil {
			statusCode := http.StatusUnauthorized
			errorCode := 40101
			message := "invalid token"

			if errors.Is(err, auth.ErrTokenExpired) {
				errorCode = 40102
				message = "token expired"
			}

			c.AbortWithStatusJSON(statusCode, gin.H{
				"code":    errorCode,
				"message": message,
			})
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyRoles, claims.Roles)

		c.Next()
	}
}

func extractToken(c *gin.Context) (string, error) {
	header := c.GetHeader(AuthorizationHeader)
	if header == "" {
		return "", ErrMissingAuthHeader
	}

	if !strings.HasPrefix(header, BearerPrefix) {
		return "", ErrInvalidAuthFormat
	}

	return strings.TrimPrefix(header, BearerPrefix), nil
}

func GetUserID(c *gin.Context) (int64, bool) {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return 0, false
	}
	userID, ok := val.(int64)
	return userID, ok
}

func GetUsername(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyUsername)
	if !exists {
		return "", false
	}
	username, ok := val.(string)
	return username, ok
}

func GetRoles(c *gin.Context) ([]string, bool) {
	val, exists := c.Get(ContextKeyRoles)
	if !exists {
		return nil, false
	}
	roles, ok := val.([]string)
	return roles, ok
}
