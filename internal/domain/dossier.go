package domain

import (
	"reflect"
	"slices"
	"time"
)

type SampleDossier struct {
	ID                     string        `json:"id"`
	SampleCode             string        `json:"sample_code"`
	SiteName               string        `json:"site_name"`
	Medium                 string        `json:"medium"`
	ContainerType          string        `json:"container_type"`
	CollectedAt            time.Time     `json:"collected_at"`
	RequiredTemperatureMin float64       `json:"required_temperature_min"`
	RequiredTemperatureMax float64       `json:"required_temperature_max"`
	MaximumTransitMinutes  int           `json:"maximum_transit_minutes"`
	ExpectedRoute          []string      `json:"expected_route"`
	ResponsiblePerson      string        `json:"responsible_person"`
	Status                 DossierStatus `json:"status"`
	Revision               int64         `json:"revision"`
	CreatedAt              time.Time     `json:"created_at"`
	SubmittedAt            *time.Time    `json:"submitted_at,omitempty"`
	ClosedAt               *time.Time    `json:"closed_at,omitempty"`
}

type DraftInput struct {
	SampleCode             string    `json:"sample_code"`
	SiteName               string    `json:"site_name"`
	Medium                 string    `json:"medium"`
	ContainerType          string    `json:"container_type"`
	CollectedAt            time.Time `json:"collected_at"`
	RequiredTemperatureMin float64   `json:"required_temperature_min"`
	RequiredTemperatureMax float64   `json:"required_temperature_max"`
	MaximumTransitMinutes  int       `json:"maximum_transit_minutes"`
	ExpectedRoute          []string  `json:"expected_route"`
	ResponsiblePerson      string    `json:"responsible_person"`
}

type FieldChange struct {
	Before any `json:"before"`
	After  any `json:"after"`
}

func NewDossier(id string, input DraftInput, now time.Time) (*SampleDossier, error) {
	if NormalizeText(input.SampleCode) == "" {
		return nil, FieldError("sample_code", "样品编号不能为空")
	}
	if NormalizeText(id) == "" {
		return nil, FieldError("id", "档案标识不能为空")
	}
	route := make([]string, len(input.ExpectedRoute))
	for i, station := range input.ExpectedRoute {
		route[i] = NormalizeStation(station)
	}
	return &SampleDossier{
		ID: id, SampleCode: NormalizeText(input.SampleCode), SiteName: NormalizeText(input.SiteName),
		Medium: NormalizeText(input.Medium), ContainerType: NormalizeText(input.ContainerType),
		CollectedAt: NormalizeTime(input.CollectedAt), RequiredTemperatureMin: input.RequiredTemperatureMin,
		RequiredTemperatureMax: input.RequiredTemperatureMax, MaximumTransitMinutes: input.MaximumTransitMinutes,
		ExpectedRoute: route, ResponsiblePerson: NormalizeText(input.ResponsiblePerson), Status: DossierDraft,
		Revision: 1, CreatedAt: NormalizeTime(now),
	}, nil
}

func NormalizeDraftInput(input DraftInput) DraftInput {
	route := make([]string, len(input.ExpectedRoute))
	for index, station := range input.ExpectedRoute {
		route[index] = NormalizeStation(station)
	}
	input.SampleCode = NormalizeText(input.SampleCode)
	input.SiteName = NormalizeText(input.SiteName)
	input.Medium = NormalizeText(input.Medium)
	input.ContainerType = NormalizeText(input.ContainerType)
	input.CollectedAt = NormalizeTime(input.CollectedAt)
	input.ExpectedRoute = route
	input.ResponsiblePerson = NormalizeText(input.ResponsiblePerson)
	return input
}

func (d *SampleDossier) DraftInput() DraftInput {
	return DraftInput{SampleCode: d.SampleCode, SiteName: d.SiteName, Medium: d.Medium, ContainerType: d.ContainerType,
		CollectedAt: d.CollectedAt, RequiredTemperatureMin: d.RequiredTemperatureMin, RequiredTemperatureMax: d.RequiredTemperatureMax,
		MaximumTransitMinutes: d.MaximumTransitMinutes, ExpectedRoute: append([]string(nil), d.ExpectedRoute...), ResponsiblePerson: d.ResponsiblePerson}
}

func (d *SampleDossier) ReviseDraft(expected int64, input DraftInput) (map[string]FieldChange, error) {
	if err := d.CheckRevision(expected); err != nil {
		return nil, err
	}
	if d.Status != DossierDraft {
		return nil, NewError(CodeState, "仅草稿档案可以修改")
	}
	input = NormalizeDraftInput(input)
	if input.SampleCode == "" {
		return nil, FieldError("sample_code", "样品编号不能为空")
	}
	changes := map[string]FieldChange{}
	add := func(field string, before, after any) {
		if !reflect.DeepEqual(before, after) {
			changes[field] = FieldChange{Before: before, After: after}
		}
	}
	add("sample_code", d.SampleCode, input.SampleCode)
	add("site_name", d.SiteName, input.SiteName)
	add("medium", d.Medium, input.Medium)
	add("container_type", d.ContainerType, input.ContainerType)
	add("collected_at", d.CollectedAt, input.CollectedAt)
	add("required_temperature_min", d.RequiredTemperatureMin, input.RequiredTemperatureMin)
	add("required_temperature_max", d.RequiredTemperatureMax, input.RequiredTemperatureMax)
	add("maximum_transit_minutes", d.MaximumTransitMinutes, input.MaximumTransitMinutes)
	add("expected_route", d.ExpectedRoute, input.ExpectedRoute)
	add("responsible_person", d.ResponsiblePerson, input.ResponsiblePerson)
	d.SampleCode, d.SiteName, d.Medium, d.ContainerType = input.SampleCode, input.SiteName, input.Medium, input.ContainerType
	d.CollectedAt, d.RequiredTemperatureMin, d.RequiredTemperatureMax = input.CollectedAt, input.RequiredTemperatureMin, input.RequiredTemperatureMax
	d.MaximumTransitMinutes, d.ExpectedRoute, d.ResponsiblePerson = input.MaximumTransitMinutes, input.ExpectedRoute, input.ResponsiblePerson
	d.Revision++
	return changes, nil
}

func (d *SampleDossier) CheckRevision(expected int64) error {
	if expected <= 0 {
		return FieldError("revision", "必须提供正数修订号")
	}
	if d.Revision != expected {
		return NewError(CodeConflict, "档案已被其他请求修改")
	}
	return nil
}

func (d *SampleDossier) EnsureMutable() error {
	if d.Status == DossierClosed {
		return NewError(CodeClosed, "档案已归档，禁止业务修改")
	}
	return nil
}

func (d *SampleDossier) Submit(expected int64, now time.Time) error {
	if err := d.CheckRevision(expected); err != nil {
		return err
	}
	if d.Status != DossierDraft {
		return NewError(CodeState, "仅草稿档案可以提交")
	}
	if issues := d.SubmissionIssues(now); len(issues) > 0 {
		return issues[0].Error()
	}
	at := NormalizeTime(now)
	d.SubmittedAt = &at
	d.Status = DossierSubmitted
	d.Revision++
	return nil
}

func (d *SampleDossier) ApplyTransferBatch(expected int64, anomalous bool) error {
	if err := d.EnsureMutable(); err != nil {
		return err
	}
	if err := d.CheckRevision(expected); err != nil {
		return err
	}
	if !slices.Contains([]DossierStatus{DossierSubmitted, DossierInTransit, DossierInvestigation}, d.Status) {
		return NewError(CodeState, "当前档案状态不能登记交接")
	}
	if anomalous {
		d.Status = DossierInvestigation
	} else {
		d.Status = DossierInTransit
	}
	d.Revision++
	return nil
}

func (d *SampleDossier) ApplyTransfer(expected int64, anomalous bool) error {
	if err := d.EnsureMutable(); err != nil {
		return err
	}
	if err := d.CheckRevision(expected); err != nil {
		return err
	}
	if !slices.Contains([]DossierStatus{DossierSubmitted, DossierInTransit, DossierInvestigation}, d.Status) {
		return NewError(CodeState, "当前档案状态不能登记交接")
	}
	if anomalous {
		d.Status = DossierInvestigation
	} else {
		d.Status = DossierInTransit
	}
	d.Revision++
	return nil
}

func (d *SampleDossier) SetStatus(status DossierStatus) {
	d.Status = status
	d.Revision++
}

func (d *SampleDossier) Close(expected int64, now time.Time) error {
	if err := d.CheckRevision(expected); err != nil {
		return err
	}
	if d.Status != DossierReadyToClose {
		return NewError(CodeState, "档案尚未满足关闭条件")
	}
	at := NormalizeTime(now)
	d.ClosedAt = &at
	d.Status = DossierClosed
	d.Revision++
	return nil
}
