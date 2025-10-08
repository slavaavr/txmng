package txmng

import "time"

type Option func(cfg *Config)

type Config struct {
	retrier Retrier
}

func WithRetrier(r Retrier) Option {
	return func(cfg *Config) { cfg.retrier = r }
}

func WithRetrierParams(
	delays []time.Duration,
	jitter float64, // [0; 1]
) Option {
	return func(cfg *Config) {
		cfg.retrier = newDefaultRetrier(delays, jitter)
	}
}

func WithDefaultRetrier() Option {
	return func(cfg *Config) {
		cfg.retrier = newDefaultRetrier(
			[]time.Duration{
				100 * time.Millisecond,
				300 * time.Millisecond,
				600 * time.Millisecond,
			},
			0.5,
		)
	}
}
