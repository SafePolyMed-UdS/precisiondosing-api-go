package testcontroller

import (
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/utils/log"

	"github.com/gin-gonic/gin"
)

type TestController struct {
	logger log.Logger
}

func New() *TestController {
	return &TestController{
		logger: log.WithComponent("testcontroller"),
	}
}

func (tc *TestController) AcceptResult(c *gin.Context) {
	errorMsg := c.PostForm("ErrorMsg")
	orderID := c.Param("orderId")

	// Parse the uploaded file (field name "file")
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		handle.BadRequestError(c, err.Error())
		return
	}
	defer file.Close()

	err = c.SaveUploadedFile(header, "./tmp/uploads/"+header.Filename)
	if err != nil {
		handle.BadRequestError(c, err.Error())
		return
	}

	tc.logger.Info("Received file",
		log.Str("file", header.Filename),
		log.Int("size", int(header.Size)),
		log.Str("orderId", orderID),
		log.Str("errorMsg", errorMsg),
	)

	// Respond OK
	handle.Success(c, gin.H{"message": "Feedback accepted"})
}
