package application

import (
	"context"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

type ConclusionInput struct {
	RootCause        string `json:"root_cause"`
	ImpactAssessment string `json:"impact_assessment"`
	RequiredAction   string `json:"required_action"`
}

func (s *Service) ClaimInvestigation(ctx context.Context, id string, revision int64, actor Actor) (*domain.AnomalyInvestigation, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleReviewer); err != nil {
		return nil, err
	}
	var investigation *domain.AnomalyInvestigation
	err := s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		var err error
		investigation, err = tx.GetInvestigation(ctx, id)
		if err != nil {
			return err
		}
		previous := investigation.Revision
		if err = investigation.Claim(revision, actor.Name); err != nil {
			return err
		}
		if err = tx.UpdateInvestigation(ctx, investigation, previous); err != nil {
			return err
		}
		dossier, err := tx.GetDossier(ctx, investigation.DossierID)
		if err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.NewAudit(dossier.ID, "investigation.claimed", actor.Name, actor.Role, dossier.Revision, map[string]any{"investigation_id": id, "investigation_revision": investigation.Revision}, s.now()))
	})
	if err != nil {
		return nil, failedOperation("认领调查", err)
	}
	return investigation, nil
}

func (s *Service) ReleaseInvestigation(ctx context.Context, id string, revision int64, actor Actor, reason string) (*domain.AnomalyInvestigation, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleReviewer); err != nil {
		return nil, err
	}
	var investigation *domain.AnomalyInvestigation
	err := s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		var err error
		investigation, err = tx.GetInvestigation(ctx, id)
		if err != nil {
			return err
		}
		previous := investigation.Revision
		normalizedReason, err := investigation.Release(revision, actor.Name, reason)
		if err != nil {
			return err
		}
		if err = tx.UpdateInvestigation(ctx, investigation, previous); err != nil {
			return err
		}
		dossier, err := tx.GetDossier(ctx, investigation.DossierID)
		if err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.NewAudit(dossier.ID, "investigation.released", actor.Name, actor.Role, dossier.Revision,
			map[string]any{"investigation_id": id, "investigation_revision": investigation.Revision, "reason": normalizedReason}, s.now()))
	})
	if err != nil {
		return nil, err
	}
	return investigation, nil
}

func (s *Service) RecordConclusion(ctx context.Context, id string, revision int64, actor Actor, input ConclusionInput) (*domain.AnomalyInvestigation, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleReviewer); err != nil {
		return nil, err
	}
	var investigation *domain.AnomalyInvestigation
	err := s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		var err error
		investigation, err = tx.GetInvestigation(ctx, id)
		if err != nil {
			return err
		}
		invPrevious := investigation.Revision
		if err = investigation.RecordConclusion(revision, actor.Name, input.RootCause, input.ImpactAssessment, input.RequiredAction); err != nil {
			return err
		}
		dossier, err := tx.GetDossier(ctx, investigation.DossierID)
		if err != nil {
			return err
		}
		if err = dossier.EnsureMutable(); err != nil {
			return err
		}
		dosPrevious := dossier.Revision
		dossier.SetStatus(domain.DossierRemediation)
		if err = tx.UpdateInvestigation(ctx, investigation, invPrevious); err != nil {
			return err
		}
		if err = tx.UpdateDossier(ctx, dossier, dosPrevious); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.NewAudit(dossier.ID, "investigation.conclusion_recorded", actor.Name, actor.Role, dossier.Revision, input, s.now()))
	})
	if err != nil {
		return nil, failedOperation("登记调查结论", err)
	}
	return investigation, nil
}
