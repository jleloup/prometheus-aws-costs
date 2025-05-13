package costexplorer

import (
	"context"
	"errors"
	"sync"
	"time"

	selfOtel "prometheus-aws-costs/src/utils/otel"

	awsCostexplorer "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/rs/zerolog/log"
)

type Statistics struct {
	CostExplorerAPICall float64
}

type CostExplorerFetcher struct {
	ctx        context.Context
	client     awsCostexplorer.Client
	telemetry  *selfOtel.Telemetry
	statistics Statistics
}

func NewCEFetcher(context context.Context, client awsCostexplorer.Client, telemetry *selfOtel.Telemetry) *CostExplorerFetcher {

	return &CostExplorerFetcher{
		ctx:       context,
		client:    client,
		telemetry: telemetry,
	}
}

func (r *CostExplorerFetcher) GetStatistics() Statistics {
	return r.statistics
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
	}

	e.statistics.CostExplorerAPICall++
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
	}

	e.statistics.CostExplorerAPICall++
}
