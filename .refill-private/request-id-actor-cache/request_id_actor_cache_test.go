package request_id_actor_cache_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/httpapi"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func TestRequestIDDoesNotReuseActorIdentity(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	handler := httpapi.New(application.New(store, chaincheck.New())).Handler()
	request := func(actorName, role string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/investigations/queue", nil)
		req.Header.Set("X-Request-ID", "reused-correlation-id")
		req.Header.Set("X-Actor", actorName)
		req.Header.Set("X-Role", role)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	first := request("审核员甲", domain.RoleReviewer)
	if first.Code != http.StatusOK {
		t.Fatalf("审核员预置请求应成功，实际状态码 %d", first.Code)
	}
	second := request("保管员乙", domain.RoleCustodian)
	if second.Code != http.StatusForbidden {
		t.Fatalf("复用 X-Request-ID 不得复用上一请求身份：期望状态码 %d，实际状态码 %d，响应 %s", http.StatusForbidden, second.Code, second.Body.String())
	}
}
