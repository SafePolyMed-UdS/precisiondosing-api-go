package jobsender

import (
	"context"
	"encoding/base64"
	"fmt"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/utils/mmc"
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
	jobDB         *gorm.DB
	mmcAPI        *mmc.API
}

func New(mmcConfig cfg.MMCConfig, jobDB *gorm.DB) *JobSender {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobSender{
		fetchInterval: mmcConfig.Interval,
		batchSize:     mmcConfig.BatchSize,
		mmcAPI: mmc.NewAPI(
			mmcConfig.Login,
			mmcConfig.Password,
			mmcConfig.AuthEndpoint,
			mmcConfig.ResultEndpoint,
			mmcConfig.PDFPrefix,
			mmcConfig.ExpiryThreshold,
		),
		ctx:    ctx,
		cancel: cancel,
		jobDB:  jobDB,
	}
}

func (js *JobSender) Start() {
	js.wg.Add(1)
	go js.run()
}

func (js *JobSender) Stop() {
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
			if err := js.processJobs(js.ctx); err != nil {
				// TODO: log error
			}
		}
	}
}

func (js *JobSender) processJobs(ctx context.Context) error {
	var orders []model.Order
	if err := js.jobDB.WithContext(ctx).
		Where("completed_at IS NOT NULL AND sent_at IS NULL").
		Order("COALESCE(sent_trys, 0) ASC").
		Limit(js.batchSize).
		Find(&orders).Error; err != nil {
		return fmt.Errorf("failed to fetch orders: %w", err)
	}

	for _, order := range orders {
		if order.ResultPDF == nil {
			// TODO: This cannot happen -> log error
			continue
		}

		pdfBytes, err := base64.StdEncoding.DecodeString(*order.ResultPDF)
		if err != nil {
			// TODO: log error
			continue
		}

		// TODO: increment sent_trys
		if err := js.mmcAPI.Send(pdfBytes, order.OrderID); err != nil {
			// TODO: log error
			continue
		}

		// TODO: update order as sent (sent_at to now)
		// log error -> do not return error
	}

	return nil
}
