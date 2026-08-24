package application

import (
	"context"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

type DossierView struct {
	Dossier       *domain.SampleDossier        `json:"dossier"`
	Transfers     []domain.CustodyTransfer     `json:"transfers"`
	Investigation *domain.AnomalyInvestigation `json:"investigation,omitempty"`
	Evidence      []domain.CorrectiveEvidence  `json:"evidence"`
	History       []domain.AuditEvent          `json:"history"`
}

func (s *Service) GetDossier(ctx context.Context, id string) (*DossierView, error) {
	dossier, err := s.store.GetDossier(ctx, id)
	if err != nil {
		return nil, err
	}
	transfers, err := s.store.ListTransfers(ctx, id)
	if err != nil {
		return nil, err
	}
	investigation, err := s.store.InvestigationByDossier(ctx, id)
	if err != nil {
		return nil, err
	}
	var evidence []domain.CorrectiveEvidence
	if investigation != nil {
		evidence, err = s.store.ListEvidence(ctx, investigation.ID)
		if err != nil {
			return nil, err
		}
	}
	history, err := s.store.ListAudit(ctx, id)
	if err != nil {
		return nil, err
	}
	if transfers == nil {
		transfers = []domain.CustodyTransfer{}
	}
	if evidence == nil {
		evidence = []domain.CorrectiveEvidence{}
	}
	if history == nil {
		history = []domain.AuditEvent{}
	}
	return &DossierView{Dossier: dossier, Transfers: transfers, Investigation: investigation, Evidence: evidence, History: history}, nil
}

func (s *Service) GetAudit(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	if _, err := s.store.GetDossier(ctx, id); err != nil {
		return nil, err
	}
	events, err := s.store.ListAudit(ctx, id)
	if events == nil {
		events = []domain.AuditEvent{}
	}
	return events, err
}
