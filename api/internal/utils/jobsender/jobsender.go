package jobsender

import (
	"context"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/utils/resultsender"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type JobSender struct {
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	fetchInterval time.Duration
	jobDB         *gorm.DB
	sender        *resultsender.Sender
}

func New(senderConfig cfg.SendRunner, jobDB *gorm.DB) *JobSender {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobSender{
		fetchInterval: senderConfig.Interval,
		sender: resultsender.New(
			senderConfig.Login,
			senderConfig.Password,
			senderConfig.AuthEndpoint,
			senderConfig.SendEndpoint,
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
			js.processJobs(js.ctx)
		}
	}
}

func (js *JobSender) processJobs(ctx context.Context) {
	var orders []model.Order
	err := js.jobDB.WithContext(ctx).
		Where("completed_at IS NOT NULL AND sent_at is NULL").
		Limit(2).
		Find(&orders).Error

	if err != nil {
		// Handle error (e.g., log it)
		return
	}

	log.Printf("Found %d orders to send", len(orders))
}
