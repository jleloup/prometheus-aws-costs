package costexplorer

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	selfOtel "prometheus-aws-costs/src/utils/otel"

	awsCostexplorer "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/codes"
)

var (
	savingPlansUtilizationTotalCommitment = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pac_savingplans_utilization_total_commitment_dollar",
		Help: "Number of dollars purchased as saving plans.",
	})
	savingPlansUtilizationUnusedCommitment = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pac_savingplans_utilization_unused_commitment_dollar",
		Help: "Number of dollars unused by saving plans.",
	})
	savingPlansUtilizationUsedCommitment = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pac_savingplans_utilization_used_commitment_dollar",
		Help: "Number of dollars used by saving plans.",
	})
	savingPlansUtilizationPercentage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pac_savingplans_utilization_percent",
		Help: "Percentage of saving plans utilization.",
	})
	costExplorerAPICalls = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pac_costexplorer_api_calls_total",
		Help: "Number of calls to the AWS Cost Explorer API",
	})
)

type CostExplorerFetcher struct {
	ctx       context.Context
	client    awsCostexplorer.Client
	telemetry *selfOtel.Telemetry
}

func NewCEFetcher(context context.Context, client awsCostexplorer.Client, telemetry *selfOtel.Telemetry) *CostExplorerFetcher {

	return &CostExplorerFetcher{
		ctx:       context,
		client:    client,
		telemetry: telemetry,
	}
}

func (e *CostExplorerFetcher) GetSavingPlansCoverageMetrics(ctx context.Context, wg *sync.WaitGroup) {

	log.Debug().Msg("Get Saving Plans coverage metrics")

	wg.Add(1)
	_, span := e.telemetry.TraceStart(ctx, "get-costexplorer-savingplans-coverage-metrics")
	defer span.End()
	defer wg.Done()

	// Time range of 1 day from two days ago
	end := time.Now().Add(-time.Hour * 48)
	endStr := end.Format(time.DateOnly)
	start := end.Add(-time.Hour * 24)
	startStr := start.Format(time.DateOnly)

	maxResult := int32(20)
	granularity := types.GranularityDaily

	instance_family := "INSTANCE_TYPE_FAMILY"
	region := "REGION"
	service := "SERVICE"

	groupBy := []types.GroupDefinition{
		{Key: &instance_family, Type: types.GroupDefinitionTypeTag},
		{Key: &region, Type: types.GroupDefinitionTypeTag},
		{Key: &service, Type: types.GroupDefinitionTypeTag},
	}
	metrics := []string{"SpendCoveredBySavingsPlans"}

	output, err := e.client.GetSavingsPlansCoverage(ctx, &awsCostexplorer.GetSavingsPlansCoverageInput{
		TimePeriod: &types.DateInterval{
			Start: &startStr,
			End:   &endStr,
		},
		MaxResults:  &maxResult,
		Granularity: granularity,
		GroupBy:     groupBy,
		Metrics:     metrics,
	})

	if err != nil {

		var dataEx *types.DataUnavailableException

		if errors.As(err, &dataEx) {
			log.Info().
				Err(err).
				Msg("No Saving Plans coverage found")
		} else {
			log.Error().
				Err(err).
				Msg("Cannot fetch Saving Plans coverage")
		}
	} else {
		log.Debug().Interface("dict", output).Msg("Saving Plans coverage output")

		// for _, coverage := range output.SavingsPlansCoverages {
		// 	coverage.
		// 	savingPlansCoverageTotalCommitment.Set(coverage.Total.Utilization.TotalCommitment)
		// }
	}

	costExplorerAPICalls.Inc()
}

func (e *CostExplorerFetcher) GetSavingPlansUtilizationMetrics(ctx context.Context, wg *sync.WaitGroup) {

	log.Debug().Msg("Get Saving Plans utilization metrics")

	wg.Add(1)
	_, span := e.telemetry.TraceStart(ctx, "get-costexplorer-savingplans-utilization-metrics")
	defer span.End()
	defer wg.Done()

	// Time range of 1 day from now
	end := time.Now().Add(-time.Hour * 48)
	endStr := end.Format(time.DateOnly)
	start := end.Add(-time.Hour * 24)
	startStr := start.Format(time.DateOnly)

	granularity := types.GranularityDaily

	output, err := e.client.GetSavingsPlansUtilization(ctx, &awsCostexplorer.GetSavingsPlansUtilizationInput{
		TimePeriod: &types.DateInterval{
			Start: &startStr,
			End:   &endStr,
		},
		Granularity: granularity,
	})

	if err != nil {

		var dataEx *types.DataUnavailableException

		if errors.As(err, &dataEx) {
			log.Info().
				Err(err).
				Msg("No Saving Plans utilization found")
		} else {
			log.Error().
				Err(err).
				Msg("Cannot fetch Saving Plans utilization")
		}
	} else {
		log.Debug().Interface("dict", output).Msg("Saving Plans utilization output")

		// Extract metrics from output
		totalCommitment, err := strconv.ParseFloat(*output.Total.Utilization.TotalCommitment, 64)
		if err != nil {
			log.Warn().Str("totalCommitment", *output.Total.Utilization.TotalCommitment).Msg("Cannot convert to float")
			span.SetStatus(codes.Error, "Error while computing TotalCommitment")
		} else {
			savingPlansUtilizationTotalCommitment.Set(totalCommitment)
		}

		unusedCommitment, err := strconv.ParseFloat(*output.Total.Utilization.UnusedCommitment, 64)
		if err != nil {
			log.Warn().Str("unusedCommitment", *output.Total.Utilization.UnusedCommitment).Msg("Cannot convert to float")
			span.SetStatus(codes.Error, "Error while computing UnusedCommitment")
		} else {
			savingPlansUtilizationUnusedCommitment.Set(unusedCommitment)
		}

		usedCommitment, err := strconv.ParseFloat(*output.Total.Utilization.UsedCommitment, 64)
		if err != nil {
			log.Warn().Str("usedCommitment", *output.Total.Utilization.UsedCommitment).Msg("Cannot convert to float")
			span.SetStatus(codes.Error, "Error while computing UsedCommitment")
		} else {
			savingPlansUtilizationUsedCommitment.Set(usedCommitment)
		}

		percentUtilization, err := strconv.ParseFloat(*output.Total.Utilization.UtilizationPercentage, 64)
		if err != nil {
			log.Warn().Str("percentUtilization", *output.Total.Utilization.UtilizationPercentage).Msg("Cannot convert to float")
			span.SetStatus(codes.Error, "Error while computing UtilizationPercentage")
		} else {
			savingPlansUtilizationPercentage.Set(percentUtilization)
		}
		span.SetStatus(codes.Ok, "Saving PLans utilization metrics fetched")
	}

	costExplorerAPICalls.Inc()
}
