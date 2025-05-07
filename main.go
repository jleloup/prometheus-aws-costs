package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	pacCostExplorer "prometheus-aws-costs/src/aws/costexplorer"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCostexplorer "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Config for auth creds
type Config struct {
	LogLevel       string        `mapstructure:"log_level"`
	MetricInterval time.Duration `mapstructure:"metric_interval"`
	MetricsPath    string        `mapstructure:"metrics_path"`
	MetricsPort    string        `mapstructure:"metrics_port"`
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

	// Start Prometheus server
	log.Info().
		Str("path", config.MetricsPath).
		Str("port", config.MetricsPort).
		Msg("Start Prometheus listener")

	go func() {
		http.Handle(config.MetricsPath, promhttp.Handler())
		log.Fatal().Err(http.ListenAndServe(":"+config.MetricsPort, nil)).Msg("Prometheus HTTP server failed")
	}()

	// Setup AWS client
	ctx := context.Background()
	sdkConfig, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return
	}
	ceClient := awsCostexplorer.NewFromConfig(sdkConfig)

	// Loop on AWS queries
	ticker := time.NewTicker(config.MetricInterval)
	for ; true; <-ticker.C {
		var wg sync.WaitGroup

		log.Debug().Msg("Refreshing AWS Cost explorer metrics")

		pacCostExplorer.LoadSavingPlans(*ceClient)

		wg.Wait()
	}
}
