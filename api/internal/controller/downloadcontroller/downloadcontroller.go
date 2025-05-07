package downloadcontroller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/model"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DownloadController struct {
	DB *gorm.DB
}

func New(resourceHandle *handle.ResourceHandle) *DownloadController {
	return &DownloadController{
		DB: resourceHandle.Databases.GormDB,
	}
}

func (ac *DownloadController) DownloadPDF(c *gin.Context) {
	orderID := c.Param("order_id")

	var order model.Order
	if err := ac.DB.
		Select("order_id", "process_result_pdf").
		Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	if order.ProcessResultPDF == nil {
		handle.NotFoundError(c, "No PDF attached for this order")
		return
	}

	pdfBytes, err := base64.StdEncoding.DecodeString(*order.ProcessResultPDF)
	if err != nil {
		handle.ServerError(c, fmt.Errorf("failed to decode PDF: %w", err))
		return
	}

	// Send the file
	c.Writer.WriteHeader(http.StatusOK)
	if _, err = c.Writer.Write(pdfBytes); err != nil {
		handle.ServerError(c, fmt.Errorf("failed to write PDF to response: %w", err))
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"order_%s.pdf\"", order.OrderID))
	c.Header("Content-Length", strconv.Itoa(len(pdfBytes)))
}

func (ac *DownloadController) DownloadOrder(c *gin.Context) {
	orderID := c.Param("order_id")

	var order model.Order
	if err := ac.DB.Select("order_id", "order_data").
		Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, order.OrderData)
}

func (ac *DownloadController) DownloadPrecheck(c *gin.Context) {
	orderID := c.Param("order_id")

	var order model.Order
	if err := ac.DB.Select("order_id", "precheck_passed", "precheck_result", "prechecked_at").
		Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	if order.PrecheckedAt == nil {
		handle.NotFoundError(c, "No precheck result available for this order")
		return
	}

	type Result struct {
		Passed    bool             `json:"passed"`
		Result    *json.RawMessage `json:"result"`
		CheckedAt string           `json:"checked_at"`
	}

	result := Result{
		Passed:    order.PrecheckPassed,
		Result:    order.PrecheckResult,
		CheckedAt: order.PrecheckedAt.Format("2006-01-02 15:04:05"),
	}

	handle.Success(c, result)
}
