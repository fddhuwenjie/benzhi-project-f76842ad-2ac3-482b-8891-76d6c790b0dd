package application

import (
	"context"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

type ArchiveCondition struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Sequence int    `json:"sequence,omitempty"`
}

type ArchivePreflight struct {
	CanClose   bool               `json:"can_close"`
	Archived   bool               `json:"archived"`
	Revision   int64              `json:"revision"`
	ClosedAt   *time.Time         `json:"closed_at,omitempty"`
	Conditions []ArchiveCondition `json:"conditions"`
}

func (s *Service) buildArchivePreflight(dossier *domain.SampleDossier, transfers []domain.CustodyTransfer, investigation *domain.AnomalyInvestigation, evidence []domain.CorrectiveEvidence) *ArchivePreflight {
	report := &ArchivePreflight{Revision: dossier.Revision, Conditions: []ArchiveCondition{}}
	if dossier.Status == domain.DossierClosed {
		report.Archived = true
		report.ClosedAt = dossier.ClosedAt
		return report
	}
	chainReport := s.evaluator.EvaluateComplete(dossier, transfers)
	for _, finding := range chainReport.Findings {
		if finding.Blocking {
			report.Conditions = append(report.Conditions, ArchiveCondition{Code: finding.Code, Message: finding.Message, Sequence: finding.Sequence})
		}
	}
	if investigation == nil {
		report.Conditions = append(report.Conditions, ArchiveCondition{Code: "INVESTIGATION_REQUIRED", Message: "档案尚无已完成的异常调查"})
	} else {
		if domain.NormalizeText(investigation.RootCause) == "" || domain.NormalizeText(investigation.ImpactAssessment) == "" || domain.NormalizeText(investigation.RequiredAction) == "" {
			report.Conditions = append(report.Conditions, ArchiveCondition{Code: "INVESTIGATION_CONCLUSION_MISSING", Message: "调查尚未登记完整结论"})
		}
	}
	pending := false
	for _, item := range evidence {
		if item.ReviewDecision == domain.DecisionPending {
			pending = true
			break
		}
	}
	if pending {
		report.Conditions = append(report.Conditions, ArchiveCondition{Code: "EVIDENCE_REVIEW_PENDING", Message: "仍有证据尚未完成复核"})
	}
	if len(evidence) == 0 || evidence[len(evidence)-1].ReviewDecision != domain.DecisionAccepted {
		report.Conditions = append(report.Conditions, ArchiveCondition{Code: "FINAL_EVIDENCE_NOT_ACCEPTED", Message: "最终证据尚未被接受"})
	}
	if dossier.Status != domain.DossierReadyToClose {
		report.Conditions = append(report.Conditions, ArchiveCondition{Code: "DOSSIER_STATE_NOT_CLOSABLE", Message: "档案当前状态不可关闭"})
	}
	report.CanClose = len(report.Conditions) == 0
	return report
}

func (s *Service) PreflightClose(ctx context.Context, id string, actor Actor) (*ArchivePreflight, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleReviewer); err != nil {
		return nil, err
	}
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
	return s.buildArchivePreflight(dossier, transfers, investigation, evidence), nil
}

func (s *Service) CloseDossier(ctx context.Context, id string, revision int64, actor Actor) (*domain.SampleDossier, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleReviewer); err != nil {
		return nil, err
	}
	var dossier *domain.SampleDossier
	err := s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		var err error
		dossier, err = tx.GetDossier(ctx, id)
		if err != nil {
			return err
		}
		if err = dossier.CheckRevision(revision); err != nil {
			return err
		}
		transfers, err := tx.ListTransfers(ctx, id)
		if err != nil {
			return err
		}
		investigation, err := tx.InvestigationByDossier(ctx, id)
		if err != nil {
			return err
		}
		var evidence []domain.CorrectiveEvidence
		if investigation != nil {
			evidence, err = tx.ListEvidence(ctx, investigation.ID)
			if err != nil {
				return err
			}
		}
		report := s.buildArchivePreflight(dossier, transfers, investigation, evidence)
		if !report.CanClose {
			return &domain.Error{Code: domain.CodeState, Message: "档案尚未满足关闭条件", Details: report}
		}
		previous := dossier.Revision
		if err = dossier.Close(dossier.Revision, s.now()); err != nil {
			return err
		}
		if err = tx.UpdateDossier(ctx, dossier, previous); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.NewAudit(id, "dossier.closed", actor.Name, actor.Role, dossier.Revision,
			map[string]any{"transfer_count": len(transfers), "preflight_conditions": report.Conditions}, s.now()))
	})
	if err != nil {
		return nil, err
	}
	return dossier, nil
}
