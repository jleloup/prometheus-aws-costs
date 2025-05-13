package config

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	LogLevel         string        `mapstructure:"log_level"`
	MetricInterval   time.Duration `mapstructure:"metric_interval"`
	MetricsPath      string        `mapstructure:"metrics_path"`
	MetricsPort      string        `mapstructure:"metrics_port"`
	ServiceName      string        `mapstructure:"service_name"`
	ServiceVersion   string        `mapstructure:"service_name"`
	TelemetryEnabled string        `mapstructure:"service_name"`
}

// Deal with configuration using environment variables,
// default values and CLI
func LoadConfig() (config Config, err error) {
	v := viper.New()

	// Load config
	log.Debug().Msg("Loading environment variables...")
	v.SetDefault("metric_interval", "30m")
	v.SetDefault("metrics_path", "/metrics")
	v.SetDefault("metrics_port", "11223")
	v.SetDefault("log_level", "info")
	v.AutomaticEnv()

	err = v.Unmarshal(&config)

	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Unable to unmarshal environment variables")
	}

	// Check configuration
	if len(config.MetricsPort) == 0 {
		log.Fatal().Msg("Unable to find Metrics Port")
	}

	return
}
