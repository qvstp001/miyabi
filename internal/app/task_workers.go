package app

import (
	"log/slog"

	"github.com/ppxb/miyabi/internal/tasks"
)

func newTaskPools(service *tasks.Service, logger *slog.Logger) []*tasks.Pool {
	return []*tasks.Pool{
		// Directory traversal and subscription batches have independent workers.
		tasks.NewPool(service.Queue(), service, service.Registry(),
			[]tasks.Kind{tasks.KindScan}, 1, logger),
		// Batches retain serial submission and pacing without occupying the library worker.
		tasks.NewPool(service.Queue(), service, service.Registry(),
			[]tasks.Kind{tasks.KindSubscriptionBatch}, 1, logger),
	}
}
