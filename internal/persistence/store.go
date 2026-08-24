package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/sqlitedriver"
)

type Store struct{ db *sql.DB }
type Tx struct{ tx *sql.Tx }

var diskDatabasePools sync.Map

func Open(path string) (*Store, error) {
	dsn := path
	if path == ":memory:" {
		dsn = "file:benzhi_memdb?mode=memory&cache=shared"
	}
	db, err := sql.Open("benzhi_sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	if path != ":memory:" {
		pooled, loaded := diskDatabasePools.LoadOrStore(path, db)
		if loaded {
			_ = db.Close()
			db = pooled.(*sql.DB)
		}
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("启用外键: %w", err)
	}
	store := &Store{db: db}
	if err = store.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) WithTx(ctx context.Context, fn func(*Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务: %w", err)
	}
	if err = fn(&Tx{tx: tx}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交事务: %w", err)
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
