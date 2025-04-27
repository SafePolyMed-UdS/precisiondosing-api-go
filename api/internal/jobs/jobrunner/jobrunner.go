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
	now := time.Now()

	tx := jr.jobDB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Find jobs not yet started
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("pre_checked = FALSE AND started_at IS NULL").
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
		Update("started_at", now).Error

	if err != nil {
		jr.logger.Error("bulk updating started_at", log.Err(err))
		tx.Rollback()
		return nil
	}

	// 3. Also update the Go structs, so later Save() will not mess up
	for i := range orders {
		orders[i].StartedAt = &now
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
	_ = json.Unmarshal(order.Order, &patientData)

	now := time.Now()
	order.LastPrecheck = &now

	precheck, err := jr.preckecker.Check(&patientData)
	if err == nil {
		precheckByte, _ := json.Marshal(precheck)
		precheckRaw := json.RawMessage(precheckByte)
		order.PreChecked = true
		order.PrecheckError = nil
		order.Precheck = &precheckRaw
		order.StartedAt = &now
	} else {
		errMsg := err.Error()
		order.PrecheckError = &errMsg

		if !err.Recoverable {
			order.PreChecked = true
			order.StartedAt = &now
		}

		jr.logger.Error("precheck error",
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

	// if unrecoverable error or precheck is done call R
	if order.PreChecked {
		jr.logger.Info("running order", log.Str("orderID", order.OrderID))
		_, err := jr.callr.Adjust(order.ID, jr.cfg.timeout)
		if err != nil {
			jr.logger.Error("calling R", log.Str("orderID", order.OrderID), log.Err(err))
			return
		}
	}
}

func (jr *JobRunner) purgeOnStart(ctx context.Context) {
	// On start, reset orders that started but did not finish
	err := jr.jobDB.WithContext(ctx).
		Model(&model.Order{}).
		Where("pre_checked = TRUE AND started_at IS NOT NULL AND completed_at IS NULL").
		Updates(map[string]interface{}{
			"pre_checked":    false,
			"precheck_error": nil,
			"precheck":       nil,
			"last_precheck":  nil,
			"started_at":     nil,
		}).Error

	if err != nil {
		jr.logger.Error("purging incomplete orders", log.Err(err))
		return
	}

	jr.logger.Info("purged incomplete orders")
}
