package service

import (
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/dev-superbear/nexus-backend/internal/worker"
)

// MetricsService collects monitoring metrics from asynq queues.
type MetricsService struct {
	inspector *asynq.Inspector
}

func NewMetricsService(redisOpt asynq.RedisClientOpt) *MetricsService {
	return &MetricsService{
		inspector: asynq.NewInspector(redisOpt),
	}
}

func (s *MetricsService) CollectMetrics() (*worker.MonitoringMetrics, error) {
	agentInfo, err := s.inspector.GetQueueInfo("default")
	if err != nil {
		return nil, fmt.Errorf("get queue info: %w", err)
	}

	m := &worker.MonitoringMetrics{}
	m.QueueDepth.Agent = agentInfo.Pending + agentInfo.Active
	return m, nil
}
