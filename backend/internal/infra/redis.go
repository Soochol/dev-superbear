package infra

import (
	"github.com/hibiken/asynq"
)

// NewRedisClientOpt returns the asynq Redis connection option
// from the given address string (e.g. "localhost:6379").
func NewRedisClientOpt(addr, password string) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
	}
}
