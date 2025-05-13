package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	selfAws "prometheus-aws-costs/src/aws"
	selfConfig "prometheus-aws-costs/src/utils/config"
	selfOtel "prometheus-aws-costs/src/utils/otel"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCostexplorer "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	ctx := context.Background()

	// Load configuration
	config, err := selfConfig.LoadConfig()
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

	// Setup telemetry
	telemetry, err := selfOtel.NewTelemetry(ctx, config)
	if err != nil {
		log.Fatal().Msg("failed to load telemetry config")
		os.Exit(1)
	}
	defer telemetry.Shutdown(ctx)

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
	sdkConfig, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return
	}
	ceClient := awsCostexplorer.NewFromConfig(sdkConfig)

	ceFetcher := selfAws.NewCEFetcher(ctx, *ceClient, telemetry)

	// Loop on AWS queries
	ticker := time.NewTicker(config.MetricInterval)
	for ; true; <-ticker.C {
		var wg sync.WaitGroup
		_, span := telemetry.TraceStart(ctx, "refresh-metrics")
		defer span.End()

		log.Debug().Msg("Refreshing AWS Cost explorer metrics")

		go ceFetcher.GetSavingPlansCoverageMetrics(ctx, &wg)
		go ceFetcher.GetSavingPlansUtilizationMetrics(ctx, &wg)

		wg.Wait()
	}
}
