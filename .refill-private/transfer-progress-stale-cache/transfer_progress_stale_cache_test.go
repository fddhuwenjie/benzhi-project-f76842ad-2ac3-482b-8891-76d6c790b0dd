package transferprogressstalecache_test

import (
	"context"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestTransferProgressCacheInvalidatesAfterMutation(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.New(store, chaincheck.New())
	ctx := context.Background()
	actor := application.Actor{Name: "保管员", Role: domain.RoleCustodian}
	collectedAt := time.Now().UTC().Add(-10 * time.Minute)
	dossier, err := service.CreateDossier(ctx, actor, domain.DraftInput{
		SampleCode: "PROGRESS-CACHE-1", SiteName: "河口站", Medium: "水", ContainerType: "玻璃瓶",
		CollectedAt: collectedAt, RequiredTemperatureMin: 2, RequiredTemperatureMax: 8,
		MaximumTransitMinutes: 120, ExpectedRoute: []string{"A", "B"}, ResponsiblePerson: "责任人",
	})
	if err != nil {
		t.Fatal(err)
	}
	dossier, err = service.SubmitDossier(ctx, dossier.ID, dossier.Revision, actor)
	if err != nil {
		t.Fatal(err)
	}

	before, err := service.GetTransferProgress(ctx, dossier.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before.CompletedStations != 0 || before.NextStation != "A" {
		t.Fatalf("初始进度错误: %+v", before)
	}
	result, err := service.RegisterTransfer(ctx, dossier.ID, "progress-cache-request", dossier.Revision, actor, domain.TransferInput{
		StationCode: "A", ReleasedBy: "甲", ReceivedBy: "乙", TransferredAt: collectedAt.Add(time.Minute),
		ObservedTemperature: 4, SealState: domain.SealIntact,
	})
	if err != nil {
		t.Fatal(err)
	}

	after, err := service.GetTransferProgress(ctx, dossier.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.DossierRevision != result.Dossier.Revision || after.CompletedStations != 1 || after.NextStation != "B" || after.CurrentCustodian != "乙" {
		t.Fatalf("TestTransferProgressCacheInvalidatesAfterMutation: 交接提交后返回了旧进度: before=%+v after=%+v dossier_revision=%d", before, after, result.Dossier.Revision)
	}
}
