package tgbot

import (
	"context"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type DispatchQueueConfig struct {
	Workers   int
	QueueSize int
}

type DispatchQueue struct {
	queue          chan tgbotapi.Update
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	workers        int
	processHandler func(tgbotapi.Update)
}

func NewDispatchQueue(workers int, size int) *DispatchQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &DispatchQueue{
		queue:   make(chan tgbotapi.Update, size),
		ctx:     ctx,
		cancel:  cancel,
		workers: workers,
	}
}

func (dq *DispatchQueue) Start() {
	for i := 0; i < dq.workers; i++ {
		dq.wg.Add(1)
		go func() {
			defer dq.wg.Done()
			for {
				select {
				case <-dq.ctx.Done():
					return
				case update, ok := <-dq.queue:
					if !ok {
						return
					}
					if dq.processHandler != nil {
						dq.processHandler(update)
					}
				}
			}
		}()
	}
}

func (dq *DispatchQueue) Stop() {
	dq.cancel()
	close(dq.queue)
	dq.wg.Wait()
}

func (dq *DispatchQueue) Enqueue(update tgbotapi.Update) {
	select {
	case <-dq.ctx.Done():
		return
	case dq.queue <- update:
	}
}

func (dq *DispatchQueue) SetProcessHandler(handler func(tgbotapi.Update)) {
	dq.processHandler = handler
}
