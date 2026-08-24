package domain

import (
	"testing"
	"time"
)

func TestDossierSubmissionAndRevision(t *testing.T) {
	now := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	dossier, err := NewDossier("dos_1", DraftInput{SampleCode: " S-1 ", SiteName: "河口", Medium: "水", ContainerType: "玻璃瓶", CollectedAt: now.Add(-time.Hour), RequiredTemperatureMin: 2, RequiredTemperatureMax: 8, MaximumTransitMinutes: 120, ExpectedRoute: []string{"field", "lab"}, ResponsiblePerson: "张三"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err = dossier.Submit(2, now); !IsCode(err, CodeConflict) {
		t.Fatalf("期望修订冲突，实际 %v", err)
	}
	if err = dossier.Submit(1, now); err != nil {
		t.Fatal(err)
	}
	if dossier.Status != DossierSubmitted || dossier.Revision != 2 {
		t.Fatalf("提交状态错误: %+v", dossier)
	}
	if err = dossier.Submit(2, now); !IsCode(err, CodeState) {
		t.Fatalf("重复提交应被状态机拒绝: %v", err)
	}
}

func TestSubmissionRequiresCompleteRoute(t *testing.T) {
	now := time.Now().UTC()
	dossier, err := NewDossier("dos_2", DraftInput{SampleCode: "S-2", SiteName: "湖心", Medium: "水", ContainerType: "瓶", CollectedAt: now, RequiredTemperatureMin: 2, RequiredTemperatureMax: 8, MaximumTransitMinutes: 10, ExpectedRoute: []string{"ONLY"}, ResponsiblePerson: "李四"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err = dossier.Submit(1, now); err == nil {
		t.Fatal("不完整路线不应通过提交")
	}
}
