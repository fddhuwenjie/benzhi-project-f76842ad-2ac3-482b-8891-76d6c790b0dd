package investigation_cursor_status_test

import (
	"context"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestQueueCursorDoesNotRepeatClaimedItem(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service := application.New(store, chaincheck.New())
	ctx := context.Background()
	custodian := application.Actor{Name: "保管员", Role: domain.RoleCustodian}
	reviewer := application.Actor{Name: "审核员", Role: domain.RoleReviewer}
	now := time.Now().UTC().Truncate(time.Second)
	insert := func(code, investigationID string, detectedAt time.Time) {
		t.Helper()
		dossier, createErr := service.CreateDossier(ctx, custodian, domain.DraftInput{SampleCode: code})
		if createErr != nil {
			t.Fatal(createErr)
		}
		if dossier.ID == "" {
			t.Fatal("档案标识不能为空")
		}
		insertErr := store.WithTx(ctx, func(tx *persistence.Tx) error {
			return tx.InsertInvestigation(ctx, domain.NewInvestigation(investigationID, dossier.ID, []string{chaincheck.CodeSealBroken}, detectedAt))
		})
		if insertErr != nil {
			t.Fatal(insertErr)
		}
	}
	insert("CURSOR-1", "inv_cursor_1", now.Add(-time.Hour))
	insert("CURSOR-2", "inv_cursor_2", now)
	first, err := service.ListInvestigationQueue(ctx, reviewer, application.InvestigationQueueFilter{Limit: 1})
	if err != nil || len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("第一页无效: page=%+v err=%v", first, err)
	}
	claimedID := first.Items[0].InvestigationID
	if _, err = service.ClaimInvestigation(ctx, claimedID, first.Items[0].Revision, reviewer); err != nil {
		t.Fatal(err)
	}
	second, err := service.ListInvestigationQueue(ctx, reviewer, application.InvestigationQueueFilter{Limit: 10, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range second.Items {
		if item.InvestigationID == claimedID {
			t.Fatalf("游标后状态变化的调查不得在后续页重复: %s", claimedID)
		}
	}
}
