package testcontroller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AcceptResult(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Feedback accepted",
	})
}
