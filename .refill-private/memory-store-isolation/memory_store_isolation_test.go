package memory_store_isolation

import (
	"context"
	"errors"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestMemoryStoresAreIsolated(t *testing.T) {
	ctx := context.Background()
	first, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	service := application.New(first, chaincheck.New())
	dossier, err := service.CreateDossier(ctx, application.Actor{Name: "采样员", Role: domain.RoleCustodian}, domain.DraftInput{
		SampleCode:             "MEMORY-ISOLATION-1",
		SiteName:               "采样点",
		Medium:                 "水",
		ContainerType:          "玻璃瓶",
		CollectedAt:            time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
		RequiredTemperatureMin: 2,
		RequiredTemperatureMax: 8,
		MaximumTransitMinutes:  120,
		ExpectedRoute:          []string{"A"},
		ResponsiblePerson:      "责任人",
	})
	if err != nil {
		t.Fatal(err)
	}

	second, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	_, err = second.GetDossier(ctx, dossier.ID)
	if err == nil {
		t.Fatalf("独立内存 Store 不应看到另一个 Store 的档案 %s", dossier.ID)
	}
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != domain.CodeNotFound {
		t.Fatalf("期望隔离 Store 返回 NOT_FOUND，得到 %v", err)
	}
}
