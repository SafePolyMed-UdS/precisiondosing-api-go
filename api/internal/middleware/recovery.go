package middleware

import (
	"errors"
	"precisiondosing-api-go/internal/handle"

	"github.com/gin-gonic/gin"
)

func RecoveryHandler(c *gin.Context, err any) {
	realErr, ok := err.(error)
	if !ok {
		realErr = errors.New("unexpected API error")
	}

	handle.ServerError(c, realErr)
	c.Abort()
}
