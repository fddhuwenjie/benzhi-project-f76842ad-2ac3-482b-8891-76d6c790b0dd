package sharedstorecloselifecycle_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestClosingStoreDoesNotInvalidatePeer(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "custody.db")
	first, err := persistence.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	dossier, err := domain.NewDossier("dos_shared_pool", domain.DraftInput{
		SampleCode: "SHARED-POOL-1",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err = first.WithTx(ctx, func(tx *persistence.Tx) error {
		return tx.InsertDossier(ctx, dossier)
	}); err != nil {
		t.Fatal(err)
	}

	second, err := persistence.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}

	loaded, err := second.GetDossier(ctx, dossier.ID)
	if err != nil {
		t.Fatalf("关闭一个 Store 不应使仍在使用的同路径 Store 失效: %v", err)
	}
	if loaded.SampleCode != dossier.SampleCode {
		t.Fatalf("第二个 Store 读取了错误档案: %+v", loaded)
	}
}
