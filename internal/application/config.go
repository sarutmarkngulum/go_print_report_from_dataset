package application

import "time"

type Config struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	MaxInputBytes   int64
}

func DefaultConfig() Config {
	return Config{
		Addr:            ":8080",
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		MaxInputBytes:   20 << 20,
	}
}
