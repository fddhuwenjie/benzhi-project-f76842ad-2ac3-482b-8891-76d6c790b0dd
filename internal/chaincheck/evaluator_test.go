package chaincheck

import (
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

func TestEvaluateAdditionSeparatesAnomalyAndBlocking(t *testing.T) {
	now := time.Now().UTC()
	d := &domain.SampleDossier{CollectedAt: now.Add(-time.Hour), RequiredTemperatureMin: 2, RequiredTemperatureMax: 8, MaximumTransitMinutes: 120, ExpectedRoute: []string{"FIELD", "LAB"}}
	first := domain.CustodyTransfer{Sequence: 1, StationCode: "FIELD", ReleasedBy: "A", ReceivedBy: "B", TransferredAt: now.Add(-30 * time.Minute), ObservedTemperature: 4, SealState: domain.SealIntact}
	next := domain.CustodyTransfer{Sequence: 2, StationCode: "LAB", ReleasedBy: "B", ReceivedBy: "C", TransferredAt: now, ObservedTemperature: 12, SealState: domain.SealBroken}
	report := New().EvaluateAddition(d, []domain.CustodyTransfer{first}, next)
	if report.HasBlocking() {
		t.Fatalf("温控和封签异常不应阻断事实登记: %+v", report)
	}
	if !report.HasAnomaly() || len(report.TriggerCodes()) != 2 {
		t.Fatalf("应识别两类异常: %+v", report)
	}
	next.ReleasedBy = "其他人"
	if report = New().EvaluateAddition(d, []domain.CustodyTransfer{first}, next); !report.HasBlocking() {
		t.Fatal("责任链断裂必须阻断")
	}
}

func TestEvaluateCompleteRequiresEveryRouteStation(t *testing.T) {
	d := &domain.SampleDossier{ExpectedRoute: []string{"A", "B"}}
	report := New().EvaluateComplete(d, []domain.CustodyTransfer{{Sequence: 1, StationCode: "A"}})
	if !report.HasBlocking() {
		t.Fatal("缺站的交接链不能关闭")
	}
}
