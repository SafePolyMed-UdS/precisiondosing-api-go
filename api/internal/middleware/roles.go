package middleware

import (
	"net/http"
	"precisiondosing-api-go/internal/utils/validate"

	"github.com/gin-gonic/gin"
)

func AdminAccessHandler() gin.HandlerFunc {
	return roleHandler("admin")
}

func UserRole(c *gin.Context) string {
	userRole := c.GetString("user_role")
	return userRole
}

func UserMail(c *gin.Context) string {
	userEmail := c.GetString("user_email")
	return userEmail
}

func UserID(c *gin.Context) uint {
	userID := c.GetUint("user_id")
	return userID
}

func roleHandler(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := UserRole(c)
		if err := validate.Access(requiredRole, userRole); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Unauthorized access"})
			return
		}

		c.Next()
	}
}
