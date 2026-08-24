package httpapi

import (
	"context"
	"net/http"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

type evidenceRequest struct {
	Revision           int64  `json:"revision"`
	PreviousEvidenceID string `json:"previous_evidence_id,omitempty"`
	Description        string `json:"description"`
	MediaType          string `json:"media_type"`
	ContentDigest      string `json:"content_digest"`
}

func (a *API) SubmitEvidence(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var input evidenceRequest
	if err = decodeJSON(w, r, &input); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, input.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.SubmitEvidence(context.WithoutCancel(r.Context()), r.PathValue("investigation_id"), rev, who,
		application.EvidenceInput{PreviousEvidenceID: input.PreviousEvidenceID, Description: input.Description, MediaType: input.MediaType, ContentDigest: input.ContentDigest})
	if err != nil {
		fail(w, r, err)
		return
	}
	success(w, r, http.StatusCreated, result)
}

type reviewRequest struct {
	Revision int64                 `json:"revision"`
	Decision domain.ReviewDecision `json:"decision"`
	Comment  string                `json:"comment"`
}

func (a *API) ReviewEvidence(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var input reviewRequest
	if err = decodeJSON(w, r, &input); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, input.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.ReviewEvidence(context.WithoutCancel(r.Context()), r.PathValue("evidence_id"), rev, who, application.ReviewInput{Decision: input.Decision, Comment: input.Comment})
	if err != nil {
		fail(w, r, err)
		return
	}
	success(w, r, http.StatusOK, result)
}
