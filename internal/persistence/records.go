package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

func (t *Tx) InsertTransfer(ctx context.Context, v *domain.CustodyTransfer) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO transfers(id,dossier_id,sequence,request_id,content_digest,snapshot) VALUES(?,?,?,?,?,?)`, v.ID, v.DossierID, v.Sequence, v.RequestID, v.ContentDigest, payload)
	return translateSQLError(err)
}

func (s *Store) ListTransfers(ctx context.Context, dossierID string) ([]domain.CustodyTransfer, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT snapshot FROM transfers WHERE dossier_id=? ORDER BY sequence`, dossierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.CustodyTransfer
	for rows.Next() {
		var payload []byte
		var item domain.CustodyTransfer
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

func (t *Tx) ListTransfers(ctx context.Context, dossierID string) ([]domain.CustodyTransfer, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT snapshot FROM transfers WHERE dossier_id=? ORDER BY sequence`, dossierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.CustodyTransfer
	for rows.Next() {
		var payload []byte
		var item domain.CustodyTransfer
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

func (t *Tx) PutIdempotency(ctx context.Context, requestID, digest string, response []byte) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO idempotency_records(request_id,payload_digest,response_json,created_at) VALUES(?,?,?,strftime('%Y-%m-%dT%H:%M:%SZ','now'))`, requestID, digest, response)
	return translateSQLError(err)
}

func (s *Store) GetIdempotency(ctx context.Context, requestID string) (string, []byte, bool, error) {
	var digest string
	var response []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload_digest,response_json FROM idempotency_records WHERE request_id=?`, requestID).Scan(&digest, &response)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}
	return digest, response, true, nil
}

func translateSQLError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique") || strings.Contains(message, "constraint") {
		return domain.NewError(domain.CodeConflict, "数据唯一性约束冲突")
	}
	return err
}
