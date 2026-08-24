package httpapi

import (
	"net/http"
	"strconv"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

func (a *API) ListInvestigationQueue(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	limit := 0
	if value := r.URL.Query().Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil {
			fail(w, r, domain.FieldError("limit", "页大小必须是整数"))
			return
		}
	}
	query := r.URL.Query()
	result, err := a.service.ListInvestigationQueue(r.Context(), who, application.InvestigationQueueFilter{
		Status: query.Get("status"), TriggerCode: query.Get("trigger_code"), AssignedReviewer: query.Get("assigned_reviewer"),
		DetectedFrom: query.Get("detected_from"), DetectedTo: query.Get("detected_to"), Limit: limit, Cursor: query.Get("cursor"),
	})
	if err != nil {
		fail(w, r, err)
		return
	}
	success(w, r, http.StatusOK, result)
}

func (a *API) ClaimInvestigation(w http.ResponseWriter, r *http.Request) {
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
	result, err := a.service.ClaimInvestigation(r.Context(), r.PathValue("investigation_id"), rev, who)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusOK, result)
}

type releaseRequest struct {
	Revision int64  `json:"revision"`
	Reason   string `json:"reason"`
}

func (a *API) ReleaseInvestigation(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var request releaseRequest
	if err = decodeJSON(w, r, &request); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, request.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.ReleaseInvestigation(r.Context(), r.PathValue("investigation_id"), rev, who, request.Reason)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusOK, result)
}

type conclusionRequest struct {
	Revision         int64  `json:"revision"`
	RootCause        string `json:"root_cause"`
	ImpactAssessment string `json:"impact_assessment"`
	RequiredAction   string `json:"required_action"`
}

func (a *API) RecordConclusion(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var input conclusionRequest
	if err = decodeJSON(w, r, &input); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, input.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.RecordConclusion(r.Context(), r.PathValue("investigation_id"), rev, who, application.ConclusionInput{RootCause: input.RootCause, ImpactAssessment: input.ImpactAssessment, RequiredAction: input.RequiredAction})
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Revision)
	success(w, r, http.StatusOK, result)
}
