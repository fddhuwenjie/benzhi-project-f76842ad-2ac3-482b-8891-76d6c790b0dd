package persistence

import (
	"context"
	"encoding/json"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

func (t *Tx) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO audit_events(dossier_id,event_type,actor,role,revision,occurred_at,details) VALUES(?,?,?,?,?,?,?)`, event.DossierID, event.EventType, event.Actor, event.Role, event.Revision, event.OccurredAt.Format(time.RFC3339), []byte(event.Details))
	return err
}

func (s *Store) ListAudit(ctx context.Context, dossierID string) ([]domain.AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,event_type,actor,role,revision,occurred_at,details FROM audit_events WHERE dossier_id=? ORDER BY id`, dossierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.AuditEvent
	for rows.Next() {
		var item domain.AuditEvent
		var occurred string
		var details []byte
		item.DossierID = dossierID
		if err = rows.Scan(&item.ID, &item.EventType, &item.Actor, &item.Role, &item.Revision, &occurred, &details); err != nil {
			return nil, err
		}
		item.OccurredAt, _ = time.Parse(time.RFC3339, occurred)
		item.Details = json.RawMessage(details)
		result = append(result, item)
	}
	return result, rows.Err()
}
