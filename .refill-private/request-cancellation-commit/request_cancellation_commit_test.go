package requestcancellationcommit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/httpapi"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestCanceledMutationDoesNotCommit(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := application.New(store, chaincheck.New())
	collectedAt := time.Now().UTC().Add(-time.Hour)
	dossier, err := service.CreateDossier(context.Background(), application.Actor{Name: "采样员", Role: domain.RoleCustodian}, domain.DraftInput{
		SampleCode: "CANCELED-001", SiteName: "原始站点", Medium: "水", ContainerType: "玻璃瓶",
		CollectedAt: collectedAt, RequiredTemperatureMin: 2, RequiredTemperatureMax: 8,
		MaximumTransitMinutes: 120, ExpectedRoute: []string{"FIELD", "LAB"}, ResponsiblePerson: "责任人",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := httpapi.New(service)
	body, err := json.Marshal(map[string]any{
		"revision":  dossier.Revision,
		"site_name": "取消后不应写入的站点",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/dossiers/"+dossier.ID, bytes.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor", "采样员")
	req.Header.Set("X-Role", domain.RoleCustodian)
	recorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, req)

	stored, err := store.GetDossier(context.Background(), dossier.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recorder.Code == http.StatusOK || stored.Revision != dossier.Revision || stored.SiteName != "原始站点" {
		t.Fatalf("取消请求仍返回 %d 并提交档案修订号 %d、站点 %q", recorder.Code, stored.Revision, stored.SiteName)
	}
}
