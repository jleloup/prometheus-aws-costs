package costexplorer

import (
	"context"
	"time"

	awsCostexplorer "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
)

type Statistics struct {
	CostExplorerAPICall float64
}

var tracer = otel.Tracer("prometheus-aws-costs/aws/costexplorer")

func NewCEFetcher(context context.Context, client awsCostexplorer.Client) *CostExplorerFetcher {
	return &CostExplorerFetcher{
		ctx:    context,
		client: client,
	}
}

type CostExplorerFetcher struct {
	ctx        context.Context
	client     awsCostexplorer.Client
	statistics Statistics
}

func (e *CostExplorerFetcher) GetSavingPlansMetrics(ctx context.Context) {

	log.Debug().Msg("Get Saving Plans metrics")
	_, span := tracer.Start(ctx, "get-costexplorer-metrics")
	defer span.End()

	now := time.Now()
	nowStr := now.Format("2052-05-09")

	output, err := e.client.GetSavingsPlansUtilization(ctx, &awsCostexplorer.GetSavingsPlansUtilizationInput{
		TimePeriod: &types.DateInterval{
			Start: &nowStr,
			End:   &nowStr,
		},
		Granularity: "DAILY",
	})

	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Cannot fetch Saving Plans utilization")
	}

	e.statistics.CostExplorerAPICall++
	log.Debug().Interface("dict", output).Msg("Saving Plans utilization output")
}
