package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

type DossierStatus string

const (
	DossierDraft         DossierStatus = "draft"
	DossierSubmitted     DossierStatus = "submitted"
	DossierInTransit     DossierStatus = "in_transit"
	DossierInvestigation DossierStatus = "anomaly_pending_investigation"
	DossierRemediation   DossierStatus = "remediation_pending"
	DossierReadyToClose  DossierStatus = "ready_to_close"
	DossierClosed        DossierStatus = "closed"
)

type InvestigationStatus string

const (
	InvestigationUnassigned InvestigationStatus = "unassigned"
	InvestigationAssigned   InvestigationStatus = "assigned"
	InvestigationActionSet  InvestigationStatus = "action_required"
	InvestigationReview     InvestigationStatus = "evidence_review"
	InvestigationResolved   InvestigationStatus = "resolved"
)

type ReviewDecision string

const (
	DecisionPending  ReviewDecision = "pending"
	DecisionRejected ReviewDecision = "rejected"
	DecisionAccepted ReviewDecision = "accepted"
)

type SealState string

const (
	SealIntact  SealState = "intact"
	SealBroken  SealState = "broken"
	SealMissing SealState = "missing"
)

const (
	RoleCustodian   = "custodian"
	RoleReceiver    = "receiver"
	RoleReviewer    = "quality_reviewer"
	RoleResponsible = "responsible_person"
)

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func NormalizeText(value string) string       { return strings.TrimSpace(value) }
func NormalizeStation(value string) string    { return strings.ToUpper(strings.TrimSpace(value)) }
func NormalizeTime(value time.Time) time.Time { return value.UTC().Truncate(time.Second) }
func ValidDigest(value string) bool           { return digestPattern.MatchString(strings.ToLower(value)) }
func DigestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func RequireRole(actual string, allowed ...string) error {
	for _, role := range allowed {
		if actual == role {
			return nil
		}
	}
	return NewError(CodeForbidden, "当前角色无权执行该操作")
}
