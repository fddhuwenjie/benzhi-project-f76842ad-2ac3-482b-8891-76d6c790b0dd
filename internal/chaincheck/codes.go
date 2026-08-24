package chaincheck

const (
	CodeRouteDeviation     = "ROUTE_DEVIATION"
	CodeTemperatureHigh    = "TEMPERATURE_HIGH"
	CodeTemperatureLow     = "TEMPERATURE_LOW"
	CodeSealBroken         = "SEAL_BROKEN"
	CodeSealMissing        = "SEAL_MISSING"
	CodeTransitTimeout     = "TRANSIT_TIMEOUT"
	CodeSequenceGap        = "SEQUENCE_GAP"
	CodePartyDiscontinuity = "PARTY_DISCONTINUITY"
	CodeTimeRegression     = "TIME_REGRESSION"
	CodeRouteIncomplete    = "ROUTE_INCOMPLETE"
)

type Finding struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Sequence int    `json:"sequence,omitempty"`
	Blocking bool   `json:"blocking"`
}

type Report struct {
	Findings []Finding `json:"findings"`
}

func (r Report) HasAnomaly() bool {
	for _, finding := range r.Findings {
		if !finding.Blocking {
			return true
		}
	}
	return false
}

func (r Report) HasBlocking() bool {
	for _, finding := range r.Findings {
		if finding.Blocking {
			return true
		}
	}
	return false
}

func (r Report) TriggerCodes() []string {
	seen := map[string]bool{}
	var result []string
	for _, finding := range r.Findings {
		if !finding.Blocking && !seen[finding.Code] {
			result = append(result, finding.Code)
			seen[finding.Code] = true
		}
	}
	return result
}
