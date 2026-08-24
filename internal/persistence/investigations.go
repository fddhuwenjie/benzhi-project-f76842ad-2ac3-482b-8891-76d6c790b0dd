package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

func (t *Tx) InsertInvestigation(ctx context.Context, v *domain.AnomalyInvestigation) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO investigations(id,dossier_id,status,revision,snapshot) VALUES(?,?,?,?,?)`, v.ID, v.DossierID, v.Status, v.Revision, payload)
	return translateSQLError(err)
}

func (t *Tx) UpdateInvestigation(ctx context.Context, v *domain.AnomalyInvestigation, previous int64) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	result, err := t.tx.ExecContext(ctx, `UPDATE investigations SET status=?,revision=?,snapshot=? WHERE id=? AND revision=?`, v.Status, v.Revision, payload, v.ID, previous)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return domain.NewError(domain.CodeConflict, "调查已被其他请求修改")
	}
	return nil
}

func (s *Store) GetInvestigation(ctx context.Context, id string) (*domain.AnomalyInvestigation, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT snapshot FROM investigations WHERE id=?`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewError(domain.CodeNotFound, "调查不存在")
	}
	if err != nil {
		return nil, err
	}
	var item domain.AnomalyInvestigation
	if err = json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (t *Tx) GetInvestigation(ctx context.Context, id string) (*domain.AnomalyInvestigation, error) {
	var payload []byte
	err := t.tx.QueryRowContext(ctx, `SELECT snapshot FROM investigations WHERE id=?`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewError(domain.CodeNotFound, "调查不存在")
	}
	if err != nil {
		return nil, err
	}
	var item domain.AnomalyInvestigation
	if err = json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) InvestigationByDossier(ctx context.Context, dossierID string) (*domain.AnomalyInvestigation, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT snapshot FROM investigations WHERE dossier_id=?`, dossierID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var item domain.AnomalyInvestigation
	if err = json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (t *Tx) InvestigationByDossier(ctx context.Context, dossierID string) (*domain.AnomalyInvestigation, error) {
	var payload []byte
	err := t.tx.QueryRowContext(ctx, `SELECT snapshot FROM investigations WHERE dossier_id=?`, dossierID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var item domain.AnomalyInvestigation
	if err = json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (t *Tx) InsertEvidence(ctx context.Context, v *domain.CorrectiveEvidence) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO evidence(id,investigation_id,decision,snapshot) VALUES(?,?,?,?)`, v.ID, v.InvestigationID, v.ReviewDecision, payload)
	return translateSQLError(err)
}

func (t *Tx) UpdateEvidence(ctx context.Context, v *domain.CorrectiveEvidence) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	result, err := t.tx.ExecContext(ctx, `UPDATE evidence SET decision=?,snapshot=? WHERE id=? AND decision=?`, v.ReviewDecision, payload, v.ID, domain.DecisionPending)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return domain.NewError(domain.CodeConflict, "证据已被复核")
	}
	return nil
}

func (t *Tx) GetEvidence(ctx context.Context, id string) (*domain.CorrectiveEvidence, error) {
	var payload []byte
	err := t.tx.QueryRowContext(ctx, `SELECT snapshot FROM evidence WHERE id=?`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewError(domain.CodeNotFound, "证据不存在")
	}
	if err != nil {
		return nil, err
	}
	var item domain.CorrectiveEvidence
	if err = json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) ListEvidence(ctx context.Context, investigationID string) ([]domain.CorrectiveEvidence, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT snapshot FROM evidence WHERE investigation_id=? ORDER BY rowid`, investigationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.CorrectiveEvidence
	for rows.Next() {
		var payload []byte
		var item domain.CorrectiveEvidence
		if err = rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (t *Tx) ListEvidence(ctx context.Context, investigationID string) ([]domain.CorrectiveEvidence, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT snapshot FROM evidence WHERE investigation_id=? ORDER BY rowid`, investigationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.CorrectiveEvidence
	for rows.Next() {
		var payload []byte
		var item domain.CorrectiveEvidence
		if err = rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

type InvestigationDossier struct {
	Investigation domain.AnomalyInvestigation
	Dossier       domain.SampleDossier
}

func (s *Store) ListInvestigationDossiers(ctx context.Context) ([]InvestigationDossier, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT i.snapshot,d.snapshot FROM investigations i JOIN dossiers d ON d.id=i.dossier_id ORDER BY CASE WHEN i.status='unassigned' THEN 0 ELSE 1 END,i.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []InvestigationDossier
	for rows.Next() {
		var investigationPayload, dossierPayload []byte
		var item InvestigationDossier
		if err = rows.Scan(&investigationPayload, &dossierPayload); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(investigationPayload, &item.Investigation); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(dossierPayload, &item.Dossier); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
