package application

import (
	"context"
	"encoding/json"
	"fmt"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

type TransferResult struct {
	Transfer         *domain.CustodyTransfer      `json:"transfer"`
	Dossier          *domain.SampleDossier        `json:"dossier"`
	Investigation    *domain.AnomalyInvestigation `json:"investigation,omitempty"`
	Findings         []chaincheck.Finding         `json:"findings"`
	IdempotentReplay bool                         `json:"idempotent_replay"`
}

type BatchTransferItem struct {
	Transfer *domain.CustodyTransfer `json:"transfer"`
	Findings []chaincheck.Finding    `json:"findings"`
}

type BatchTransferResult struct {
	Items            []BatchTransferItem          `json:"items"`
	Dossier          *domain.SampleDossier        `json:"dossier"`
	Investigation    *domain.AnomalyInvestigation `json:"investigation,omitempty"`
	IdempotentReplay bool                         `json:"idempotent_replay"`
}

func (s *Service) RegisterTransfer(ctx context.Context, dossierID, requestID string, revision int64, actor Actor, input domain.TransferInput) (*TransferResult, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleCustodian, domain.RoleReceiver); err != nil {
		return nil, err
	}
	if domain.NormalizeText(requestID) == "" {
		return nil, domain.FieldError("Idempotency-Key", "交接请求必须提供幂等标识")
	}
	canonical, _ := json.Marshal(input)
	digest := domain.DigestBytes(canonical)
	storedDigest, stored, found, err := s.store.GetIdempotency(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if found {
		if storedDigest != digest {
			return nil, domain.NewError(domain.CodeIdempotency, "同一幂等标识对应了不同载荷")
		}
		var result TransferResult
		if err = json.Unmarshal(stored, &result); err != nil {
			return nil, err
		}
		result.IdempotentReplay = true
		return &result, nil
	}
	var result *TransferResult
	err = s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		dossier, err := tx.GetDossier(ctx, dossierID)
		if err != nil {
			return err
		}
		transfers, err := tx.ListTransfers(ctx, dossierID)
		if err != nil {
			return err
		}
		transfer, err := domain.NewTransfer(s.newID("trn"), dossierID, requestID, digest, len(transfers)+1, input)
		if err != nil {
			return err
		}
		report := s.evaluator.EvaluateAddition(dossier, transfers, *transfer)
		if report.HasBlocking() {
			return domain.NewError(domain.CodeChain, "交接记录破坏监管链连续性")
		}
		previous := dossier.Revision
		if err = dossier.ApplyTransfer(revision, report.HasAnomaly()); err != nil {
			return err
		}
		if err = tx.InsertTransfer(ctx, transfer); err != nil {
			return err
		}
		if err = tx.UpdateDossier(ctx, dossier, previous); err != nil {
			return err
		}
		var investigation *domain.AnomalyInvestigation
		if report.HasAnomaly() {
			existing, err := tx.InvestigationByDossier(ctx, dossierID)
			if err != nil {
				return err
			}
			if existing == nil {
				investigation = domain.NewInvestigation(s.newID("inv"), dossierID, report.TriggerCodes(), s.now())
				if err = tx.InsertInvestigation(ctx, investigation); err != nil {
					return err
				}
			} else {
				previousInv := existing.Revision
				existing.MergeTriggers(report.TriggerCodes())
				if err = tx.UpdateInvestigation(ctx, existing, previousInv); err != nil {
					return err
				}
				investigation = existing
			}
		} else {
			investigation, err = tx.InvestigationByDossier(ctx, dossierID)
			if err != nil {
				return err
			}
		}
		event := domain.NewAudit(dossierID, "transfer.registered", actor.Name, actor.Role, dossier.Revision, map[string]any{"transfer_id": transfer.ID, "request_id": requestID, "findings": report.Findings}, s.now())
		if err = tx.AppendAudit(ctx, event); err != nil {
			return err
		}
		result = &TransferResult{Transfer: transfer, Dossier: dossier, Investigation: investigation, Findings: report.Findings}
		response, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return tx.PutIdempotency(ctx, requestID, digest, response)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func normalizeTransferInput(input domain.TransferInput) domain.TransferInput {
	input.StationCode = domain.NormalizeStation(input.StationCode)
	input.ReleasedBy = domain.NormalizeText(input.ReleasedBy)
	input.ReceivedBy = domain.NormalizeText(input.ReceivedBy)
	input.TransferredAt = domain.NormalizeTime(input.TransferredAt)
	return input
}

func (s *Service) RegisterTransferBatch(ctx context.Context, dossierID, requestID string, revision int64, actor Actor, inputs []domain.TransferInput) (*BatchTransferResult, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleCustodian, domain.RoleReceiver); err != nil {
		return nil, err
	}
	requestID = domain.NormalizeText(requestID)
	if requestID == "" {
		return nil, domain.FieldError("Idempotency-Key", "批量交接请求必须提供幂等标识")
	}
	if len(inputs) == 0 {
		return nil, domain.FieldError("transfers", "批量交接至少包含一笔记录")
	}
	if len(inputs) > 100 {
		return nil, domain.FieldError("transfers", "单次批量交接不能超过 100 笔")
	}
	for index := range inputs {
		inputs[index] = normalizeTransferInput(inputs[index])
	}
	canonical, _ := json.Marshal(struct {
		DossierID string                 `json:"dossier_id"`
		Transfers []domain.TransferInput `json:"transfers"`
	}{DossierID: dossierID, Transfers: inputs})
	digest := domain.DigestBytes(canonical)
	storedDigest, stored, found, err := s.store.GetIdempotency(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if found {
		if storedDigest != digest {
			return nil, domain.NewError(domain.CodeIdempotency, "同一幂等标识对应了不同载荷")
		}
		var result BatchTransferResult
		if err = json.Unmarshal(stored, &result); err != nil {
			return nil, err
		}
		result.IdempotentReplay = true
		return &result, nil
	}
	var result *BatchTransferResult
	err = s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		dossier, err := tx.GetDossier(ctx, dossierID)
		if err != nil {
			return err
		}
		if err = dossier.CheckRevision(revision); err != nil {
			return err
		}
		if dossier.Status != domain.DossierSubmitted && dossier.Status != domain.DossierInTransit && dossier.Status != domain.DossierInvestigation {
			return domain.NewError(domain.CodeState, "当前档案状态不能登记交接")
		}
		existing, err := tx.ListTransfers(ctx, dossierID)
		if err != nil {
			return err
		}
		staged := append([]domain.CustodyTransfer(nil), existing...)
		items := make([]BatchTransferItem, 0, len(inputs))
		var allFindings []chaincheck.Finding
		for index, input := range inputs {
			transfer, createErr := domain.NewTransfer(s.newID("trn"), dossierID, fmt.Sprintf("%s:%d", requestID, index+1), digest, len(staged)+1, input)
			if createErr != nil {
				return createErr
			}
			report := s.evaluator.EvaluateAddition(dossier, staged, *transfer)
			if report.HasBlocking() {
				return &domain.Error{Code: domain.CodeChain, Message: "批量交接记录破坏监管链连续性", Details: report.Findings}
			}
			items = append(items, BatchTransferItem{Transfer: transfer, Findings: report.Findings})
			allFindings = append(allFindings, report.Findings...)
			staged = append(staged, *transfer)
		}
		anomalous := false
		triggerSeen := map[string]bool{}
		var triggers []string
		for _, finding := range allFindings {
			if !finding.Blocking {
				anomalous = true
				if !triggerSeen[finding.Code] {
					triggers = append(triggers, finding.Code)
					triggerSeen[finding.Code] = true
				}
			}
		}
		previous := dossier.Revision
		if err = dossier.ApplyTransferBatch(revision, anomalous); err != nil {
			return err
		}
		for _, item := range items {
			if err = tx.InsertTransfer(ctx, item.Transfer); err != nil {
				return err
			}
		}
		if err = tx.UpdateDossier(ctx, dossier, previous); err != nil {
			return err
		}
		var investigation *domain.AnomalyInvestigation
		if anomalous {
			existingInv, err := tx.InvestigationByDossier(ctx, dossierID)
			if err != nil {
				return err
			}
			if existingInv == nil {
				investigation = domain.NewInvestigation(s.newID("inv"), dossierID, triggers, s.now())
				if err = tx.InsertInvestigation(ctx, investigation); err != nil {
					return err
				}
			} else {
				previousInv := existingInv.Revision
				existingInv.MergeTriggers(triggers)
				if err = tx.UpdateInvestigation(ctx, existingInv, previousInv); err != nil {
					return err
				}
				investigation = existingInv
			}
		} else {
			investigation, err = tx.InvestigationByDossier(ctx, dossierID)
			if err != nil {
				return err
			}
		}
		transferIDs := make([]string, len(items))
		for index, item := range items {
			transferIDs[index] = item.Transfer.ID
		}
		if err = tx.AppendAudit(ctx, domain.NewAudit(dossierID, "transfer.batch_registered", actor.Name, actor.Role, dossier.Revision,
			map[string]any{"request_id": requestID, "transfer_ids": transferIDs, "findings": allFindings}, s.now())); err != nil {
			return err
		}
		result = &BatchTransferResult{Items: items, Dossier: dossier, Investigation: investigation}
		response, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return marshalErr
		}
		return tx.PutIdempotency(ctx, requestID, digest, response)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
