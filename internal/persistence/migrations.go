package persistence

import (
	"context"
	"fmt"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS dossiers (id TEXT PRIMARY KEY, sample_code TEXT NOT NULL UNIQUE, status TEXT NOT NULL, revision INTEGER NOT NULL, snapshot BLOB NOT NULL, updated_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS transfers (id TEXT PRIMARY KEY, dossier_id TEXT NOT NULL REFERENCES dossiers(id), sequence INTEGER NOT NULL, request_id TEXT NOT NULL UNIQUE, content_digest TEXT NOT NULL, snapshot BLOB NOT NULL, UNIQUE(dossier_id, sequence))`,
	`CREATE TABLE IF NOT EXISTS investigations (id TEXT PRIMARY KEY, dossier_id TEXT NOT NULL UNIQUE REFERENCES dossiers(id), status TEXT NOT NULL, revision INTEGER NOT NULL, snapshot BLOB NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS evidence (id TEXT PRIMARY KEY, investigation_id TEXT NOT NULL REFERENCES investigations(id), decision TEXT NOT NULL, snapshot BLOB NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS idempotency_records (request_id TEXT PRIMARY KEY, payload_digest TEXT NOT NULL, response_json BLOB NOT NULL, created_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS audit_events (id INTEGER PRIMARY KEY AUTOINCREMENT, dossier_id TEXT NOT NULL REFERENCES dossiers(id), event_type TEXT NOT NULL, actor TEXT NOT NULL, role TEXT NOT NULL, revision INTEGER NOT NULL, occurred_at TEXT NOT NULL, details BLOB NOT NULL)`,
	`CREATE INDEX IF NOT EXISTS idx_transfers_dossier ON transfers(dossier_id, sequence)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_dossier ON audit_events(dossier_id, id)`,
	`CREATE INDEX IF NOT EXISTS idx_investigations_queue ON investigations(status, id)`,
}

func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for index, statement := range migrations {
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行数据库迁移 %d: %w", index+1, err)
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations(version,applied_at) VALUES(1,strftime('%Y-%m-%dT%H:%M:%SZ','now'))`); err != nil {
		return err
	}
	return tx.Commit()
}
