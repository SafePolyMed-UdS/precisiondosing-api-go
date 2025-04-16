package responder

import (
	"precisiondosing-api-go/internal/model"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type ResultHandler struct {
	DB       *gorm.DB
	Endpoint string
}

func NewResultHandler(
	db *gorm.DB,
	endpoint string,
) *ResultHandler {
	return &ResultHandler{
		DB:       db,
		Endpoint: endpoint,
	}
}

func (h *ResultHandler) SendResults() {
	var orders []model.Order

	err := h.DB.
		Where("result_success IS NOT NULL").
		Where("sent_at IS NULL").
		Find(&orders).Error

	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch orders")
		return
	}

	for _, order := range orders {
		println("Sending order:", order.OrderID)
		// err := h.processOrder(order)
		// if err != nil {
		// 	fmt.Printf("❌ Error sending order %s: %v\n", order.OrderID, err)
		// 	continue
		// }
		continue

		// mark as sent
		// now := time.Now()
		// _, err = h.DB.Exec(`UPDATE orders SET sent_at = ? WHERE id = ?`, now, order.ID)
		// if err != nil {
		// 	log.Error().Err(err).Msgf("Failed to update order %s as sent", order.OrderID)
		// 	continue
		// }
	}
}
