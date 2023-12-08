package txmng

import "time"

type Option func(cfg *Config)

type Config struct {
	forbidRawDB bool
	retries     *retries
}

type retries struct {
	count    int
	interval time.Duration
}

func WithForbidRawDB(v bool) Option {
	return func(cfg *Config) {
		cfg.forbidRawDB = v
	}
}

func WithRetries(
	retriesCount int,
	retriesInterval time.Duration,
) Option {
	return func(cfg *Config) {
		cfg.retries = &retries{
			count:    retriesCount,
			interval: retriesInterval,
		}
	}
}
