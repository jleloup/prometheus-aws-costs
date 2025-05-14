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
	savingPlansCoveragePercentage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pac_savingplans_coverage_percent",
		Help: "Percentage of cost covered by Saving Plans",
	}, []string{"aws_instance_type_family", "aws_region", "aws_service"})
	savingPlansCoverageOnDemandCost = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pac_savingplans_coverage_ondemandcost_dollar",
		Help: "Cost for On Demand in Dollars",
	}, []string{"aws_instance_type_family", "aws_region", "aws_service"})
	savingPlansCoverageSpendCoveredBySavingsPlans = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pac_savingplans_coverage_coveredcost_dollar",
		Help: "Number of dollar covered by Saving Plans",
	}, []string{"aws_instance_type_family", "aws_region", "aws_service"})
	savingPlansCoverageTotalCost = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pac_savingplans_coverage_totalcost_dollar",
		Help: "total cost spent on AWS regardless of Saving Plans",
	}, []string{"aws_instance_type_family", "aws_region", "aws_service"})
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
	savingPlansUtilizationNetSavings = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pac_savingplans_utilization_netsavings_dollar",
		Help: "Number of dollar saved by saving plans.",
	})
	savingPlansUtilizationOnDemandEquivalent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pac_savingplans_utilization_ondemand_equivalent_dollar",
		Help: "Number of dollar you would have paid using on-demand.",
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

	log.Info().Msg("Get Saving Plans coverage metrics")

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

	var nextToken *string = nil

	for {
		output, err := e.client.GetSavingsPlansCoverage(ctx, &awsCostexplorer.GetSavingsPlansCoverageInput{
			TimePeriod: &types.DateInterval{
				Start: &startStr,
				End:   &endStr,
			},
			MaxResults:  &maxResult,
			Granularity: granularity,
			GroupBy:     groupBy,
			Metrics:     metrics,
			NextToken:   nextToken,
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
			break
		} else {
			costExplorerAPICalls.Inc()

			// Process SavingPlansCoverage items
			for _, item := range output.SavingsPlansCoverages {
				log.Debug().Interface("coverage", item).Msg("Output from GetSavingsPlansCoverage")

				// Fetch attributes for labels
				instance_type_family := item.Attributes["INSTANCE_TYPE_FAMILY"]
				region := item.Attributes["REGION"]
				service := item.Attributes["SERVICE"]

				// Saving plan coverage percentage
				coveragePercentage, err := strconv.ParseFloat(*item.Coverage.CoveragePercentage, 64)
				if err != nil {
					log.Warn().Str("coveragePercentage", *item.Coverage.CoveragePercentage).Msg("Cannot convert to float")
					span.SetStatus(codes.Error, "Error while computing CoveragePercentage")
				} else {
					savingPlansCoveragePercentage.WithLabelValues(
						instance_type_family, region, service,
					).Set(coveragePercentage)
				}

				// On Demand cost
				onDemandCost, err := strconv.ParseFloat(*item.Coverage.OnDemandCost, 64)
				if err != nil {
					log.Warn().Str("onDemandCost", *item.Coverage.OnDemandCost).Msg("Cannot convert to float")
					span.SetStatus(codes.Error, "Error while computing OnDemandCost")
				} else {
					savingPlansCoverageOnDemandCost.WithLabelValues(
						instance_type_family, region, service,
					).Set(onDemandCost)
				}

				// Cost covered by saving plans
				coveredCost, err := strconv.ParseFloat(*item.Coverage.SpendCoveredBySavingsPlans, 64)
				if err != nil {
					log.Warn().Str("coveredCost", *item.Coverage.SpendCoveredBySavingsPlans).Msg("Cannot convert to float")
					span.SetStatus(codes.Error, "Error while computing SpendCoveredBySavingsPlans")
				} else {
					savingPlansCoverageSpendCoveredBySavingsPlans.WithLabelValues(
						instance_type_family, region, service,
					).Set(coveredCost)
				}

				// Total cost regardless of Saving plans
				totalCost, err := strconv.ParseFloat(*item.Coverage.TotalCost, 64)
				if err != nil {
					log.Warn().Str("totalCost", *item.Coverage.TotalCost).Msg("Cannot convert to float")
					span.SetStatus(codes.Error, "Error while computing TotalCost")
				} else {
					savingPlansCoverageTotalCost.WithLabelValues(
						instance_type_family, region, service,
					).Set(totalCost)
				}
			}

			if output.NextToken == nil {
				break // No more pages
			}

			nextToken = output.NextToken

		}
	}
}

func (e *CostExplorerFetcher) GetSavingPlansUtilizationMetrics(ctx context.Context, wg *sync.WaitGroup) {

	log.Info().Msg("Get Saving Plans utilization metrics")

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
	costExplorerAPICalls.Inc()

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
		log.Debug().Interface("utilizationOutput", output).Msg("Saving Plans utilization output")

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

		netSavings, err := strconv.ParseFloat(*output.Total.Savings.NetSavings, 64)
		if err != nil {
			log.Warn().Str("netSavings", *output.Total.Savings.NetSavings).Msg("Cannot convert to float")
			span.SetStatus(codes.Error, "Error while computing NetSavings")
		} else {
			savingPlansUtilizationNetSavings.Set(netSavings)
		}

		onDemandEquivalent, err := strconv.ParseFloat(*output.Total.Savings.OnDemandCostEquivalent, 64)
		if err != nil {
			log.Warn().Str("onDemandCostEquivalent", *output.Total.Savings.OnDemandCostEquivalent).Msg("Cannot convert to float")
			span.SetStatus(codes.Error, "Error while computing OnDemandCostEquivalent")
		} else {
			savingPlansUtilizationOnDemandEquivalent.Set(onDemandEquivalent)
		}

		span.SetStatus(codes.Ok, "Saving Plans utilization metrics fetched")
	}
}
