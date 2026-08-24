package httpapi

import (
	"net/http"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
)

const maxBodyBytes int64 = 1 << 20

type API struct {
	service *application.Service
	mux     *http.ServeMux
}

func New(service *application.Service) *API {
	a := &API{service: service, mux: http.NewServeMux()}
	a.routes()
	return a
}

func (a *API) Handler() http.Handler { return requestContext(a.mux) }

func (a *API) routes() {
	a.mux.HandleFunc("GET /healthz", a.Health)
	a.mux.HandleFunc("POST /api/v1/dossiers", a.CreateDossier)
	a.mux.HandleFunc("GET /api/v1/dossiers/{dossier_id}", a.GetDossier)
	a.mux.HandleFunc("PATCH /api/v1/dossiers/{dossier_id}", a.ReviseDossier)
	a.mux.HandleFunc("POST /api/v1/dossiers/{dossier_id}/submit/preflight", a.PreflightSubmission)
	a.mux.HandleFunc("POST /api/v1/dossiers/{dossier_id}/submit", a.SubmitDossier)
	a.mux.HandleFunc("GET /api/v1/dossiers/{dossier_id}/transfers/progress", a.GetTransferProgress)
	a.mux.HandleFunc("POST /api/v1/dossiers/{dossier_id}/transfers/batch", a.RegisterTransferBatch)
	a.mux.HandleFunc("POST /api/v1/dossiers/{dossier_id}/transfers", a.RegisterTransfer)
	a.mux.HandleFunc("GET /api/v1/dossiers/{dossier_id}/close/preflight", a.PreflightClose)
	a.mux.HandleFunc("POST /api/v1/dossiers/{dossier_id}/close", a.CloseDossier)
	a.mux.HandleFunc("GET /api/v1/dossiers/{dossier_id}/audit", a.GetAudit)
	a.mux.HandleFunc("GET /api/v1/investigations/queue", a.ListInvestigationQueue)
	a.mux.HandleFunc("POST /api/v1/investigations/{investigation_id}/claim", a.ClaimInvestigation)
	a.mux.HandleFunc("POST /api/v1/investigations/{investigation_id}/release", a.ReleaseInvestigation)
	a.mux.HandleFunc("POST /api/v1/investigations/{investigation_id}/conclusion", a.RecordConclusion)
	a.mux.HandleFunc("POST /api/v1/investigations/{investigation_id}/evidence", a.SubmitEvidence)
	a.mux.HandleFunc("POST /api/v1/evidence/{evidence_id}/review", a.ReviewEvidence)
}
