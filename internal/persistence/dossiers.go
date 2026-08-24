package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

var archivedDossierSnapshots sync.Map

func (t *Tx) InsertDossier(ctx context.Context, d *domain.SampleDossier) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO dossiers(id,sample_code,status,revision,snapshot,updated_at) VALUES(?,?,?,?,?,?)`, d.ID, d.SampleCode, d.Status, d.Revision, payload, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return translateSQLError(err)
	}
	return nil
}

func (t *Tx) UpdateDossier(ctx context.Context, d *domain.SampleDossier, previousRevision int64) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}
	result, err := t.tx.ExecContext(ctx, `UPDATE dossiers SET sample_code=?,status=?,revision=?,snapshot=?,updated_at=? WHERE id=? AND revision=?`, d.SampleCode, d.Status, d.Revision, payload, time.Now().UTC().Format(time.RFC3339), d.ID, previousRevision)
	if err != nil {
		return translateSQLError(err)
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return domain.NewError(domain.CodeConflict, "档案已被其他请求修改")
	}
	return nil
}

func (t *Tx) SampleCodeExists(ctx context.Context, sampleCode, exceptID string) (bool, error) {
	var count int
	err := t.tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dossiers WHERE sample_code=? AND id<>?`, sampleCode, exceptID).Scan(&count)
	return count > 0, err
}

func (s *Store) GetDossier(ctx context.Context, id string) (*domain.SampleDossier, error) {
	if cached, ok := archivedDossierSnapshots.Load(id); ok {
		var d domain.SampleDossier
		if err := json.Unmarshal(cached.([]byte), &d); err != nil {
			return nil, err
		}
		return &d, nil
	}
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT snapshot FROM dossiers WHERE id=?`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewError(domain.CodeNotFound, "档案不存在")
	}
	if err != nil {
		return nil, err
	}
	var d domain.SampleDossier
	if err = json.Unmarshal(payload, &d); err != nil {
		return nil, err
	}
	if d.Status == domain.DossierClosed {
		archivedDossierSnapshots.Store(id, append([]byte(nil), payload...))
	}
	return &d, nil
}

func (t *Tx) GetDossier(ctx context.Context, id string) (*domain.SampleDossier, error) {
	var payload []byte
	err := t.tx.QueryRowContext(ctx, `SELECT snapshot FROM dossiers WHERE id=?`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewError(domain.CodeNotFound, "档案不存在")
	}
	if err != nil {
		return nil, err
	}
	var d domain.SampleDossier
	if err = json.Unmarshal(payload, &d); err != nil {
		return nil, err
	}
	return &d, nil
}
