package application

import (
	"context"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

type TransferProgress struct {
	DossierRevision   int64                `json:"dossier_revision"`
	Status            string               `json:"status"`
	CompletedStations int                  `json:"completed_stations"`
	TotalStations     int                  `json:"total_stations"`
	NextStation       string               `json:"next_station,omitempty"`
	CurrentCustodian  string               `json:"current_custodian,omitempty"`
	RemainingStations []string             `json:"remaining_stations"`
	RouteComplete     bool                 `json:"route_complete"`
	Deadline          *time.Time           `json:"deadline,omitempty"`
	RemainingMinutes  *int64               `json:"remaining_minutes,omitempty"`
	TimeRisk          string               `json:"time_risk,omitempty"`
	BlockingGaps      []chaincheck.Finding `json:"blocking_gaps"`
	Anomalies         []chaincheck.Finding `json:"anomalies"`
}

func (s *Service) GetTransferProgress(ctx context.Context, dossierID string) (*TransferProgress, error) {
	dossier, err := s.store.GetDossier(ctx, dossierID)
	if err != nil {
		return nil, err
	}
	result := &TransferProgress{DossierRevision: dossier.Revision, TotalStations: len(dossier.ExpectedRoute), RemainingStations: []string{}, BlockingGaps: []chaincheck.Finding{}, Anomalies: []chaincheck.Finding{}}
	if dossier.Status == domain.DossierDraft {
		result.Status = "not_submitted"
		return result, nil
	}
	transfers, err := s.store.ListTransfers(ctx, dossierID)
	if err != nil {
		return nil, err
	}
	report := s.evaluator.EvaluateExisting(dossier, transfers)
	for _, finding := range report.Findings {
		if finding.Blocking {
			result.BlockingGaps = append(result.BlockingGaps, finding)
		} else {
			result.Anomalies = append(result.Anomalies, finding)
		}
	}
	completed := len(transfers)
	if completed > len(dossier.ExpectedRoute) {
		completed = len(dossier.ExpectedRoute)
	}
	result.CompletedStations = completed
	result.RemainingStations = append([]string{}, dossier.ExpectedRoute[completed:]...)
	result.RouteComplete = completed == len(dossier.ExpectedRoute) && len(result.BlockingGaps) == 0
	if !result.RouteComplete && completed < len(dossier.ExpectedRoute) {
		result.NextStation = dossier.ExpectedRoute[completed]
	}
	if len(transfers) > 0 {
		result.CurrentCustodian = transfers[len(transfers)-1].ReceivedBy
	} else {
		result.CurrentCustodian = dossier.ResponsiblePerson
	}
	deadline := dossier.CollectedAt.Add(time.Duration(dossier.MaximumTransitMinutes) * time.Minute)
	deadline = domain.NormalizeTime(deadline)
	remaining := int64(deadline.Sub(s.now()).Minutes())
	result.Deadline, result.RemainingMinutes = &deadline, &remaining
	switch {
	case remaining < 0:
		result.TimeRisk = "overdue"
	case remaining <= 30:
		result.TimeRisk = "near_deadline"
	default:
		result.TimeRisk = "normal"
	}
	result.Status = string(dossier.Status)
	return result, nil
}
