package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/quocanhngo/gotalk/pkg/auth"
	"github.com/redis/go-redis/v9"
)

// AuthMiddleware validates JWT tokens and injects user claims into context
func AuthMiddleware(jwtManager *auth.JWTManager, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format. Use: Bearer <token>"})
			return
		}

		tokenString := parts[1]

		// Check blacklist
		ctx := context.Background()
		exists, err := rdb.Exists(ctx, "blacklist:"+tokenString).Result()
		if err != nil {
			// Redis error, fail safe or fail closed? Fail closed for security.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth server error"})
			return
		}
		if exists > 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
			return
		}

		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Store user info in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}
