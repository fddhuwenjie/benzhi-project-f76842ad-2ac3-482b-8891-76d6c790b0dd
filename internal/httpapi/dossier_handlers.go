package httpapi

import (
	"net/http"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	if err := a.service.Ping(r.Context()); err != nil {
		fail(w, r, err)
		return
	}
	success(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) CreateDossier(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var input domain.DraftInput
	if err = decodeJSON(w, r, &input); err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.CreateDossier(r.Context(), who, input)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusCreated, result)
}

type reviseDossierRequest struct {
	Revision               int64       `json:"revision"`
	SampleCode             *string     `json:"sample_code"`
	SiteName               *string     `json:"site_name"`
	Medium                 *string     `json:"medium"`
	ContainerType          *string     `json:"container_type"`
	CollectedAt            *domainTime `json:"collected_at"`
	RequiredTemperatureMin *float64    `json:"required_temperature_min"`
	RequiredTemperatureMax *float64    `json:"required_temperature_max"`
	MaximumTransitMinutes  *int        `json:"maximum_transit_minutes"`
	ExpectedRoute          *[]string   `json:"expected_route"`
	ResponsiblePerson      *string     `json:"responsible_person"`
}

func (a *API) ReviseDossier(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var request reviseDossierRequest
	if err = decodeJSON(w, r, &request); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, request.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	patch := application.DraftPatch{SampleCode: request.SampleCode, SiteName: request.SiteName, Medium: request.Medium, ContainerType: request.ContainerType,
		RequiredTemperatureMin: request.RequiredTemperatureMin, RequiredTemperatureMax: request.RequiredTemperatureMax,
		MaximumTransitMinutes: request.MaximumTransitMinutes, ExpectedRoute: request.ExpectedRoute, ResponsiblePerson: request.ResponsiblePerson}
	if request.CollectedAt != nil {
		patch.CollectedAt = &request.CollectedAt.Time
	}
	result, err := a.service.ReviseDossierPatch(r.Context(), r.PathValue("dossier_id"), rev, who, patch)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusOK, result)
}

func (a *API) PreflightSubmission(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var command application.RevisionCommand
	if err = decodeJSON(w, r, &command); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, command.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.PreflightSubmission(r.Context(), r.PathValue("dossier_id"), rev, who)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusOK, result)
}

func (a *API) GetDossier(w http.ResponseWriter, r *http.Request) {
	result, err := a.service.GetDossier(r.Context(), r.PathValue("dossier_id"))
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Dossier.Revision)
	success(w, r, http.StatusOK, result)
}

func (a *API) SubmitDossier(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var command application.RevisionCommand
	if err = decodeJSON(w, r, &command); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, command.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.SubmitDossier(r.Context(), r.PathValue("dossier_id"), rev, who)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusOK, result)
}

func (a *API) CloseDossier(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var command application.RevisionCommand
	if err = decodeJSON(w, r, &command); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, command.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.CloseDossier(r.Context(), r.PathValue("dossier_id"), rev, who)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusOK, result)
}

func (a *API) PreflightClose(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.PreflightClose(r.Context(), r.PathValue("dossier_id"), who)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusOK, result)
}

func (a *API) GetAudit(w http.ResponseWriter, r *http.Request) {
	result, err := a.service.GetAudit(r.Context(), r.PathValue("dossier_id"))
	if err != nil {
		fail(w, r, err)
		return
	}
	success(w, r, http.StatusOK, result)
}
