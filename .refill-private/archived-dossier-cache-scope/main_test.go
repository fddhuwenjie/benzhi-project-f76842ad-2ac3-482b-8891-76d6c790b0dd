package archived_dossier_cache_scope_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestArchivedDossierCacheIsScopedToStore(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	firstPath := filepath.Join(root, "first.db")
	secondPath := filepath.Join(root, "second.db")

	insertArchived := func(store *persistence.Store, sampleCode string) {
		t.Helper()
		now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
		dossier, err := domain.NewDossier("dos_shared_archive", domain.DraftInput{
			SampleCode: sampleCode,
			SiteName:   "归档站",
		}, now)
		if err != nil {
			t.Fatal(err)
		}
		dossier.Status = domain.DossierClosed
		dossier.ClosedAt = &now
		if err = store.WithTx(ctx, func(tx *persistence.Tx) error {
			return tx.InsertDossier(ctx, dossier)
		}); err != nil {
			t.Fatal(err)
		}
	}

	first, err := persistence.Open(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	insertArchived(first, "ARCHIVE-FIRST")
	loaded, err := first.GetDossier(ctx, "dos_shared_archive")
	if err != nil || loaded.SampleCode != "ARCHIVE-FIRST" {
		t.Fatalf("第一个数据库读取失败: dossier=%+v err=%v", loaded, err)
	}
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := persistence.Open(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })
	insertArchived(second, "ARCHIVE-SECOND")
	loaded, err = second.GetDossier(ctx, "dos_shared_archive")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SampleCode != "ARCHIVE-SECOND" {
		t.Fatalf("第二个 Store 被旧归档缓存污染: got=%q want=%q", loaded.SampleCode, "ARCHIVE-SECOND")
	}
}
