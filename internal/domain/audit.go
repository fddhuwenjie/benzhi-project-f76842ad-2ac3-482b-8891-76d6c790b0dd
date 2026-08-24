package domain

import (
	"encoding/json"
	"time"
)

type AuditEvent struct {
	ID         int64           `json:"id"`
	DossierID  string          `json:"dossier_id"`
	EventType  string          `json:"event_type"`
	Actor      string          `json:"actor"`
	Role       string          `json:"role"`
	Revision   int64           `json:"revision"`
	OccurredAt time.Time       `json:"occurred_at"`
	Details    json.RawMessage `json:"details"`
}

func NewAudit(dossierID, eventType, actor, role string, revision int64, details any, now time.Time) AuditEvent {
	payload, _ := json.Marshal(details)
	return AuditEvent{DossierID: dossierID, EventType: eventType, Actor: NormalizeText(actor), Role: role, Revision: revision, OccurredAt: NormalizeTime(now), Details: payload}
}
