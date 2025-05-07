package costexplorer

import (
	awsCostexplorer "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/rs/zerolog/log"
)

func LoadSavingPlans(ceClient awsCostexplorer.Client) {

	log.Debug().Msg("Loading Saving Plans")
}
