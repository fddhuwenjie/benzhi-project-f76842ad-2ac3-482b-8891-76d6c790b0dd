package anomaly_transfer_continuation_test

import (
	"context"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestAnomalousTransferDoesNotBlockRemainingRoute(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service := application.New(store, chaincheck.New())
	ctx := context.Background()
	actor := application.Actor{Name: "保管员", Role: domain.RoleCustodian}
	collectedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	dossier, err := service.CreateDossier(ctx, actor, domain.DraftInput{
		SampleCode: "EARLY-ANOMALY", SiteName: "站点", Medium: "水", ContainerType: "瓶",
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
	first, err := service.RegisterTransfer(ctx, dossier.ID, "early-anomaly-1", dossier.Revision, actor, domain.TransferInput{
		StationCode: "A", ReleasedBy: "甲", ReceivedBy: "乙", TransferredAt: collectedAt.Add(10 * time.Minute),
		ObservedTemperature: 12, SealState: domain.SealIntact,
	})
	if err != nil || first.Investigation == nil {
		t.Fatalf("首站异常应被登记并创建调查: result=%+v err=%v", first, err)
	}
	second, err := service.RegisterTransfer(ctx, dossier.ID, "early-anomaly-2", first.Dossier.Revision, actor, domain.TransferInput{
		StationCode: "B", ReleasedBy: "乙", ReceivedBy: "丙", TransferredAt: collectedAt.Add(20 * time.Minute),
		ObservedTemperature: 4, SealState: domain.SealIntact,
	})
	if err != nil {
		t.Fatalf("异常调查存在时仍应允许登记后续监管链事实: %v", err)
	}
	view, err := service.GetDossier(ctx, dossier.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Transfers) != 2 || view.Investigation == nil || second.Dossier.ID != dossier.ID {
		t.Fatalf("后续交接应与既有调查共同保留: %+v", view)
	}
}
