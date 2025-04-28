package jobrunner

import (
	"context"
	"encoding/json"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/precheck"
	"precisiondosing-api-go/internal/utils/callr"
	"precisiondosing-api-go/internal/utils/log"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Config struct {
	fetchInterval  time.Duration
	timeout        time.Duration
	workerPoolSize int
}

type JobRunner struct {
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	jobs   chan *model.Order

	callr      *callr.CallR
	preckecker *precheck.PreCheck
	jobDB      *gorm.DB

	logger log.Logger
}

func New(config cfg.JobRunnerConfig, preckecker *precheck.PreCheck, callr *callr.CallR, jobDB *gorm.DB) *JobRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobRunner{
		cfg:        Config{fetchInterval: config.Interval, timeout: config.Timeout, workerPoolSize: config.MaxJobs},
		ctx:        ctx,
		cancel:     cancel,
		preckecker: preckecker,
		callr:      callr,
		jobDB:      jobDB,
		logger:     log.WithComponent("jobrunner"),
	}
}

func (jr *JobRunner) Start() {
	jr.logger.Info("started")
	jr.purgeOnStart(jr.ctx)

	jr.jobs = make(chan *model.Order, jr.cfg.workerPoolSize*2) // Buffered channel (can tweak size)
	for i := 0; i < jr.cfg.workerPoolSize; i++ {
		jr.wg.Add(1)
		go jr.worker()
	}

	jr.wg.Add(1)
	go jr.run()
}

func (jr *JobRunner) Stop() {
	jr.logger.Info("stopped")

	jr.cancel()
	close(jr.jobs)
	jr.wg.Wait()
}

func (jr *JobRunner) run() {
	defer jr.wg.Done()
	ticker := time.NewTicker(jr.cfg.fetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-jr.ctx.Done():
			return
		case <-ticker.C:
			orders := jr.fetchJobs(jr.ctx)
			if len(orders) == 0 {
				continue
			}

			jr.logger.Info("fetched orders", log.Int("count", len(orders)))
			for i := range orders {
				select {
				case jr.jobs <- &orders[i]:
				case <-jr.ctx.Done():
					return
				}
			}
		}
	}
}

func (jr *JobRunner) fetchJobs(ctx context.Context) []model.Order {
	freeSlots := cap(jr.jobs) - len(jr.jobs)
	if freeSlots <= 0 {
		return nil
	}

	var orders []model.Order
	tx := jr.jobDB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Find jobs not yet started
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where(&model.Order{Status: "queued"}).
		Limit(freeSlots).
		Find(&orders).Error

	if err != nil {
		jr.logger.Error("fetching orders with lock", log.Err(err))
		tx.Rollback()
		return nil
	}

	if len(orders) == 0 {
		tx.Rollback()
		return nil
	}

	// 2. Bulk update started_at in the database
	ids := make([]uint, len(orders))
	for i, order := range orders {
		ids[i] = order.ID
	}

	err = tx.Model(&model.Order{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{"status": "staged"}).Error

	if err != nil {
		jr.logger.Error("bulk updating to staged", log.Err(err))
		tx.Rollback()
		return nil
	}

	// 3. Also update the Go structs, so later Save() will not mess up
	for i := range orders {
		orders[i].Status = "staged"
	}

	if err = tx.Commit().Error; err != nil {
		jr.logger.Error("committing transaction", log.Err(err))
		return nil
	}

	return orders
}

func (jr *JobRunner) worker() {
	defer jr.wg.Done()
	for {
		select {
		case <-jr.ctx.Done():
			return
		case order, ok := <-jr.jobs:
			if !ok {
				return
			}
			jr.processJob(order)
		}
	}
}

func (jr *JobRunner) processJob(order *model.Order) {
	patientData := model.PatientData{}
	_ = json.Unmarshal(order.OrderData, &patientData)

	now := time.Now()
	order.PrecheckedAt = &now

	precheck, err := jr.preckecker.Check(&patientData)
	precheckByte, _ := json.Marshal(precheck)
	precheckRaw := json.RawMessage(precheckByte)
	order.PrecheckResult = &precheckRaw

	if err == nil {
		// precheck passed
		order.PrecheckPassed = true
	} else {
		order.PrecheckPassed = false
		if err.Recoverable {
			order.Status = "queued"
		}

		jr.logger.Warn("precheck error",
			log.Str("orderID", order.OrderID),
			log.Bool("recoverable", err.Recoverable),
			log.Err(err),
		)
	}

	// update order in db
	if saveErr := jr.jobDB.Save(order).Error; saveErr != nil {
		jr.logger.Error("updating order", log.Str("orderID", order.OrderID), log.Err(saveErr))
		return
	}

	// if precheck failed and is recoverable, return
	if order.Status == "queued" {
		jr.logger.Info("order precheck failed, re-queued", log.Str("orderID", order.OrderID))
		return
	}

	// run order
	jr.logger.Info("running order", log.Str("orderID", order.OrderID))
	order.Status = "processing"
	if saveErr := jr.jobDB.Save(order).Error; saveErr != nil {
		jr.logger.Error("updating order", log.Str("orderID", order.OrderID), log.Err(saveErr))
		return
	}

	adjust := order.PrecheckPassed
	errMsg := precheck.Message
	resp, adjustErr := jr.callr.Adjust(order.ID, adjust, errMsg, jr.cfg.timeout)
	order.ProcessedAt = &now

	if adjustErr != nil {
		jr.logger.Error("calling R", log.Str("orderID", order.OrderID), log.Err(err))
		order.Status = "error"
		adjErrMsg := adjustErr.Error()
		order.ProcessErrorMessage = &adjErrMsg
	} else {
		order.Status = "processed"
		order.DoseAdjusted = resp.DoseAdjusted
	}

	if saveErr := jr.jobDB.Save(order).Error; saveErr != nil {
		jr.logger.Error("updating order", log.Str("orderID", order.OrderID), log.Err(saveErr))
		return
	}
}

func (jr *JobRunner) purgeOnStart(ctx context.Context) {
	// On start, reset orders that started but did not finish
	err := jr.jobDB.WithContext(ctx).
		Model(&model.Order{}).
		Where(&model.Order{Status: "staged"}).
		Or(&model.Order{Status: "prechecked"}).
		Or(&model.Order{Status: "processing"}).
		Updates(map[string]interface{}{
			"status":                "queued",
			"precheck_result":       nil,
			"precheck_passed":       false,
			"prechecked_at":         nil,
			"process_result_PDF":    nil,
			"process_error_message": nil,
			"processed_at":          nil,
		}).Error

	if err != nil {
		jr.logger.Error("purging incomplete orders", log.Err(err))
		return
	}

	jr.logger.Info("purged incomplete orders")
}
