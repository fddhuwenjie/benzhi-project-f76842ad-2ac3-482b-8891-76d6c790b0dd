package investigation_error_chain_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/httpapi"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestStaleClaimPreservesConflictErrorChain(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	dossier, err := domain.NewDossier("dos_error_chain", domain.DraftInput{
		SampleCode:             "ERROR-CHAIN-1",
		SiteName:               "河口站",
		Medium:                 "水",
		ContainerType:          "玻璃瓶",
		CollectedAt:            now.Add(-time.Minute),
		RequiredTemperatureMin: 2,
		RequiredTemperatureMax: 8,
		MaximumTransitMinutes:  120,
		ExpectedRoute:          []string{"FIELD", "LAB"},
		ResponsiblePerson:      "责任人",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	investigation := domain.NewInvestigation("inv_error_chain", dossier.ID, []string{chaincheck.CodeSealBroken}, now)
	if err = store.WithTx(context.Background(), func(tx *persistence.Tx) error {
		if insertErr := tx.InsertDossier(context.Background(), dossier); insertErr != nil {
			return insertErr
		}
		return tx.InsertInvestigation(context.Background(), investigation)
	}); err != nil {
		t.Fatal(err)
	}

	handler := httpapi.New(application.New(store, chaincheck.New())).Handler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/investigations/"+investigation.ID+"/claim", strings.NewReader(`{"revision":99}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Actor", "审核员")
	request.Header.Set("X-Role", domain.RoleReviewer)
	request.Header.Set("If-Match", `"99"`)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("过期调查修订号应保留 CONFLICT 错误链并返回 409，实际状态码为 %d，响应为 %s", response.Code, response.Body.String())
	}
}
