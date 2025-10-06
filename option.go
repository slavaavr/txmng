package txmng

import "time"

type Option func(cfg *Config)

type Config struct {
	retrier Retrier
}

func WithRetrier(r Retrier) Option {
	return func(cfg *Config) { cfg.retrier = r }
}

func WithDefaultRetrier() Option {
	return func(cfg *Config) {
		cfg.retrier = newDefaultRetrier(
			[]time.Duration{
				100 * time.Millisecond,
				300 * time.Millisecond,
				600 * time.Millisecond,
			},
		)
	}
}

func WithDefaultRetrierDelays(delays []time.Duration) Option {
	return func(cfg *Config) {
		cfg.retrier = newDefaultRetrier(delays)
	}
}
