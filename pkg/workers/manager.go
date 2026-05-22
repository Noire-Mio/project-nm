package workers

import (
	"context"
	"log"
)

type IWorker interface {
	Start(ctx context.Context) 
}

type WorkerManager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	workers []IWorker
}

func NewWorkerManager(globalCtx context.Context) *WorkerManager {
	ctx, cancel := context.WithCancel(globalCtx)
	return &WorkerManager{
		ctx:     ctx,
		cancel:  cancel,
		workers: make([]IWorker, 0),
	}
}

// Register 
func (m *WorkerManager) Register(worker IWorker) {
	m.workers = append(m.workers, worker)
}

// StartAll 
func (m *WorkerManager) StartAll() {
	for _, worker := range m.workers {
		go worker.Start(m.ctx)
	}
}

// StopAll 停機
func (m *WorkerManager) StopAll() {
	log.Println("[Worker Manager] 正在發送停機訊號...")
	m.cancel() 
}
