package jobsender

import (
	"context"
	"encoding/base64"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/services/mmc"
	"precisiondosing-api-go/internal/utils/log"
	"sync"
	"time"

	"gorm.io/gorm"
)

type JobSender struct {
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	fetchInterval time.Duration
	batchSize     int
	MaxRetries    int
	jobDB         *gorm.DB
	mmcAPI        *mmc.API

	logger log.Logger
}

func New(mmcConfig cfg.MMCConfig, jobDB *gorm.DB) *JobSender {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobSender{
		fetchInterval: mmcConfig.Interval,
		batchSize:     mmcConfig.BatchSize,
		MaxRetries:    mmcConfig.MaxRetries,
		mmcAPI:        mmc.NewAPI(mmcConfig),
		ctx:           ctx,
		cancel:        cancel,
		jobDB:         jobDB,
		logger:        log.WithComponent("jobsender"),
	}
}

func (js *JobSender) Start() {
	js.logger.Info("started")

	js.wg.Add(1)
	go js.run()
}

func (js *JobSender) Stop() {
	js.logger.Info("stopped")

	js.cancel()
	js.wg.Wait()
}

func (js *JobSender) run() {
	defer js.wg.Done()
	ticker := time.NewTicker(js.fetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-js.ctx.Done():
			return
		case <-ticker.C:
			js.processJobs(js.ctx)
		}
	}
}

func calculateBackoff(sendTries int) time.Duration {
	switch sendTries {
	case 1:
		return 1 * time.Minute
	case 2:
		return 2 * time.Minute
	case 3:
		return 5 * time.Minute
	case 4:
		return 15 * time.Minute
	case 5:
		return 30 * time.Minute
	default:
		return 1 * time.Hour // Cap backoff at 1 hour
	}
}

func (js *JobSender) processJobs(ctx context.Context) {
	var orders []model.Order
	now := time.Now()

	// Fetch only orders that are:
	// - In the "processed" status
	// - With less than MaxSendTries attempts (or nil, which is treated as 0)
	// - And either without a NextSendAttemptAt or that time is due
	err := js.jobDB.WithContext(ctx).
		Where(
			"status = ? AND (send_tries < ? OR send_tries IS NULL) "+
				"AND (next_send_attempt_at IS NULL OR next_send_attempt_at <= ?)",
			model.StatusProcessed, js.MaxRetries, now,
		).
		Order("COALESCE(send_tries, 0) ASC").
		Limit(js.batchSize).
		Find(&orders).Error
	if err != nil {
		js.logger.Error("fetching orders", log.Err(err))
		return
	}

	if len(orders) == 0 {
		return
	}

	js.logger.Info("fetched orders", log.Int("count", len(orders)))
	for i := range orders {
		order := &orders[i]

		if order.ProcessResultPDF == nil {
			js.logger.Error("order has no result PDF", log.Str("orderID", order.OrderID))

			order.Status = model.StatusError
			errorMessage := "no result PDF created"
			order.ProcessErrorMessage = &errorMessage
			continue
		}

		// Try to decode PDF
		pdfBytes, decodeErr := base64.StdEncoding.DecodeString(*order.ProcessResultPDF)
		if err != nil {
			js.logger.Error("decoding PDF", log.Str("orderID", order.OrderID), log.Err(decodeErr))

			order.Status = model.StatusError
			errorMessage := "decoding PDF failed"
			order.ProcessErrorMessage = &errorMessage
			continue
		}

		// Try sending
		order.SendTries++
		order.LastSendAttemptAt = &now

		err = js.mmcAPI.Send(pdfBytes, order.OrderID)
		if err != nil {
			js.logger.Warn("sending PDF failed", log.Str("orderID", order.OrderID), log.Err(err))

			// Save error message
			errMsg := err.Error()
			order.LastSendError = &errMsg

			// Set next send attempt time
			backoff := calculateBackoff(order.SendTries)
			nextRetry := now.Add(backoff)
			order.NextSendAttemptAt = &nextRetry

			// If too many tries -> give up
			if order.SendTries >= js.MaxRetries {
				order.Status = model.StatusSendFailed
				errorMessage := "maximum send tries exceeded"
				order.ProcessErrorMessage = &errorMessage

				js.logger.Error("max send tries exceeded", log.Str("orderID", order.OrderID))
			}

			continue
		}

		// If sending successful
		order.Status = model.StatusSent
		order.SentAt = &now
		order.LastSendError = nil // Clear last error
	}

	// Save all updated orders
	if err = js.jobDB.WithContext(ctx).Save(&orders).Error; err != nil {
		js.logger.Error("saving orders", log.Err(err))
		return
	}

	js.logger.Info("successfully processed batch", log.Int("count", len(orders)))
}
