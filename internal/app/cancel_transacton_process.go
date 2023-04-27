package app

import (
	"context"
	"github.com/mufasadev/enlabs-test/internal/config"
	"strconv"
	"time"
)

type CancelTransactionHandler interface {
	Execute(ctx context.Context) error
}

type CancelTransactionProcess struct {
	handler CancelTransactionHandler
	config  config.Process
}

func NewCancelTransactionProcess(h CancelTransactionHandler, cfg config.Process) *CancelTransactionProcess {
	return &CancelTransactionProcess{handler: h, config: cfg}
}

// Run runs the cancel transaction process.
func (p *CancelTransactionProcess) Run(ctx context.Context) error {
	timeout, cancel := context.WithTimeout(ctx, time.Duration(5)*time.Second)
	defer cancel()

	interval, err := strconv.Atoi(p.config.Interval)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.handler.Execute(timeout)
		}
	}
}
