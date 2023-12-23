package txmng

import (
	"time"
)

type Option func(cfg *Config)

type Config struct {
	forbidRawDB bool
	retries     *retries
}

type retries struct {
	count    int
	interval time.Duration
	need     func(error) bool
}

func WithForbidRawDB(v bool) Option {
	return func(cfg *Config) {
		cfg.forbidRawDB = v
	}
}

func WithRetries(
	retryCount int,
	retryInterval time.Duration,
	retryNeed func(error) bool,
) Option {
	return func(cfg *Config) {
		cfg.retries = &retries{
			count:    retryCount,
			interval: retryInterval,
			need:     retryNeed,
		}
	}
}

func WithDefaultRetries(
	retryCount int,
	retryInterval time.Duration,
) Option {
	return func(cfg *Config) {
		cfg.retries = &retries{
			count:    retryCount,
			interval: retryInterval,
			need:     isPGXSerializationError,
		}
	}
}
