package middleware

import (
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

type CasbinMiddleware struct {
	enforcer *casbin.Enforcer
}

func NewCasbinMiddleware(enforcer *casbin.Enforcer) *CasbinMiddleware {
	return &CasbinMiddleware{enforcer: enforcer}
}

func (m *CasbinMiddleware) RequirePermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := GetRoles(c)
		if !exists || len(roles) == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40300,
				"message": "no roles assigned",
			})
			return
		}

		resource := extractResource(c.Request.URL.Path)
		action := mapMethodToAction(c.Request.Method)

		allowed := false
		for _, role := range roles {
			ok, err := m.enforcer.Enforce(role, resource, action)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    50001,
					"message": "permission check failed",
				})
				return
			}
			if ok {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "permission denied",
			})
			return
		}

		c.Next()
	}
}

func extractResource(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}
	return "/" + parts[0]
}

func mapMethodToAction(method string) string {
	switch method {
	case http.MethodGet:
		return "read"
	case http.MethodPost:
		return "write"
	case http.MethodPut, http.MethodPatch:
		return "write"
	case http.MethodDelete:
		return "write"
	default:
		return "read"
	}
}

func (m *CasbinMiddleware) Enforcer() *casbin.Enforcer {
	return m.enforcer
}
