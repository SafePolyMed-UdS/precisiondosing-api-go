package jobrunner

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Config struct {
	Interval   time.Duration // Time between fetches
	Timeout    time.Duration // Timeout for each job
	WorkerPool int           // Number of concurrent workers
	JobDB      *gorm.DB      // Database connection for job management
}

type JobRunner struct {
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(cfg Config) *JobRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobRunner{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (jr *JobRunner) Start() {
	jr.wg.Add(1)
	go jr.run()
}

func (jr *JobRunner) Stop() {
	jr.cancel()
	jr.wg.Wait()
}

func (jr *JobRunner) run() {
	defer jr.wg.Done()
	ticker := time.NewTicker(jr.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-jr.ctx.Done():
			return
		case <-ticker.C:
			log.Info().Msg("JobRunner tick")
		}
	}
}
