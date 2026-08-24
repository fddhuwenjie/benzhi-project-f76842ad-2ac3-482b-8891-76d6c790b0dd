package httpapi

import (
	"context"
	"net/http"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

type transferRequest struct {
	Revision            int64            `json:"revision"`
	StationCode         string           `json:"station_code"`
	ReleasedBy          string           `json:"released_by"`
	ReceivedBy          string           `json:"received_by"`
	TransferredAt       domainTime       `json:"transferred_at"`
	ObservedTemperature float64          `json:"observed_temperature"`
	SealState           domain.SealState `json:"seal_state"`
}

type transferItemRequest struct {
	StationCode         string           `json:"station_code"`
	ReleasedBy          string           `json:"released_by"`
	ReceivedBy          string           `json:"received_by"`
	TransferredAt       domainTime       `json:"transferred_at"`
	ObservedTemperature float64          `json:"observed_temperature"`
	SealState           domain.SealState `json:"seal_state"`
}

func (v transferItemRequest) input() domain.TransferInput {
	return domain.TransferInput{StationCode: v.StationCode, ReleasedBy: v.ReleasedBy, ReceivedBy: v.ReceivedBy, TransferredAt: v.TransferredAt.Time, ObservedTemperature: v.ObservedTemperature, SealState: v.SealState}
}

type transferBatchRequest struct {
	Revision  int64                 `json:"revision"`
	Transfers []transferItemRequest `json:"transfers"`
}

func (v transferRequest) input() domain.TransferInput {
	return domain.TransferInput{StationCode: v.StationCode, ReleasedBy: v.ReleasedBy, ReceivedBy: v.ReceivedBy, TransferredAt: v.TransferredAt.Time, ObservedTemperature: v.ObservedTemperature, SealState: v.SealState}
}

func (a *API) RegisterTransfer(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var input transferRequest
	if err = decodeJSON(w, r, &input); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, input.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	result, err := a.service.RegisterTransfer(context.WithoutCancel(r.Context()), r.PathValue("dossier_id"), r.Header.Get("Idempotency-Key"), rev, who, input.input())
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Dossier.Revision)
	status := http.StatusCreated
	if result.IdempotentReplay {
		status = http.StatusOK
	}
	success(w, r, status, result)
}

func (a *API) RegisterTransferBatch(w http.ResponseWriter, r *http.Request) {
	who, err := actor(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	var request transferBatchRequest
	if err = decodeJSON(w, r, &request); err != nil {
		fail(w, r, err)
		return
	}
	rev, err := revision(r, request.Revision)
	if err != nil {
		fail(w, r, err)
		return
	}
	inputs := make([]domain.TransferInput, len(request.Transfers))
	for index, item := range request.Transfers {
		inputs[index] = item.input()
	}
	result, err := a.service.RegisterTransferBatch(context.WithoutCancel(r.Context()), r.PathValue("dossier_id"), r.Header.Get("Idempotency-Key"), rev, who, inputs)
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.Dossier.Revision)
	status := http.StatusCreated
	if result.IdempotentReplay {
		status = http.StatusOK
	}
	success(w, r, status, result)
}

func (a *API) GetTransferProgress(w http.ResponseWriter, r *http.Request) {
	result, err := a.service.GetTransferProgress(r.Context(), r.PathValue("dossier_id"))
	if err != nil {
		fail(w, r, err)
		return
	}
	setRevision(w, result.DossierRevision)
	success(w, r, http.StatusOK, result)
}
