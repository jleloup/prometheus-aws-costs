package main

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Config for auth creds
type Config struct {
	LogLevel       string        `mapstructure:"log_level"`
	MetricInterval time.Duration `mapstructure:"metric_interval"`
	MetricsPort    string        `mapstructure:"metrics_port"`
}

// Deal with configuration using environment variables,
// default values and CLI
func LoadConfig() (config Config, err error) {
	v := viper.New()

	// Load config
	log.Debug().Msg("Loading environment variables...")
	v.SetDefault("metric_interval", "30s")
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

func main() {

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Cannot load config")
	}

	// Setup logging
	level, err := zerolog.ParseLevel(config.LogLevel)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Cannot load log level")
	}
	zerolog.SetGlobalLevel(level)

	// Core functionality
	fmt.Println("Hello, World!")
}
