package application

import (
	"context"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestTransferIdempotencyAndConflict(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store, chaincheck.New())
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Hour)
	d, err := service.CreateDossier(ctx, Actor{Name: "采样员", Role: domain.RoleCustodian}, domain.DraftInput{SampleCode: "IDEM-1", SiteName: "站点", Medium: "水", ContainerType: "瓶", CollectedAt: now, RequiredTemperatureMin: 2, RequiredTemperatureMax: 8, MaximumTransitMinutes: 120, ExpectedRoute: []string{"A", "B"}, ResponsiblePerson: "责任人"})
	if err != nil {
		t.Fatal(err)
	}
	d, err = service.SubmitDossier(ctx, d.ID, d.Revision, Actor{Name: "采样员", Role: domain.RoleCustodian})
	if err != nil {
		t.Fatal(err)
	}
	input := domain.TransferInput{StationCode: "A", ReleasedBy: "甲", ReceivedBy: "乙", TransferredAt: now.Add(time.Minute), ObservedTemperature: 4, SealState: domain.SealIntact}
	first, err := service.RegisterTransfer(ctx, d.ID, "request-1", d.Revision, Actor{Name: "采样员", Role: domain.RoleCustodian}, input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.RegisterTransfer(ctx, d.ID, "request-1", d.Revision, Actor{Name: "采样员", Role: domain.RoleCustodian}, input)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.IdempotentReplay || replay.Transfer.ID != first.Transfer.ID {
		t.Fatal("重复请求未返回原始结果")
	}
	input.ObservedTemperature = 5
	if _, err = service.RegisterTransfer(ctx, d.ID, "request-1", d.Revision, Actor{Name: "采样员", Role: domain.RoleCustodian}, input); !domain.IsCode(err, domain.CodeIdempotency) {
		t.Fatalf("冲突载荷错误: %v", err)
	}
}
