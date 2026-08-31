package domain

import "time"

type CustomerAuthorizations struct {
	UploadsAuthorized                bool
	AnalysisUseAuthorized            bool
	ExternalRedistributionAuthorized bool
	ConfirmedAt                      *time.Time
}
