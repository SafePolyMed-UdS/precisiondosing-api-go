package dsscontroller

import (
	"encoding/json"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/model"

	"github.com/gin-gonic/gin"
)

type AdaptResponse struct {
	OrderID string `json:"order_id"`
	Message string `json:"message"`
}

func (sc *DSSController) AdaptDose(c *gin.Context) {
	query := PatientData{}

	if !handle.JSONBind(c, &query) {
		return
	}

	marshalledQuery, err := json.Marshal(query)
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	newOrder := model.Order{Order: marshalledQuery}
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
