package ordercontroller

import (
	"errors"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/utils/log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderController struct {
	DB     *gorm.DB
	logger log.Logger
}

func New(resourceHandle *handle.ResourceHandle) *OrderController {
	return &OrderController{
		DB:     resourceHandle.Databases.GormDB,
		logger: log.WithComponent("ordercontroller"),
	}
}

type orderOverview struct {
	OrderID             string     `json:"order_id"`
	User                string     `json:"user"`
	DoseAdjusted        bool       `json:"dose_adjusted"`
	PrecheckPassed      bool       `json:"precheck_passed"`
	PrecheckError       *string    `json:"precheck_error,omitempty"`
	ProcessErrorMessage *string    `json:"process_error,omitempty"`
	LastSendError       *string    `json:"last_send_error,omitempty"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	ProcessedAt         *time.Time `json:"processed_at,omitempty"`
	ProcessingDuration  *string    `json:"processing_duration,omitempty"`
	SentAt              *time.Time `json:"sent_at,omitempty"`
}

func (oc *OrderController) GetOrders(c *gin.Context) {
	var orders []model.Order
	query := oc.DB.Preload("User")

	// Optional filters
	status := c.Query("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	owner := c.Query("user")
	if owner != "" {
		if id, err := strconv.Atoi(owner); err == nil {
			query = query.Where("user_id = ?", id)
		} else {
			query = query.Joins("JOIN users ON users.id = orders.user_id").
				Where("users.email = ?", owner)
		}
	}

	if err := query.
		Omit("order_data", "precheck_result", "process_result_pdf").
		Order("created_at desc").Find(&orders).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	var response []orderOverview
	for _, o := range orders {
		response = append(response, orderOverview{
			OrderID:             o.OrderID,
			User:                o.User.Email,
			DoseAdjusted:        o.DoseAdjusted,
			PrecheckPassed:      o.PrecheckPassed,
			ProcessErrorMessage: o.ProcessErrorMessage,
			LastSendError:       o.LastSendError,
			Status:              o.Status,
			CreatedAt:           o.CreatedAt,
			ProcessedAt:         o.ProcessedAt,
			ProcessingDuration:  o.ProcessingDuration,
			SentAt:              o.SentAt,
		})
	}

	if len(response) == 0 {
		handle.NotFoundError(c, "No orders found that match the query")
		return
	}

	handle.Success(c, response)
}

func (oc *OrderController) GetOrderByID(c *gin.Context) {
	orderID := c.Param("order_id")
	var order model.Order

	// You could add validation here to make sure orderID is an integer if needed
	query := oc.DB.Preload("User")

	if err := query.
		Omit("order_data", "precheck_result", "process_result_pdf").
		Where("order_id = ?", orderID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
		} else {
			handle.ServerError(c, err)
		}
		return
	}

	response := orderOverview{
		OrderID:             order.OrderID,
		User:                order.User.Email,
		DoseAdjusted:        order.DoseAdjusted,
		PrecheckPassed:      order.PrecheckPassed,
		ProcessErrorMessage: order.ProcessErrorMessage,
		LastSendError:       order.LastSendError,
		Status:              order.Status,
		CreatedAt:           order.CreatedAt,
		ProcessedAt:         order.ProcessedAt,
		SentAt:              order.SentAt,
	}

	handle.Success(c, response)
}

func (oc *OrderController) ResetFailedSends(c *gin.Context) {
	result := oc.DB.Model(&model.Order{}).
		Where("status = ?", model.StatusSendFailed).
		Updates(map[string]interface{}{
			"status":               model.StatusProcessed,
			"last_send_error":      nil,
			"last_send_attempt_at": nil,
			"next_send_attempt_at": nil,
			"sent_at":              nil,
			"send_tries":           0,
		})

	if result.Error != nil {
		handle.ServerError(c, result.Error)
		return
	}

	orderAffected := result.RowsAffected

	oc.logger.Info("Requeue orders for sending", log.Int("orders", int(orderAffected)))
	handle.Success(c, gin.H{
		"message": "Orders with failed sends resetted",
		"orders":  orderAffected,
	})
}

func (oc *OrderController) ResendOrder(c *gin.Context) {
	orderID := c.Param("order_id")

	var order model.Order
	if err := oc.DB.
		Select("id", "order_id", "status").
		Where("order_id = ?", orderID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	if order.Status != model.StatusSendFailed &&
		order.Status != model.StatusSent {
		handle.BadRequestError(c, "Order must be in send_failed or sent state")
		return
	}

	if err := oc.DB.Model(&order).Updates(map[string]interface{}{
		"status":               model.StatusProcessed,
		"last_send_error":      nil,
		"last_send_attempt_at": nil,
		"next_send_attempt_at": nil,
		"sent_at":              nil,
		"send_tries":           0,
	}).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	oc.logger.Info("Requeue order for sending", log.Str("orderID", order.OrderID))
	handle.Success(c, gin.H{
		"message": "Order reset to processed state",
		"orderId": order.OrderID,
	})
}

func (oc *OrderController) RequeueOrderByID(c *gin.Context) {
	orderID := c.Param("order_id")

	var status string
	if err := oc.DB.
		Model(&model.Order{}).
		Select("status").
		Where("order_id = ?", orderID).
		First(&status).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	if status == model.StatusProcessing {
		handle.BadRequestError(c, "Order is in processing state")
		return
	}

	if err := oc.DB.Model(&model.Order{}).
		Where("order_id = ?", orderID).
		Updates(map[string]interface{}{
			"precheck_result":       nil,
			"precheck_passed":       false,
			"prechecked_at":         nil,
			"process_result_pdf":    nil,
			"dose_adjusted":         false,
			"process_error_message": nil,
			"processed_at":          nil,
			"sent_at":               nil,
			"send_tries":            0,
			"last_send_attempt_at":  nil,
			"last_send_error":       nil,
			"next_send_attempt_at":  nil,
			"status":                model.StatusQueued,
		}).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	oc.logger.Info("Requeue order for processing", log.Str("orderID", orderID))
	handle.Success(c, gin.H{
		"message": "Order requeued",
		"orderId": orderID,
	})
}

func (oc *OrderController) RequeueErrorOrders(c *gin.Context) {
	res := oc.DB.
		Model(&model.Order{}).
		Where("status = ?", model.StatusError).
		Updates(map[string]interface{}{
			"precheck_result":       nil,
			"precheck_passed":       false,
			"prechecked_at":         nil,
			"process_result_pdf":    nil,
			"dose_adjusted":         false,
			"process_error_message": nil,
			"processed_at":          nil,
			"sent_at":               nil,
			"send_tries":            0,
			"last_send_attempt_at":  nil,
			"last_send_error":       nil,
			"next_send_attempt_at":  nil,
			"status":                model.StatusQueued,
		})

	if res.Error != nil {
		handle.ServerError(c, res.Error)
		return
	}

	ordersAffected := res.RowsAffected

	oc.logger.Info("Requeue orders with error status", log.Int("orders", int(ordersAffected)))
	handle.Success(c, gin.H{
		"message":       "Requeued orders with error status",
		"rows_affected": ordersAffected,
	})
}

func (oc *OrderController) DeleteOrderByID(c *gin.Context) {
	orderID := c.Param("order_id")

	if err := oc.DB.
		Where("order_id = ?", orderID).
		Delete(&model.Order{}).
		Limit(1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	oc.logger.Info("Order deleted", log.Str("orderID", orderID))
	handle.Success(c, gin.H{
		"message": "Order deleted",
	})
}
