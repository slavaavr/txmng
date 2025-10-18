package txmng

import "time"

type config struct {
	retrier Retrier
}
type Option func(cfg *config)

func WithRetrier(r Retrier) Option {
	return func(cfg *config) { cfg.retrier = r }
}

func WithRetrierParams(
	delays []time.Duration,
	jitter float64, // [0; 1]
) Option {
	return func(cfg *config) {
		cfg.retrier = newDefaultRetrier(delays, jitter)
	}
}

func WithDefaultRetrier() Option {
	const fiftyPercentJitter = 0.5

	return func(cfg *config) {
		cfg.retrier = newDefaultRetrier(
			[]time.Duration{
				100 * time.Millisecond,
				300 * time.Millisecond,
				600 * time.Millisecond,
			},
			fiftyPercentJitter,
		)
	}
}
