package testcontroller

import (
	"fmt"
	"precisiondosing-api-go/internal/handle"

	"github.com/gin-gonic/gin"
)

func AcceptResult(c *gin.Context) {
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

	fmt.Printf("Received OrderID: %s\n", orderID)
	fmt.Printf("Received ErrorMsg: %s\n", errorMsg)
	fmt.Printf("Received file: %s (%d bytes)\n", header.Filename, header.Size)

	// Respond OK
	handle.Success(c, gin.H{"message": "Feedback accepted"})
}
