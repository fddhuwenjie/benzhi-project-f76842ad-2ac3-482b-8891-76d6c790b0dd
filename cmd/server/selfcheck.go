package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type checkEnvelope struct {
	Data json.RawMessage `json:"data"`
}
type checkDossier struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
}
type checkTransfer struct {
	Dossier       checkDossier `json:"dossier"`
	Investigation *struct {
		ID       string `json:"id"`
		Revision int64  `json:"revision"`
	} `json:"investigation"`
}
type checkEvidence struct {
	ID string `json:"id"`
}
type checkInvestigation struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
}

func runSelfcheck(ctx context.Context, address string) error {
	baseURL := "http://" + address
	client := &http.Client{Timeout: 3 * time.Second}
	now := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Second)
	draft := map[string]any{"sample_code": "SELF-CHECK-001", "site_name": "自检采样点", "medium": "地表水", "container_type": "玻璃瓶", "collected_at": now.Format(time.RFC3339), "required_temperature_min": 2, "required_temperature_max": 8, "maximum_transit_minutes": 120, "expected_route": []string{"FIELD", "LAB"}, "responsible_person": "整改责任人"}
	var dossier checkDossier
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/dossiers", draft, map[string]string{"X-Actor": "采样员", "X-Role": "custodian"}, http.StatusCreated, &dossier); err != nil {
		return err
	}
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/dossiers/"+dossier.ID+"/submit", map[string]any{"revision": dossier.Revision}, map[string]string{"X-Actor": "采样员", "X-Role": "custodian"}, http.StatusOK, &dossier); err != nil {
		return err
	}
	first := map[string]any{"revision": dossier.Revision, "station_code": "FIELD", "released_by": "采样员", "received_by": "运输员", "transferred_at": now.Add(10 * time.Minute).Format(time.RFC3339), "observed_temperature": 4.5, "seal_state": "intact"}
	var transfer checkTransfer
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/dossiers/"+dossier.ID+"/transfers", first, map[string]string{"X-Actor": "采样员", "X-Role": "custodian", "Idempotency-Key": "selfcheck-transfer-1"}, http.StatusCreated, &transfer); err != nil {
		return err
	}
	dossier = transfer.Dossier
	second := map[string]any{"revision": dossier.Revision, "station_code": "LAB", "released_by": "运输员", "received_by": "接收员", "transferred_at": now.Add(20 * time.Minute).Format(time.RFC3339), "observed_temperature": 12.0, "seal_state": "intact"}
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/dossiers/"+dossier.ID+"/transfers", second, map[string]string{"X-Actor": "接收员", "X-Role": "receiver", "Idempotency-Key": "selfcheck-transfer-2"}, http.StatusCreated, &transfer); err != nil {
		return err
	}
	if transfer.Investigation == nil {
		return fmt.Errorf("自检交接未触发预期温控异常")
	}
	dossier = transfer.Dossier
	inv := checkInvestigation{ID: transfer.Investigation.ID, Revision: transfer.Investigation.Revision}
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/investigations/"+inv.ID+"/claim", map[string]any{"revision": inv.Revision}, map[string]string{"X-Actor": "质控员", "X-Role": "quality_reviewer"}, http.StatusOK, &inv); err != nil {
		return err
	}
	conclusion := map[string]any{"revision": inv.Revision, "root_cause": "运输冷藏介质失效", "impact_assessment": "短时超温，复测可确认有效性", "required_action": "补充温度记录并完成复测"}
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/investigations/"+inv.ID+"/conclusion", conclusion, map[string]string{"X-Actor": "质控员", "X-Role": "quality_reviewer"}, http.StatusOK, &inv); err != nil {
		return err
	}
	evidenceInput := map[string]any{"revision": inv.Revision, "description": "已补充连续温度记录并完成复测", "media_type": "application/pdf", "content_digest": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	var evidence checkEvidence
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/investigations/"+inv.ID+"/evidence", evidenceInput, map[string]string{"X-Actor": "整改责任人", "X-Role": "responsible_person"}, http.StatusCreated, &evidence); err != nil {
		return err
	}
	inv.Revision++
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/evidence/"+evidence.ID+"/review", map[string]any{"revision": inv.Revision, "decision": "accepted", "comment": "证据与复测结果完整"}, map[string]string{"X-Actor": "质控员", "X-Role": "quality_reviewer"}, http.StatusOK, &evidence); err != nil {
		return err
	}
	var view struct {
		Dossier checkDossier `json:"dossier"`
	}
	if err := checkRequest(ctx, client, "GET", baseURL+"/api/v1/dossiers/"+dossier.ID, nil, nil, http.StatusOK, &view); err != nil {
		return err
	}
	dossier = view.Dossier
	if err := checkRequest(ctx, client, "POST", baseURL+"/api/v1/dossiers/"+dossier.ID+"/close", map[string]any{"revision": dossier.Revision}, map[string]string{"X-Actor": "质控员", "X-Role": "quality_reviewer"}, http.StatusOK, &dossier); err != nil {
		return err
	}
	var history []json.RawMessage
	if err := checkRequest(ctx, client, "GET", baseURL+"/api/v1/dossiers/"+dossier.ID+"/audit", nil, nil, http.StatusOK, &history); err != nil {
		return err
	}
	if len(history) < 8 {
		return fmt.Errorf("自检审计事件不足: %d", len(history))
	}
	return nil
}

func checkRequest(ctx context.Context, client *http.Client, method, url string, body any, headers map[string]string, wantStatus int, target any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, url, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != wantStatus {
		return fmt.Errorf("%s %s 返回 %d，期望 %d: %s", method, url, response.StatusCode, wantStatus, string(payload))
	}
	if target == nil {
		return nil
	}
	var envelope checkEnvelope
	if err = json.Unmarshal(payload, &envelope); err != nil {
		return err
	}
	if err = json.Unmarshal(envelope.Data, target); err != nil {
		return fmt.Errorf("解析自检响应: %w", err)
	}
	return nil
}
