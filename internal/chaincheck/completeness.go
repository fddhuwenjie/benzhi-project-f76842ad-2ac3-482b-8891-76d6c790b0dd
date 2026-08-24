package chaincheck

import "benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"

func (e *Evaluator) EvaluateComplete(d *domain.SampleDossier, transfers []domain.CustodyTransfer) Report {
	report := Report{}
	if len(transfers) != len(d.ExpectedRoute) {
		report.Findings = append(report.Findings, Finding{Code: CodeRouteIncomplete, Message: "交接数量与预期路线不一致", Blocking: true})
	}
	for index, transfer := range transfers {
		if transfer.Sequence != index+1 {
			report.Findings = append(report.Findings, Finding{Code: CodeSequenceGap, Message: "交接序号存在缺口", Sequence: transfer.Sequence, Blocking: true})
		}
		if index < len(d.ExpectedRoute) && transfer.StationCode != d.ExpectedRoute[index] {
			report.Findings = append(report.Findings, Finding{Code: CodeRouteDeviation, Message: "交接站点与预期路线不一致", Sequence: transfer.Sequence, Blocking: true})
		}
		if index > 0 && transfers[index-1].ReceivedBy != transfer.ReleasedBy {
			report.Findings = append(report.Findings, Finding{Code: CodePartyDiscontinuity, Message: "交接责任链不连续", Sequence: transfer.Sequence, Blocking: true})
		}
		if index > 0 && transfer.TransferredAt.Before(transfers[index-1].TransferredAt) {
			report.Findings = append(report.Findings, Finding{Code: CodeTimeRegression, Message: "交接时间顺序错误", Sequence: transfer.Sequence, Blocking: true})
		}
		if index == 0 && transfer.TransferredAt.Before(d.CollectedAt) {
			report.Findings = append(report.Findings, Finding{Code: CodeTimeRegression, Message: "首次交接时间早于采样时间", Sequence: transfer.Sequence, Blocking: true})
		}
	}
	return report
}
