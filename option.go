package txmng

import "time"

type config struct {
	retrier           Retrier
	dynamicFallbackDB bool
}
type Option func(cfg *config)

func WithRetrier(r Retrier) Option {
	return func(cfg *config) { cfg.retrier = r }
}

func WithRetrierParams(
	jitter float64, // [0; 1]
	delays ...time.Duration,
) Option {
	return func(cfg *config) {
		cfg.retrier = newDefaultRetrier(jitter, delays)
	}
}

func WithDefaultRetrier() Option {
	const fiftyPercentJitter = 0.5

	return func(cfg *config) {
		cfg.retrier = newDefaultRetrier(
			fiftyPercentJitter,
			[]time.Duration{
				100 * time.Millisecond,
				300 * time.Millisecond,
				600 * time.Millisecond,
			},
		)
	}
}

func WithDynamicFallbackDB() Option {
	return func(cfg *config) { cfg.dynamicFallbackDB = true }
}
