package domain

import "time"

type CorrectiveEvidence struct {
	ID                 string         `json:"id"`
	InvestigationID    string         `json:"investigation_id"`
	PreviousEvidenceID string         `json:"previous_evidence_id,omitempty"`
	SubmittedBy        string         `json:"submitted_by"`
	Description        string         `json:"description"`
	MediaType          string         `json:"media_type"`
	ContentDigest      string         `json:"content_digest"`
	SubmittedAt        time.Time      `json:"submitted_at"`
	ReviewedBy         string         `json:"reviewed_by,omitempty"`
	ReviewDecision     ReviewDecision `json:"review_decision"`
	ReviewComment      string         `json:"review_comment,omitempty"`
	ReviewedAt         *time.Time     `json:"reviewed_at,omitempty"`
}

func NewEvidence(id, investigationID, submitter, description, mediaType, digest string, now time.Time, previousEvidenceID ...string) (*CorrectiveEvidence, error) {
	if NormalizeText(submitter) == "" {
		return nil, FieldError("submitted_by", "提交人不能为空")
	}
	if NormalizeText(description) == "" {
		return nil, FieldError("description", "整改说明不能为空")
	}
	if NormalizeText(mediaType) == "" {
		return nil, FieldError("media_type", "媒体类型不能为空")
	}
	digest = NormalizeText(digest)
	if !ValidDigest(digest) {
		return nil, FieldError("content_digest", "内容摘要必须是 64 位 SHA-256 十六进制字符串")
	}
	previous := ""
	if len(previousEvidenceID) > 0 {
		previous = NormalizeText(previousEvidenceID[0])
	}
	return &CorrectiveEvidence{ID: id, InvestigationID: investigationID, PreviousEvidenceID: previous, SubmittedBy: NormalizeText(submitter), Description: NormalizeText(description),
		MediaType: NormalizeText(mediaType), ContentDigest: digest, SubmittedAt: NormalizeTime(now), ReviewDecision: DecisionPending}, nil
}

func (e *CorrectiveEvidence) Review(reviewer string, decision ReviewDecision, comment string, now time.Time) error {
	if e.ReviewDecision != DecisionPending {
		return NewError(CodeState, "证据已经完成复核")
	}
	if decision != DecisionAccepted && decision != DecisionRejected {
		return FieldError("decision", "复核决定必须是 accepted 或 rejected")
	}
	if NormalizeText(reviewer) == "" {
		return FieldError("reviewed_by", "复核人不能为空")
	}
	if decision == DecisionRejected && NormalizeText(comment) == "" {
		return FieldError("comment", "驳回时必须填写补充要求")
	}
	at := NormalizeTime(now)
	e.ReviewedBy, e.ReviewDecision, e.ReviewComment, e.ReviewedAt = NormalizeText(reviewer), decision, NormalizeText(comment), &at
	return nil
}
