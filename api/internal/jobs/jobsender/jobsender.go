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
	jobDB         *gorm.DB
	mmcAPI        *mmc.API

	logger log.Logger
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
		logger: log.WithComponent("jobsender"),
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

func (js *JobSender) processJobs(ctx context.Context) {
	var orders []model.Order
	if err := js.jobDB.WithContext(ctx).
		Where(&model.Order{Status: "processed"}).
		Order("COALESCE(send_tries, 0) ASC").
		Limit(js.batchSize).
		Find(&orders).Error; err != nil {
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
			order.Status = "error"
			errorMessage := "no result PDF created"
			order.ProcessErrorMessage = &errorMessage
			continue
		}

		pdfBytes, err := base64.StdEncoding.DecodeString(*order.ProcessResultPDF)
		if err != nil {
			js.logger.Error("decoding PDF", log.Str("orderID", order.OrderID), log.Err(err))
			order.Status = "error"
			errorMessage := "decoding PDF failed"
			order.ProcessErrorMessage = &errorMessage
			continue
		}

		order.SendTries++
		if err = js.mmcAPI.Send(pdfBytes, order.OrderID); err != nil {
			js.logger.Warn("sending PDF", log.Str("orderID", order.OrderID), log.Err(err))
			continue
		}
		js.logger.Info("sent PDF", log.Str("orderID", order.OrderID))

		now := time.Now()
		order.SentAt = &now
	}

	if err := js.jobDB.WithContext(ctx).Save(&orders).Error; err != nil {
		js.logger.Error("saving orders", log.Err(err))
		return
	}
}
