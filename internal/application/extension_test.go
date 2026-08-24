package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

func extensionService(t *testing.T, now time.Time) *Service {
	t.Helper()
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service := New(store, chaincheck.New())
	service.now = func() time.Time { return now }
	sequence := 0
	service.newID = func(prefix string) string {
		sequence++
		return prefix + "_test_" + time.Unix(int64(sequence), 0).UTC().Format("150405")
	}
	return service
}

func validDraft(code string, now time.Time, route ...string) domain.DraftInput {
	return domain.DraftInput{SampleCode: code, SiteName: "河口站", Medium: "水", ContainerType: "玻璃瓶", CollectedAt: now.Add(-10 * time.Minute),
		RequiredTemperatureMin: 2, RequiredTemperatureMax: 8, MaximumTransitMinutes: 120, ExpectedRoute: route, ResponsiblePerson: "责任人"}
}

func TestDraftRevisionAndSubmissionPreflight(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	service := extensionService(t, now)
	ctx := context.Background()
	actor := Actor{Name: "保管员", Role: domain.RoleCustodian}
	dossier, err := service.CreateDossier(ctx, actor, validDraft("DRAFT-1", now, "field", "lab"))
	if err != nil {
		t.Fatal(err)
	}
	input := dossier.DraftInput()
	input.ContainerType = " 不锈钢罐 "
	input.ExpectedRoute = []string{"field", "archive"}
	dossier, err = service.ReviseDossier(ctx, dossier.ID, 1, actor, input)
	if err != nil {
		t.Fatal(err)
	}
	if dossier.Revision != 2 || dossier.ContainerType != "不锈钢罐" || dossier.ExpectedRoute[1] != "ARCHIVE" {
		t.Fatalf("草稿修订结果错误: %+v", dossier)
	}
	history, err := service.GetAudit(ctx, dossier.ID)
	if err != nil {
		t.Fatal(err)
	}
	var details struct {
		Changes map[string]domain.FieldChange `json:"changes"`
	}
	if err = json.Unmarshal(history[len(history)-1].Details, &details); err != nil {
		t.Fatal(err)
	}
	if len(details.Changes) != 2 || details.Changes["container_type"].After != "不锈钢罐" {
		t.Fatalf("审计差异应只包含实际变化字段: %s", history[len(history)-1].Details)
	}
	stale := dossier.DraftInput()
	stale.ContainerType = "塑料瓶"
	if _, err = service.ReviseDossier(ctx, dossier.ID, 1, actor, stale); !domain.IsCode(err, domain.CodeConflict) {
		t.Fatalf("过期草稿修订应冲突: %v", err)
	}
	view, err := service.GetDossier(ctx, dossier.ID)
	if err != nil || view.Dossier.ContainerType != "不锈钢罐" || view.Dossier.Revision != 2 {
		t.Fatalf("冲突不应覆盖快照: %+v, %v", view, err)
	}

	invalid := validDraft("DRAFT-2", now, "A", "A")
	invalid.ResponsiblePerson = ""
	invalid.RequiredTemperatureMin, invalid.RequiredTemperatureMax = 9, 2
	second, err := service.CreateDossier(ctx, actor, invalid)
	if err != nil {
		t.Fatal(err)
	}
	report, err := service.PreflightSubmission(ctx, second.ID, second.Revision, actor)
	if err != nil {
		t.Fatal(err)
	}
	if report.CanSubmit || len(report.Issues) != 3 || report.Revision != second.Revision {
		t.Fatalf("预检应一次返回三个问题: %+v", report)
	}
	after, _ := service.GetDossier(ctx, second.ID)
	if after.Dossier.Status != domain.DossierDraft || after.Dossier.Revision != 1 || len(after.History) != 1 {
		t.Fatalf("预检不应写入业务数据: %+v", after)
	}
}

func TestTransferBatchIsAtomicIdempotentAndVisibleInProgress(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	service := extensionService(t, now)
	ctx := context.Background()
	custodian := Actor{Name: "保管员", Role: domain.RoleCustodian}
	dossier, err := service.CreateDossier(ctx, custodian, validDraft("BATCH-1", now, "A", "B", "C"))
	if err != nil {
		t.Fatal(err)
	}
	dossier, err = service.SubmitDossier(ctx, dossier.ID, dossier.Revision, custodian)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []domain.TransferInput{
		{StationCode: "a", ReleasedBy: "甲", ReceivedBy: "乙", TransferredAt: now.Add(-8 * time.Minute), ObservedTemperature: 4, SealState: domain.SealIntact},
		{StationCode: "b", ReleasedBy: "乙", ReceivedBy: "丙", TransferredAt: now.Add(-6 * time.Minute), ObservedTemperature: 5, SealState: domain.SealIntact},
		{StationCode: "c", ReleasedBy: "丙", ReceivedBy: "丁", TransferredAt: now.Add(-4 * time.Minute), ObservedTemperature: 6, SealState: domain.SealIntact},
	}
	batch, err := service.RegisterTransferBatch(ctx, dossier.ID, "batch-normal", dossier.Revision, custodian, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Items) != 3 || batch.Dossier.Revision != dossier.Revision+1 {
		t.Fatalf("批量交接结果错误: %+v", batch)
	}
	replay, err := service.RegisterTransferBatch(ctx, dossier.ID, "batch-normal", dossier.Revision, custodian, inputs)
	if err != nil || !replay.IdempotentReplay || replay.Items[0].Transfer.ID != batch.Items[0].Transfer.ID {
		t.Fatalf("批量幂等重放错误: %+v, %v", replay, err)
	}
	progress, err := service.GetTransferProgress(ctx, dossier.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !progress.RouteComplete || progress.CompletedStations != 3 || progress.CurrentCustodian != "丁" || progress.TimeRisk != "normal" {
		t.Fatalf("进度视图错误: %+v", progress)
	}

	blocked, err := service.CreateDossier(ctx, custodian, validDraft("BATCH-2", now, "A", "B"))
	if err != nil {
		t.Fatal(err)
	}
	blocked, err = service.SubmitDossier(ctx, blocked.ID, blocked.Revision, custodian)
	if err != nil {
		t.Fatal(err)
	}
	bad := []domain.TransferInput{
		{StationCode: "A", ReleasedBy: "甲", ReceivedBy: "乙", TransferredAt: now.Add(-8 * time.Minute), ObservedTemperature: 4, SealState: domain.SealIntact},
		{StationCode: "B", ReleasedBy: "其他人", ReceivedBy: "丁", TransferredAt: now.Add(-6 * time.Minute), ObservedTemperature: 4, SealState: domain.SealIntact},
	}
	if _, err = service.RegisterTransferBatch(ctx, blocked.ID, "batch-blocked", blocked.Revision, custodian, bad); !domain.IsCode(err, domain.CodeChain) {
		t.Fatalf("链路断裂应阻断批次: %v", err)
	}
	blockedView, _ := service.GetDossier(ctx, blocked.ID)
	if len(blockedView.Transfers) != 0 || blockedView.Dossier.Revision != blocked.Revision || blockedView.Investigation != nil {
		t.Fatalf("阻断批次不应部分写入: %+v", blockedView)
	}
}

func TestReleaseEvidenceChainAndArchivePreflight(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	service := extensionService(t, now)
	ctx := context.Background()
	custodian := Actor{Name: "保管员", Role: domain.RoleCustodian}
	dossier, err := service.CreateDossier(ctx, custodian, validDraft("CHAIN-1", now, "A", "B"))
	if err != nil {
		t.Fatal(err)
	}
	dossier, err = service.SubmitDossier(ctx, dossier.ID, dossier.Revision, custodian)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := service.RegisterTransferBatch(ctx, dossier.ID, "batch-anomaly", dossier.Revision, custodian, []domain.TransferInput{
		{StationCode: "A", ReleasedBy: "甲", ReceivedBy: "乙", TransferredAt: now.Add(-8 * time.Minute), ObservedTemperature: 4, SealState: domain.SealIntact},
		{StationCode: "B", ReleasedBy: "乙", ReceivedBy: "丙", TransferredAt: now.Add(-6 * time.Minute), ObservedTemperature: 12, SealState: domain.SealBroken},
	})
	if err != nil || batch.Investigation == nil || len(batch.Investigation.TriggerCodes) != 2 {
		t.Fatalf("异常批次应只创建一项汇总调查: %+v, %v", batch, err)
	}
	investigation := batch.Investigation
	reviewerA := Actor{Name: "审核甲", Role: domain.RoleReviewer}
	reviewerB := Actor{Name: "审核乙", Role: domain.RoleReviewer}
	investigation, err = service.ClaimInvestigation(ctx, investigation.ID, investigation.Revision, reviewerA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ReleaseInvestigation(ctx, investigation.ID, investigation.Revision, reviewerB, "无法继续"); !domain.IsCode(err, domain.CodeForbidden) {
		t.Fatalf("非认领人不能释放: %v", err)
	}
	investigation, err = service.ReleaseInvestigation(ctx, investigation.ID, investigation.Revision, reviewerA, "交接班")
	if err != nil || investigation.Status != domain.InvestigationUnassigned {
		t.Fatalf("释放调查失败: %+v, %v", investigation, err)
	}
	page, err := service.ListInvestigationQueue(ctx, reviewerB, InvestigationQueueFilter{Status: "unassigned"})
	if err != nil || len(page.Items) != 1 || page.Items[0].InvestigationID != investigation.ID {
		t.Fatalf("释放后调查应进入未认领队列: %+v, %v", page, err)
	}
	investigation, err = service.ClaimInvestigation(ctx, investigation.ID, investigation.Revision, reviewerB)
	if err != nil {
		t.Fatal(err)
	}
	investigation, err = service.RecordConclusion(ctx, investigation.ID, investigation.Revision, reviewerB,
		ConclusionInput{RootCause: "冷媒失效", ImpactAssessment: "需复测", RequiredAction: "补充记录"})
	if err != nil {
		t.Fatal(err)
	}
	responsible := Actor{Name: "责任人", Role: domain.RoleResponsible}
	first, err := service.SubmitEvidence(ctx, investigation.ID, investigation.Revision, responsible,
		EvidenceInput{Description: "首次材料", MediaType: "application/pdf", ContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err != nil {
		t.Fatal(err)
	}
	investigation, _ = service.store.GetInvestigation(ctx, investigation.ID)
	if _, err = service.ReviewEvidence(ctx, first.ID, investigation.Revision, reviewerB, ReviewInput{Decision: domain.DecisionRejected, Comment: "补充温度曲线"}); err != nil {
		t.Fatal(err)
	}
	investigation, _ = service.store.GetInvestigation(ctx, investigation.ID)
	if _, err = service.SubmitEvidence(ctx, investigation.ID, investigation.Revision, responsible,
		EvidenceInput{Description: "遗漏引用", MediaType: "application/pdf", ContentDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}); err == nil {
		t.Fatal("补充证据必须引用最近被驳回证据")
	}
	second, err := service.SubmitEvidence(ctx, investigation.ID, investigation.Revision, responsible,
		EvidenceInput{PreviousEvidenceID: first.ID, Description: "补充温度曲线", MediaType: "application/pdf", ContentDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil {
		t.Fatal(err)
	}
	view, _ := service.GetDossier(ctx, dossier.ID)
	if len(view.Evidence) != 2 || view.Evidence[1].PreviousEvidenceID != first.ID {
		t.Fatalf("详情应返回不可变证据链: %+v", view.Evidence)
	}
	preflight, err := service.PreflightClose(ctx, dossier.ID, reviewerB)
	if err != nil || preflight.CanClose || len(preflight.Conditions) < 2 {
		t.Fatalf("待复核证据不能关闭且应汇总条件: %+v, %v", preflight, err)
	}
	investigation, _ = service.store.GetInvestigation(ctx, investigation.ID)
	if _, err = service.ReviewEvidence(ctx, second.ID, investigation.Revision, reviewerB, ReviewInput{Decision: domain.DecisionAccepted, Comment: "完整"}); err != nil {
		t.Fatal(err)
	}
	preflight, err = service.PreflightClose(ctx, dossier.ID, reviewerB)
	if err != nil || !preflight.CanClose {
		t.Fatalf("完整链和已接受证据应可关闭: %+v, %v", preflight, err)
	}
	closed, err := service.CloseDossier(ctx, dossier.ID, preflight.Revision, reviewerB)
	if err != nil {
		t.Fatal(err)
	}
	archived, err := service.PreflightClose(ctx, dossier.ID, reviewerB)
	if err != nil || !archived.Archived || archived.ClosedAt == nil || archived.Revision != closed.Revision {
		t.Fatalf("关闭后预检应返回归档信息: %+v, %v", archived, err)
	}
}

func TestInvestigationQueueCursorDoesNotRepeatAfterEarlierInsert(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	service := extensionService(t, now)
	ctx := context.Background()
	custodian := Actor{Name: "保管员", Role: domain.RoleCustodian}
	insert := func(code, investigationID string, detectedAt time.Time) {
		t.Helper()
		dossier, err := service.CreateDossier(ctx, custodian, validDraft(code, now, "A", "B"))
		if err != nil {
			t.Fatal(err)
		}
		err = service.store.WithTx(ctx, func(tx *persistence.Tx) error {
			return tx.InsertInvestigation(ctx, domain.NewInvestigation(investigationID, dossier.ID, []string{chaincheck.CodeSealBroken}, detectedAt))
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("QUEUE-1", "inv_1", now.Add(-time.Hour))
	insert("QUEUE-2", "inv_2", now)
	reviewer := Actor{Name: "审核员", Role: domain.RoleReviewer}
	first, err := service.ListInvestigationQueue(ctx, reviewer, InvestigationQueueFilter{Status: "unassigned", TriggerCode: chaincheck.CodeSealBroken, Limit: 1})
	if err != nil || len(first.Items) != 1 || first.Items[0].InvestigationID != "inv_1" || first.NextCursor == "" {
		t.Fatalf("队列第一页错误: %+v, %v", first, err)
	}
	insert("QUEUE-0", "inv_0", now.Add(-2*time.Hour))
	second, err := service.ListInvestigationQueue(ctx, reviewer, InvestigationQueueFilter{Status: "unassigned", TriggerCode: chaincheck.CodeSealBroken, Limit: 1, Cursor: first.NextCursor})
	if err != nil || len(second.Items) != 1 || second.Items[0].InvestigationID != "inv_2" {
		t.Fatalf("新增更早调查后续页不应重复或回退: %+v, %v", second, err)
	}
	if _, err = service.ListInvestigationQueue(ctx, custodian, InvestigationQueueFilter{}); !domain.IsCode(err, domain.CodeForbidden) {
		t.Fatalf("非审核角色不能查询队列: %v", err)
	}
}
