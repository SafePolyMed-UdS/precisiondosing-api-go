package dsscontroller

import (
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/model"

	"encoding/json"

	"github.com/gin-gonic/gin"
)

type AdaptResponse struct {
	OrderID string `json:"order_id"`
	Message string `json:"message"`
}

func (sc *DSSController) AdaptDose(c *gin.Context) {
	patientData, err := sc.readPatientData(c)
	if err != nil {
		handle.BadRequestError(c, err.Error())
		return
	}

	marshalledData, _ := json.Marshal(patientData)
	newOrder := model.Order{Order: marshalledData}
	if err = sc.DB.Create(&newOrder).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	result := AdaptResponse{
		OrderID: newOrder.OrderID,
		Message: "Order queued",
	}

	handle.Success(c, result)
}
