package domain

import (
	"math"
	"sort"
	"time"
)

type ValidationIssue struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (i ValidationIssue) Error() error { return FieldError(i.Field, i.Message) }

func (d *SampleDossier) SubmissionIssues(now time.Time) []ValidationIssue {
	var issues []ValidationIssue
	add := func(field, code, message string) {
		issues = append(issues, ValidationIssue{Field: field, Code: code, Message: message})
	}
	if NormalizeText(d.SampleCode) == "" {
		add("sample_code", "REQUIRED", "样品编号不能为空")
	}
	required := []struct{ name, value string }{
		{"site_name", d.SiteName}, {"medium", d.Medium}, {"container_type", d.ContainerType},
		{"responsible_person", d.ResponsiblePerson},
	}
	for _, item := range required {
		if NormalizeText(item.value) == "" {
			add(item.name, "REQUIRED", "字段不能为空")
		}
	}
	if d.CollectedAt.IsZero() {
		add("collected_at", "REQUIRED", "采样时间不能为空")
	} else if d.CollectedAt.After(NormalizeTime(now).Add(5 * time.Minute)) {
		add("collected_at", "COLLECTED_AT_FUTURE", "采样时间不能晚于当前时间")
	}
	if math.IsNaN(d.RequiredTemperatureMin) || math.IsNaN(d.RequiredTemperatureMax) || d.RequiredTemperatureMin > d.RequiredTemperatureMax {
		add("required_temperature_min", "TEMPERATURE_RANGE_INVALID", "保存温度范围无效")
	}
	if d.RequiredTemperatureMin < -100 || d.RequiredTemperatureMax > 100 {
		add("required_temperature_max", "TEMPERATURE_OUT_OF_RANGE", "保存温度超出可接受范围")
	}
	if d.MaximumTransitMinutes <= 0 || d.MaximumTransitMinutes > 43200 {
		add("maximum_transit_minutes", "TRANSIT_LIMIT_INVALID", "最大流转时限必须在 1 到 43200 分钟之间")
	}
	if len(d.ExpectedRoute) < 2 {
		add("expected_route", "ROUTE_TOO_SHORT", "预期路线至少包含两个站点")
	}
	seen := map[string]bool{}
	for index, station := range d.ExpectedRoute {
		if station == "" {
			add("expected_route", "ROUTE_STATION_EMPTY", "路线站点不能为空")
			continue
		}
		if seen[station] {
			add("expected_route", "ROUTE_STATION_DUPLICATE", "路线站点不能重复: "+station)
			_ = index
		}
		seen[station] = true
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Field != issues[j].Field {
			return issues[i].Field < issues[j].Field
		}
		return issues[i].Code < issues[j].Code
	})
	return issues

}

func (d *SampleDossier) ValidateForSubmission(now time.Time) error {
	issues := d.SubmissionIssues(now)
	if len(issues) == 0 {
		return nil
	}
	return issues[0].Error()
}
