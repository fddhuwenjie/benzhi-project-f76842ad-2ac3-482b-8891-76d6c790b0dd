package application

import (
	"context"
	"sort"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

type SubmissionPreview struct {
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

type SubmissionPreflight struct {
	CanSubmit bool                     `json:"can_submit"`
	Revision  int64                    `json:"revision"`
	Preview   SubmissionPreview        `json:"normalized_preview"`
	Issues    []domain.ValidationIssue `json:"issues"`
}

type DraftPatch struct {
	SampleCode             *string
	SiteName               *string
	Medium                 *string
	ContainerType          *string
	CollectedAt            *time.Time
	RequiredTemperatureMin *float64
	RequiredTemperatureMax *float64
	MaximumTransitMinutes  *int
	ExpectedRoute          *[]string
	ResponsiblePerson      *string
}

func (p DraftPatch) apply(input domain.DraftInput) domain.DraftInput {
	result := input
	if p.SampleCode != nil {
		result.SampleCode = *p.SampleCode
	}
	if p.SiteName != nil {
		result.SiteName = *p.SiteName
	}
	if p.Medium != nil {
		result.Medium = *p.Medium
	}
	if p.ContainerType != nil {
		result.ContainerType = *p.ContainerType
	}
	if p.CollectedAt != nil {
		result.CollectedAt = *p.CollectedAt
	}
	if p.RequiredTemperatureMin != nil {
		result.RequiredTemperatureMin = *p.RequiredTemperatureMin
	}
	if p.RequiredTemperatureMax != nil {
		result.RequiredTemperatureMax = *p.RequiredTemperatureMax
	}
	if p.MaximumTransitMinutes != nil {
		result.MaximumTransitMinutes = *p.MaximumTransitMinutes
	}
	if p.ExpectedRoute != nil {
		result.ExpectedRoute = append([]string(nil), (*p.ExpectedRoute)...)
	}
	if p.ResponsiblePerson != nil {
		result.ResponsiblePerson = *p.ResponsiblePerson
	}
	return result
}

func (s *Service) CreateDossier(ctx context.Context, actor Actor, input domain.DraftInput) (*domain.SampleDossier, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleCustodian); err != nil {
		return nil, err
	}
	dossier, err := domain.NewDossier(s.newID("dos"), input, s.now())
	if err != nil {
		return nil, err
	}
	err = s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		if err := tx.InsertDossier(ctx, dossier); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.NewAudit(dossier.ID, "dossier.created", actor.Name, actor.Role, dossier.Revision, map[string]any{"sample_code": dossier.SampleCode}, s.now()))
	})
	if err != nil {
		return nil, err
	}
	return dossier, nil
}

func (s *Service) ReviseDossier(ctx context.Context, id string, revision int64, actor Actor, input domain.DraftInput) (*domain.SampleDossier, error) {
	return s.reviseDossier(ctx, id, revision, actor, func(domain.DraftInput) domain.DraftInput { return input })
}

func (s *Service) ReviseDossierPatch(ctx context.Context, id string, revision int64, actor Actor, patch DraftPatch) (*domain.SampleDossier, error) {
	return s.reviseDossier(ctx, id, revision, actor, func(current domain.DraftInput) domain.DraftInput {
		return patch.apply(current)
	})
}

func (s *Service) reviseDossier(ctx context.Context, id string, revision int64, actor Actor, buildInput func(domain.DraftInput) domain.DraftInput) (*domain.SampleDossier, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleCustodian); err != nil {
		return nil, err
	}
	var dossier *domain.SampleDossier
	err := s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		var err error
		dossier, err = tx.GetDossier(ctx, id)
		if err != nil {
			return err
		}
		input := buildInput(dossier.DraftInput())
		input = domain.NormalizeDraftInput(input)
		if input.SampleCode == "" {
			return domain.FieldError("sample_code", "样品编号不能为空")
		}
		exists, err := tx.SampleCodeExists(ctx, input.SampleCode, id)
		if err != nil {
			return err
		}
		if exists {
			return domain.FieldError("sample_code", "样品编号已被其他档案使用")
		}
		previous := dossier.Revision
		changes, err := dossier.ReviseDraft(revision, input)
		if err != nil {
			return err
		}
		if err = tx.UpdateDossier(ctx, dossier, previous); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.NewAudit(id, "dossier.draft_revised", actor.Name, actor.Role, dossier.Revision, map[string]any{"changes": changes}, s.now()))
	})
	if err != nil {
		return nil, err
	}
	return dossier, nil
}

func (s *Service) PreflightSubmission(ctx context.Context, id string, revision int64, actor Actor) (*SubmissionPreflight, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleCustodian); err != nil {
		return nil, err
	}
	dossier, err := s.store.GetDossier(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = dossier.CheckRevision(revision); err != nil {
		return nil, err
	}
	if dossier.Status != domain.DossierDraft {
		return nil, domain.NewError(domain.CodeState, "仅草稿档案可以执行提交预检")
	}
	issues := dossier.SubmissionIssues(s.now())
	if issues == nil {
		issues = []domain.ValidationIssue{}
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Field != issues[j].Field {
			return issues[i].Field < issues[j].Field
		}
		return issues[i].Code < issues[j].Code
	})
	input := dossier.DraftInput()
	return &SubmissionPreflight{CanSubmit: len(issues) == 0, Revision: dossier.Revision,
		Preview: SubmissionPreview{SampleCode: input.SampleCode, SiteName: input.SiteName, Medium: input.Medium, ContainerType: input.ContainerType,
			CollectedAt: input.CollectedAt, RequiredTemperatureMin: input.RequiredTemperatureMin, RequiredTemperatureMax: input.RequiredTemperatureMax,
			MaximumTransitMinutes: input.MaximumTransitMinutes, ExpectedRoute: input.ExpectedRoute, ResponsiblePerson: input.ResponsiblePerson}, Issues: issues}, nil
}

func (s *Service) SubmitDossier(ctx context.Context, id string, revision int64, actor Actor) (*domain.SampleDossier, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleCustodian); err != nil {
		return nil, err
	}
	var dossier *domain.SampleDossier
	err := s.store.WithTx(ctx, func(tx *persistence.Tx) error {
		var err error
		dossier, err = tx.GetDossier(ctx, id)
		if err != nil {
			return err
		}
		previous := dossier.Revision
		if err = dossier.Submit(revision, s.now()); err != nil {
			return err
		}
		if err = tx.UpdateDossier(ctx, dossier, previous); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.NewAudit(id, "dossier.submitted", actor.Name, actor.Role, dossier.Revision, map[string]any{"route": dossier.ExpectedRoute}, s.now()))
	})
	if err != nil {
		return nil, err
	}
	return dossier, nil
}
