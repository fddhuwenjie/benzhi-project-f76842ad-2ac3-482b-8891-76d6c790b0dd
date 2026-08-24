package chaincheck

import (
	"fmt"
	"math"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

type Evaluator struct{}

func New() *Evaluator { return &Evaluator{} }

func (e *Evaluator) EvaluateAddition(d *domain.SampleDossier, existing []domain.CustodyTransfer, next domain.CustodyTransfer) Report {
	report := Report{Findings: []Finding{}}
	expectedSequence := len(existing) + 1
	if next.Sequence != expectedSequence {
		report.Findings = append(report.Findings, Finding{Code: CodeSequenceGap, Message: "交接序号不连续", Sequence: next.Sequence, Blocking: true})
	}
	if next.TransferredAt.Before(d.CollectedAt) {
		report.Findings = append(report.Findings, Finding{Code: CodeTimeRegression, Message: "交接时间早于采样时间", Sequence: next.Sequence, Blocking: true})
	}
	if len(existing) > 0 {
		previous := existing[len(existing)-1]
		if next.TransferredAt.Before(previous.TransferredAt) {
			report.Findings = append(report.Findings, Finding{Code: CodeTimeRegression, Message: "交接时间早于上一站", Sequence: next.Sequence, Blocking: true})
		}
		if previous.ReceivedBy != next.ReleasedBy {
			report.Findings = append(report.Findings, Finding{Code: CodePartyDiscontinuity, Message: "上一站接收人与本站交出人不一致", Sequence: next.Sequence, Blocking: true})
		}
	}
	if next.Sequence <= len(d.ExpectedRoute) && next.StationCode != d.ExpectedRoute[next.Sequence-1] {
		report.Findings = append(report.Findings, Finding{Code: CodeRouteDeviation, Message: fmt.Sprintf("站点 %s 偏离预期路线", next.StationCode), Sequence: next.Sequence})
	}
	if next.Sequence > len(d.ExpectedRoute) {
		report.Findings = append(report.Findings, Finding{Code: CodeRouteDeviation, Message: "交接站点超出预期路线", Sequence: next.Sequence})
	}
	if next.ObservedTemperature < d.RequiredTemperatureMin {
		report.Findings = append(report.Findings, Finding{Code: CodeTemperatureLow, Message: "观测温度低于保存下限", Sequence: next.Sequence})
	}
	if next.ObservedTemperature > d.RequiredTemperatureMax {
		report.Findings = append(report.Findings, Finding{Code: CodeTemperatureHigh, Message: "观测温度高于保存上限", Sequence: next.Sequence})
	}
	if math.IsNaN(next.ObservedTemperature) || math.IsInf(next.ObservedTemperature, 0) {
		report.Findings = append(report.Findings, Finding{Code: CodeTemperatureHigh, Message: "观测温度不是有效数值", Sequence: next.Sequence, Blocking: true})
	}
	if next.SealState == domain.SealBroken {
		report.Findings = append(report.Findings, Finding{Code: CodeSealBroken, Message: "封签破损", Sequence: next.Sequence})
	}
	if next.SealState == domain.SealMissing {
		report.Findings = append(report.Findings, Finding{Code: CodeSealMissing, Message: "封签缺失", Sequence: next.Sequence})
	}
	deadline := d.CollectedAt.Add(time.Duration(d.MaximumTransitMinutes) * time.Minute)
	if next.TransferredAt.After(deadline) {
		report.Findings = append(report.Findings, Finding{Code: CodeTransitTimeout, Message: "交接已超过规定流转时限", Sequence: next.Sequence})
	}
	return report
}

func (e *Evaluator) EvaluateExisting(d *domain.SampleDossier, transfers []domain.CustodyTransfer) Report {
	report := Report{Findings: []Finding{}}
	for index := range transfers {
		addition := e.EvaluateAddition(d, transfers[:index], transfers[index])
		report.Findings = append(report.Findings, addition.Findings...)
	}
	return report
}
