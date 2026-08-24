package application

import (
	"context"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

type EvidenceInput struct {
	PreviousEvidenceID string `json:"previous_evidence_id,omitempty"`
	Description        string `json:"description"`
	MediaType          string `json:"media_type"`
	ContentDigest      string `json:"content_digest"`
}
type ReviewInput struct {
	Decision domain.ReviewDecision `json:"decision"`
	Comment  string                `json:"comment"`
}

func (s *Service) SubmitEvidence(ctx context.Context, investigationID string, revision int64, actor Actor, input EvidenceInput) (*domain.CorrectiveEvidence, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleResponsible); err != nil {
		return nil, err
	}
	var evidence *domain.CorrectiveEvidence
	err := s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		investigation, err := tx.GetInvestigation(ctx, investigationID)
		if err != nil {
			return err
		}
		if err = investigation.CheckRevision(revision); err != nil {
			return err
		}
		evidenceChain, err := tx.ListEvidence(ctx, investigationID)
		if err != nil {
			return err
		}
		for _, existing := range evidenceChain {
			switch existing.ReviewDecision {
			case domain.DecisionPending:
				return domain.NewError(domain.CodeState, "该调查已有待复核证据")
			case domain.DecisionAccepted:
				return domain.NewError(domain.CodeState, "证据已接受，不能继续补充")
			}
		}
		previousEvidenceID := domain.NormalizeText(input.PreviousEvidenceID)
		if len(evidenceChain) == 0 {
			if previousEvidenceID != "" {
				return domain.FieldError("previous_evidence_id", "首次提交证据不能引用前序证据")
			}
		} else {
			latest := evidenceChain[len(evidenceChain)-1]
			if latest.ReviewDecision != domain.DecisionRejected || previousEvidenceID != latest.ID {
				return domain.FieldError("previous_evidence_id", "补充证据必须引用该调查最近一份被驳回证据")
			}
			for _, existing := range evidenceChain {
				if existing.PreviousEvidenceID == previousEvidenceID {
					return domain.FieldError("previous_evidence_id", "被引用证据已经存在补充版本")
				}
			}
		}
		previous := investigation.Revision
		evidence, err = domain.NewEvidence(s.newID("evd"), investigationID, actor.Name, input.Description, input.MediaType, input.ContentDigest, s.now(), previousEvidenceID)
		if err != nil {
			return err
		}
		if err = investigation.MarkEvidenceSubmitted(); err != nil {
			return err
		}
		if err = tx.InsertEvidence(ctx, evidence); err != nil {
			return err
		}
		if err = tx.UpdateInvestigation(ctx, investigation, previous); err != nil {
			return err
		}
		dossier, err := tx.GetDossier(ctx, investigation.DossierID)
		if err != nil {
			return err
		}
		eventType := "evidence.submitted"
		if previousEvidenceID != "" {
			eventType = "evidence.supplemented"
		}
		return tx.AppendAudit(ctx, domain.NewAudit(dossier.ID, eventType, actor.Name, actor.Role, dossier.Revision,
			map[string]any{"evidence_id": evidence.ID, "previous_evidence_id": previousEvidenceID, "digest": evidence.ContentDigest}, s.now()))
	})
	if err != nil {
		return nil, err
	}
	return evidence, nil
}

func (s *Service) ReviewEvidence(ctx context.Context, evidenceID string, investigationRevision int64, actor Actor, input ReviewInput) (*domain.CorrectiveEvidence, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleReviewer); err != nil {
		return nil, err
	}
	var evidence *domain.CorrectiveEvidence
	err := s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		var err error
		evidence, err = tx.GetEvidence(ctx, evidenceID)
		if err != nil {
			return err
		}
		investigation, err := tx.GetInvestigation(ctx, evidence.InvestigationID)
		if err != nil {
			return err
		}
		if err = investigation.CheckRevision(investigationRevision); err != nil {
			return err
		}
		if investigation.AssignedReviewer != actor.Name {
			return domain.NewError(domain.CodeForbidden, "仅调查认领人可以复核证据")
		}
		invPrevious := investigation.Revision
		if err = evidence.Review(actor.Name, input.Decision, input.Comment, s.now()); err != nil {
			return err
		}
		dossier, err := tx.GetDossier(ctx, investigation.DossierID)
		if err != nil {
			return err
		}
		dosPrevious := dossier.Revision
		if input.Decision == domain.DecisionAccepted {
			investigation.Resolve()
			dossier.SetStatus(domain.DossierReadyToClose)
		} else {
			investigation.RequestSupplement()
			dossier.SetStatus(domain.DossierRemediation)
		}
		if err = tx.UpdateEvidence(ctx, evidence); err != nil {
			return err
		}
		if err = tx.UpdateInvestigation(ctx, investigation, invPrevious); err != nil {
			return err
		}
		if err = tx.UpdateDossier(ctx, dossier, dosPrevious); err != nil {
			return err
		}
		eventType := "evidence.accepted"
		if input.Decision == domain.DecisionRejected {
			eventType = "evidence.rejected"
		}
		return tx.AppendAudit(ctx, domain.NewAudit(dossier.ID, eventType, actor.Name, actor.Role, dossier.Revision,
			map[string]any{"evidence_id": evidence.ID, "digest": evidence.ContentDigest, "decision": input.Decision, "comment": input.Comment}, s.now()))
	})
	if err != nil {
		return nil, err
	}
	return evidence, nil
}
