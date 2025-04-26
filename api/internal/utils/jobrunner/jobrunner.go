package jobrunner

import (
	"context"
	"encoding/json"
	"log"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/utils/callr"
	"precisiondosing-api-go/internal/utils/precheck"
	"sync"
	"time"

	"gorm.io/gorm"
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
}

func New(config cfg.JobRunner, preckecker *precheck.PreCheck, callr *callr.CallR, jobDB *gorm.DB) *JobRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobRunner{
		cfg:        Config{fetchInterval: config.Interval, timeout: config.Timeout, workerPoolSize: config.MaxJobs},
		ctx:        ctx,
		cancel:     cancel,
		preckecker: preckecker,
		callr:      callr,
		jobDB:      jobDB,
	}
}

func (jr *JobRunner) Start() {
	_ = jr.purgeOnStart(jr.ctx)

	jr.jobs = make(chan *model.Order, jr.cfg.workerPoolSize*2) // Buffered channel (can tweak size)
	for i := 0; i < jr.cfg.workerPoolSize; i++ {
		jr.wg.Add(1)
		go jr.worker()
	}

	jr.wg.Add(1)
	go jr.run()
}

func (jr *JobRunner) Stop() {
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
			log.Printf("Fetched %d orders", len(orders))
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
	err := jr.jobDB.WithContext(ctx).
		Where("pre_checked IS FALSE").
		Limit(freeSlots).
		Find(&orders).Error

	if err != nil {
		log.Printf("Error fetching orders: %v", err)
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
	}

	// update order in db
	if saveErr := jr.jobDB.Save(order).Error; saveErr != nil {
		log.Printf("Error updating order: %v", saveErr)
		return
	}

	// if unrecoverable error or precheck is done call R
	if order.PreChecked {
		log.Println("Precheck done, starting order")
	}
}

func (jr *JobRunner) purgeOnStart(ctx context.Context) error {
	// on start purge orders that started but did not finish

	var orders []model.Order
	err := jr.jobDB.WithContext(ctx).
		Where("pre_checked IS TRUE AND started_at IS NOT NULL AND completed_at IS NULL").
		Find(&orders).Error

	if err != nil {
		log.Printf("Error fetching purge orders: %v", err)
		return nil
	}

	for _, order := range orders {
		order.PreChecked = false
		order.PrecheckError = nil
		order.Precheck = nil
		order.LastPrecheck = nil
		order.StartedAt = nil

		if err = jr.jobDB.Save(&order).Error; err != nil {
			log.Printf("Error purging order: %v", err)
			return nil
		}
	}

	return nil
}
