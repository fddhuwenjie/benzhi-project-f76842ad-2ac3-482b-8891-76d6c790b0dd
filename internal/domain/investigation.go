package domain

import "time"

type AnomalyInvestigation struct {
	ID               string              `json:"id"`
	DossierID        string              `json:"dossier_id"`
	TriggerCodes     []string            `json:"trigger_codes"`
	DetectedAt       time.Time           `json:"detected_at"`
	AssignedReviewer string              `json:"assigned_reviewer,omitempty"`
	RootCause        string              `json:"root_cause,omitempty"`
	ImpactAssessment string              `json:"impact_assessment,omitempty"`
	RequiredAction   string              `json:"required_action,omitempty"`
	Status           InvestigationStatus `json:"status"`
	Revision         int64               `json:"revision"`
}

func NewInvestigation(id, dossierID string, triggers []string, now time.Time) *AnomalyInvestigation {
	return &AnomalyInvestigation{ID: id, DossierID: dossierID, TriggerCodes: append([]string(nil), triggers...),
		DetectedAt: NormalizeTime(now), Status: InvestigationUnassigned, Revision: 1}
}

// MergeTriggers appends any new trigger codes to the investigation, preserving the
// original detection timestamp and investigation identity. This supports registering
// additional custody transfers after an anomaly has already been detected without
// creating a duplicate investigation or losing the original trigger information.
func (i *AnomalyInvestigation) MergeTriggers(triggers []string) {
	for _, code := range triggers {
		if !i.hasTrigger(code) {
			i.TriggerCodes = append(i.TriggerCodes, code)
		}
	}
}

func (i *AnomalyInvestigation) hasTrigger(code string) bool {
	for _, existing := range i.TriggerCodes {
		if existing == code {
			return true
		}
	}
	return false
}

func (i *AnomalyInvestigation) CheckRevision(expected int64) error {
	if expected <= 0 {
		return FieldError("revision", "必须提供正数修订号")
	}
	if i.Revision != expected {
		return NewError(CodeConflict, "调查已被其他请求修改")
	}
	return nil
}

func (i *AnomalyInvestigation) Claim(expected int64, reviewer string) error {
	if err := i.CheckRevision(expected); err != nil {
		return err
	}
	if i.Status != InvestigationUnassigned {
		return NewError(CodeState, "调查已被认领")
	}
	if NormalizeText(reviewer) == "" {
		return FieldError("reviewer", "审核人员不能为空")
	}
	i.AssignedReviewer = NormalizeText(reviewer)
	i.Status = InvestigationAssigned
	i.Revision++
	return nil
}

func (i *AnomalyInvestigation) Release(expected int64, reviewer, reason string) (string, error) {
	if err := i.CheckRevision(expected); err != nil {
		return "", err
	}
	if i.Status != InvestigationAssigned {
		return "", NewError(CodeState, "仅尚未登记结论的已认领调查可以释放")
	}
	if i.AssignedReviewer != NormalizeText(reviewer) {
		return "", NewError(CodeForbidden, "仅当前认领人可以释放调查")
	}
	reason = NormalizeText(reason)
	if reason == "" {
		return "", FieldError("reason", "释放原因不能为空")
	}
	i.AssignedReviewer = ""
	i.Status = InvestigationUnassigned
	i.Revision++
	return reason, nil
}

func (i *AnomalyInvestigation) RecordConclusion(expected int64, reviewer, cause, impact, action string) error {
	if err := i.CheckRevision(expected); err != nil {
		return err
	}
	if i.Status != InvestigationAssigned {
		return NewError(CodeState, "调查必须先认领")
	}
	if i.AssignedReviewer != NormalizeText(reviewer) {
		return NewError(CodeForbidden, "仅认领该调查的审核人员可以登记结论")
	}
	values := []struct{ name, value string }{{"root_cause", cause}, {"impact_assessment", impact}, {"required_action", action}}
	for _, value := range values {
		if NormalizeText(value.value) == "" {
			return FieldError(value.name, "字段不能为空")
		}
	}
	i.RootCause, i.ImpactAssessment, i.RequiredAction = NormalizeText(cause), NormalizeText(impact), NormalizeText(action)
	i.Status = InvestigationActionSet
	i.Revision++
	return nil
}

func (i *AnomalyInvestigation) MarkEvidenceSubmitted() error {
	if i.Status != InvestigationActionSet && i.Status != InvestigationReview {
		return NewError(CodeState, "调查尚未进入整改阶段")
	}
	i.Status = InvestigationReview
	i.Revision++
	return nil
}

func (i *AnomalyInvestigation) Resolve() { i.Status = InvestigationResolved; i.Revision++ }

func (i *AnomalyInvestigation) RequestSupplement() {
	i.Status = InvestigationActionSet
	i.Revision++
}
