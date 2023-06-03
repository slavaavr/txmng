package txmng

type Option func(cfg *Config)

type Config struct {
	forbidRawDB bool
}

func WithForbidRawDB(v bool) Option {
	return func(cfg *Config) {
		cfg.forbidRawDB = v
	}
}
