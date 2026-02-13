package tgbot

import (
	"context"
	"sync"
)

type DispatchQueueConfig struct {
	Workers   int
	QueueSize int
}

type DispatchQueue struct {
	queue          chan *Update
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	workers        int
	processHandler func(*Update)
}

func newDispatchQueue(workers int, size int) *DispatchQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &DispatchQueue{
		queue:   make(chan *Update, size),
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
				case upd, ok := <-dq.queue:
					if !ok {
						return
					}
					if dq.processHandler != nil && upd != nil {
						dq.processHandler(upd)
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

func (dq *DispatchQueue) Enqueue(update *Update) {
	select {
	case <-dq.ctx.Done():
		return
	case dq.queue <- update:
	}
}

func (dq *DispatchQueue) SetProcessHandler(handler func(*Update)) {
	dq.processHandler = handler
}
