package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

type InvestigationQueueFilter struct {
	Status           string
	TriggerCode      string
	AssignedReviewer string
	DetectedFrom     string
	DetectedTo       string
	Limit            int
	Cursor           string
}

type InvestigationQueueItem struct {
	InvestigationID  string                     `json:"investigation_id"`
	DossierID        string                     `json:"dossier_id"`
	SampleCode       string                     `json:"sample_code"`
	TriggerCodes     []string                   `json:"trigger_codes"`
	DetectedAt       time.Time                  `json:"detected_at"`
	Status           domain.InvestigationStatus `json:"status"`
	AssignedReviewer string                     `json:"assigned_reviewer,omitempty"`
	Revision         int64                      `json:"revision"`
}

type InvestigationQueuePage struct {
	Items      []InvestigationQueueItem `json:"items"`
	NextCursor string                   `json:"next_cursor,omitempty"`
}

type queueCursor struct {
	StatusRank int                  `json:"status_rank"`
	DetectedAt time.Time            `json:"detected_at"`
	ID         string               `json:"investigation_id"`
	Seen       []string             `json:"seen,omitempty"`
	seenSet    map[string]struct{} `json:"-"`
}

func queueRank(status domain.InvestigationStatus) int {
	if status == domain.InvestigationUnassigned {
		return 0
	}
	return 1
}

func parseQueueTime(field, value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, domain.FieldError(field, "时间必须使用 RFC3339 格式")
	}
	parsed = domain.NormalizeTime(parsed)
	return &parsed, nil
}

func validInvestigationStatus(value string) bool {
	for _, status := range []domain.InvestigationStatus{domain.InvestigationUnassigned, domain.InvestigationAssigned, domain.InvestigationActionSet, domain.InvestigationReview, domain.InvestigationResolved} {
		if value == string(status) {
			return true
		}
	}
	return false
}

func validTriggerCode(value string) bool {
	for _, code := range []string{chaincheck.CodeRouteDeviation, chaincheck.CodeTemperatureHigh, chaincheck.CodeTemperatureLow, chaincheck.CodeSealBroken, chaincheck.CodeSealMissing, chaincheck.CodeTransitTimeout} {
		if value == code {
			return true
		}
	}
	return false
}

func decodeQueueCursor(value string) (*queueCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, domain.FieldError("cursor", "游标无效")
	}
	var cursor queueCursor
	if err = json.Unmarshal(payload, &cursor); err != nil || cursor.ID == "" || cursor.DetectedAt.IsZero() || (cursor.StatusRank != 0 && cursor.StatusRank != 1) {
		return nil, domain.FieldError("cursor", "游标无效")
	}
	cursor.DetectedAt = domain.NormalizeTime(cursor.DetectedAt)
	seen := make(map[string]struct{}, len(cursor.Seen))
	for _, id := range cursor.Seen {
		if strings.TrimSpace(id) == "" {
			return nil, domain.FieldError("cursor", "游标无效")
		}
		seen[id] = struct{}{}
	}
	cursor.seenSet = seen
	return &cursor, nil
}

func encodeQueueCursor(item InvestigationQueueItem, seen []string) string {
	payload, _ := json.Marshal(queueCursor{StatusRank: queueRank(item.Status), DetectedAt: item.DetectedAt, ID: item.InvestigationID, Seen: seen})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func queueAfter(item InvestigationQueueItem, cursor *queueCursor) bool {
	rank := queueRank(item.Status)
	if rank != cursor.StatusRank {
		return rank > cursor.StatusRank
	}
	if !item.DetectedAt.Equal(cursor.DetectedAt) {
		return item.DetectedAt.After(cursor.DetectedAt)
	}
	return item.InvestigationID > cursor.ID
}

func (s *Service) ListInvestigationQueue(ctx context.Context, actor Actor, filter InvestigationQueueFilter) (*InvestigationQueuePage, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleReviewer); err != nil {
		return nil, err
	}
	filter.Status = domain.NormalizeText(filter.Status)
	filter.TriggerCode = strings.ToUpper(domain.NormalizeText(filter.TriggerCode))
	filter.AssignedReviewer = domain.NormalizeText(filter.AssignedReviewer)
	if filter.Status != "" && !validInvestigationStatus(filter.Status) {
		return nil, domain.FieldError("status", "调查状态无效")
	}
	if filter.TriggerCode != "" && !validTriggerCode(filter.TriggerCode) {
		return nil, domain.FieldError("trigger_code", "触发规则代码无效")
	}
	if filter.Limit == 0 {
		filter.Limit = 25
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		return nil, domain.FieldError("limit", "页大小必须在 1 到 100 之间")
	}
	from, err := parseQueueTime("detected_from", filter.DetectedFrom)
	if err != nil {
		return nil, err
	}
	to, err := parseQueueTime("detected_to", filter.DetectedTo)
	if err != nil {
		return nil, err
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, domain.FieldError("detected_to", "发现时间区间无效")
	}
	cursor, err := decodeQueueCursor(filter.Cursor)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListInvestigationDossiers(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]InvestigationQueueItem, 0, len(rows))
	for _, row := range rows {
		investigation := row.Investigation
		if filter.Status != "" && string(investigation.Status) != filter.Status {
			continue
		}
		if filter.AssignedReviewer != "" && investigation.AssignedReviewer != filter.AssignedReviewer {
			continue
		}
		if filter.TriggerCode != "" && !slices.Contains(investigation.TriggerCodes, filter.TriggerCode) {
			continue
		}
		if from != nil && investigation.DetectedAt.Before(*from) || to != nil && investigation.DetectedAt.After(*to) {
			continue
		}
		item := InvestigationQueueItem{InvestigationID: investigation.ID, DossierID: investigation.DossierID, SampleCode: row.Dossier.SampleCode,
			TriggerCodes: append([]string(nil), investigation.TriggerCodes...), DetectedAt: investigation.DetectedAt, Status: investigation.Status,
			AssignedReviewer: investigation.AssignedReviewer, Revision: investigation.Revision}
		if cursor != nil {
			if _, ok := cursor.seenSet[item.InvestigationID]; ok {
				continue
			}
			if !queueAfter(item, cursor) {
				continue
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		leftRank, rightRank := queueRank(items[i].Status), queueRank(items[j].Status)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if !items[i].DetectedAt.Equal(items[j].DetectedAt) {
			return items[i].DetectedAt.Before(items[j].DetectedAt)
		}
		return items[i].InvestigationID < items[j].InvestigationID
	})
	page := &InvestigationQueuePage{Items: []InvestigationQueueItem{}}
	if len(items) <= filter.Limit {
		page.Items = items
		return page, nil
	}
	page.Items = items[:filter.Limit]
	nextSeen := make([]string, 0, len(page.Items))
	if cursor != nil {
		nextSeen = append(nextSeen, cursor.Seen...)
	}
	for _, item := range page.Items {
		nextSeen = append(nextSeen, item.InvestigationID)
	}
	page.NextCursor = encodeQueueCursor(page.Items[len(page.Items)-1], nextSeen)
	return page, nil
}
