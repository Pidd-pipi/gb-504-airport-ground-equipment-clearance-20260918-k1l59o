package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"groundclearance/internal/config"
	"groundclearance/internal/handler"
	"groundclearance/internal/model"
	"groundclearance/internal/repository"
	"groundclearance/internal/router"
	"groundclearance/internal/service"
	"groundclearance/internal/util"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type checkData struct {
	ID       uint64   `json:"id"`
	Result   string   `json:"result"`
	Remark   string   `json:"remark"`
	Evidence []string `json:"evidence"`
}

type batchData struct {
	Reviewed int         `json:"reviewed"`
	Items    []checkData `json:"items"`
}

func setupBatchServer(t *testing.T) (*gorm.DB, *httptest.Server, string) {
	t.Helper()
	dsn := os.Getenv("BATCH_TEST_DSN")
	if dsn == "" {
		t.Skip("BATCH_TEST_DSN not set; skipping Postgres integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.GroundUnit{}, &model.Turnaround{},
		&model.SafetyCheck{}, &model.ClearanceDecision{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec(`TRUNCATE audit_logs, safety_checks, turnarounds, ground_units, clearance_decisions, users RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userRepo := repository.NewUserRepository(db)
	unitRepo := repository.NewGroundUnitRepository(db)
	turnaroundRepo := repository.NewTurnaroundRepository(db)
	checkRepo := repository.NewSafetyCheckRepository(db)
	clearanceRepo := repository.NewClearanceDecisionRepository(db)
	turnaroundSvc := service.NewTurnaroundService(db, turnaroundRepo, checkRepo, clearanceRepo, unitRepo, userRepo, logger)
	checkSvc := service.NewSafetyCheckService(db, checkRepo, turnaroundRepo, clearanceRepo, unitRepo, logger)
	clearanceSvc := service.NewClearanceDecisionService(db, clearanceRepo, turnaroundRepo, checkRepo, unitRepo, logger)
	cfg := &config.Config{JWTSecret: "integration-test-secret-at-least-32-characters", RateLimitPerMinute: 99999, CORSOrigins: []string{"http://localhost"}}
	engine := router.New(cfg, db, nil, logger,
		handler.NewUserHandler(service.NewUserService(userRepo, logger), logger),
		handler.NewTurnaroundHandler(turnaroundSvc, logger),
		handler.NewGroundUnitHandler(service.NewGroundUnitService(db, unitRepo, turnaroundRepo, clearanceRepo, logger), logger),
		handler.NewSafetyCheckHandler(checkSvc, logger),
		handler.NewClearanceDecisionHandler(clearanceSvc, logger),
		handler.NewAuditLogHandler(db, logger),
	).Setup()
	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)
	// Inspector user directly in DB; role mirrors router RequireRole.
	user := model.User{Phone: "13900000001", PasswordHash: "x", Name: "集成检查员", Role: "inspector"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := util.GenerateToken(cfg.JWTSecret, time.Hour, user.ID, user.Phone, user.Role)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	now := time.Now()
	ta := model.Turnaround{FlightNo: "IT-BATCH", Stand: "T01", Phase: "servicing", ScheduledAt: now.Add(time.Hour), RiskLevel: "high", Status: "open", CoordinatorID: user.ID}
	if err := db.Create(&ta).Error; err != nil {
		t.Fatalf("create turnaround: %v", err)
	}
	if err := db.Create(&model.ClearanceDecision{TurnaroundID: ta.ID, State: "pending", Reason: "test"}).Error; err != nil {
		t.Fatalf("create decision: %v", err)
	}
	ids := make([]uint64, 0, 4)
	for i, code := range []string{"BC-01", "BC-02", "BC-03", "BC-04"} {
		check := model.SafetyCheck{TurnaroundID: ta.ID, Sequence: i + 1, CheckCode: code, ItemName: "批量检查项 " + code, RiskLevel: "medium", Result: "pending"}
		if err := db.Create(&check).Error; err != nil {
			t.Fatalf("create check: %v", err)
		}
		ids = append(ids, check.ID)
	}
	return db, server, token
}

func doJSON(t *testing.T, method, url, token string, body any) (int, apiEnvelope) {
	t.Helper()
	status, envelope, err := doJSONErr(method, url, token, body)
	if err != nil {
		t.Fatal(err)
	}
	return status, envelope
}

func doJSONErr(method, url, token string, body any) (int, apiEnvelope, error) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(method, url, bytes.NewReader(raw))
	if err != nil {
		return 0, apiEnvelope{}, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, apiEnvelope{}, err
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	var envelope apiEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return resp.StatusCode, envelope, fmt.Errorf("decode response %s: %w", string(payload), err)
	}
	return resp.StatusCode, envelope, nil
}

func TestBatchReviewHTTP(t *testing.T) {
	db, server, token := setupBatchServer(t)
	var checkIDs []uint64
	if err := db.Model(&model.SafetyCheck{}).Order("id asc").Pluck("id", &checkIDs).Error; err != nil {
		t.Fatal(err)
	}

	// 403 without token: permission control unchanged for the new route.
	if status, envelope := doJSON(t, http.MethodPost, server.URL+"/api/v1/checks/batch-review", "", map[string]any{}); status != http.StatusUnauthorized || envelope.Code != 40100 {
		t.Fatalf("unauthenticated batch review: status=%d envelope=%+v", status, envelope)
	}

	// Binding failure: missing per-item result.
	badBody := map[string]any{"items": []map[string]any{{"id": checkIDs[0]}}, "evidence": []string{"e.jpg"}}
	if status, _ := doJSON(t, http.MethodPost, server.URL+"/api/v1/checks/batch-review", token, badBody); status != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, want 400", status)
	}

	// Invalid evidence shared by the batch: no check may be written.
	invalid := map[string]any{
		"items":    []map[string]any{{"id": checkIDs[0], "result": "passed"}, {"id": checkIDs[1], "result": "failed"}},
		"evidence": []string{"   "}, "remark": "",
	}
	if status, envelope := doJSON(t, http.MethodPost, server.URL+"/api/v1/checks/batch-review", token, invalid); status != http.StatusUnprocessableEntity {
		t.Fatalf("invalid evidence status = %d envelope=%+v, want 422", status, envelope)
	}
	assertPendingCount(t, db, len(checkIDs))

	// Duplicate check id inside the same batch.
	dup := map[string]any{
		"items":    []map[string]any{{"id": checkIDs[0], "result": "passed"}, {"id": checkIDs[0], "result": "failed"}},
		"evidence": []string{"shared-1.jpg"},
	}
	if status, _ := doJSON(t, http.MethodPost, server.URL+"/api/v1/checks/batch-review", token, dup); status != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate item status = %d, want 422", status)
	}
	assertPendingCount(t, db, len(checkIDs))

	// Happy path: three items, shared evidence and remark.
	body := map[string]any{
		"items": []map[string]any{
			{"id": checkIDs[0], "result": "passed"},
			{"id": checkIDs[1], "result": "failed"},
			{"id": checkIDs[2], "result": "passed"},
		},
		"evidence": []string{"batch-evidence-1.jpg", " batch-evidence-1.jpg ", "batch-evidence-2.jpg"},
		"remark":   " 批量现场复核 ",
	}
	status, envelope := doJSON(t, http.MethodPost, server.URL+"/api/v1/checks/batch-review", token, body)
	if status != http.StatusOK {
		t.Fatalf("batch review status = %d envelope=%+v", status, envelope)
	}
	var result batchData
	if err := json.Unmarshal(envelope.Data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Reviewed != 3 || len(result.Items) != 3 {
		t.Fatalf("unexpected batch payload: %+v", result)
	}
	// Response preserves request order and carries shared evidence/remark.
	wantResults := []string{"passed", "failed", "passed"}
	for i, row := range result.Items {
		if row.ID != checkIDs[i] || row.Result != wantResults[i] {
			t.Fatalf("item %d mismatch: %+v", i, row)
		}
		if row.Remark != "批量现场复核" || len(row.Evidence) != 2 || row.Evidence[0] != "batch-evidence-1.jpg" {
			t.Fatalf("shared fields not applied to item %d: %+v", i, row)
		}
	}
	// One audit row per item, all sharing the same request id.
	var audits []model.AuditLog
	if err := db.Where("action = ? AND entity_type = ?", "SAFETY_CHECK_REVIEW", "checks").Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 3 {
		t.Fatalf("audit rows = %d, want 3", len(audits))
	}
	requestIDs := map[string]struct{}{}
	for _, audit := range audits {
		var detail map[string]any
		if err := json.Unmarshal([]byte(audit.Detail), &detail); err != nil || detail["batch"] != true {
			t.Fatalf("audit detail missing batch marker: %s", audit.Detail)
		}
		rid, _ := detail["request_id"].(string)
		requestIDs[rid] = struct{}{}
	}
	if len(requestIDs) != 1 {
		t.Fatalf("batch audit request ids = %v, want one shared id", requestIDs)
	}
	// Turnaround promoted open -> checking.
	var ta model.Turnaround
	if err := db.First(&ta, 1).Error; err != nil || ta.Status != "checking" || ta.Version != 2 {
		t.Fatalf("turnaround after batch: %+v err=%v", ta, err)
	}

	// Whole-batch failure when one item was already reviewed: no partial writes.
	conflict := map[string]any{
		"items": []map[string]any{
			{"id": checkIDs[3], "result": "passed"},
			{"id": checkIDs[0], "result": "failed"},
		},
		"evidence": []string{"should-not-persist.jpg"},
	}
	if status, envelope := doJSON(t, http.MethodPost, server.URL+"/api/v1/checks/batch-review", token, conflict); status != http.StatusConflict {
		t.Fatalf("conflict status = %d envelope=%+v, want 409", status, envelope)
	} else if envelope.Code != 40901 {
		t.Fatalf("conflict code = %d, want 40901", envelope.Code)
	}
	var stillPending int64
	if err := db.Model(&model.SafetyCheck{}).Where("id = ? AND result = ?", checkIDs[3], "pending").Count(&stillPending).Error; err != nil {
		t.Fatal(err)
	}
	if stillPending != 1 {
		t.Fatalf("check %d was partially written despite failed batch", checkIDs[3])
	}
	var leakedEvidence int64
	if err := db.Model(&model.SafetyCheck{}).Where("evidence::text LIKE ?", "%should-not-persist%").Count(&leakedEvidence).Error; err != nil {
		t.Fatal(err)
	}
	if leakedEvidence != 0 {
		t.Fatalf("failed batch left evidence behind: %d rows", leakedEvidence)
	}
	var auditCount int64
	db.Model(&model.AuditLog{}).Where("action = ?", "SAFETY_CHECK_REVIEW").Count(&auditCount)
	if auditCount != 3 {
		t.Fatalf("audit rows after failed batch = %d, want still 3", auditCount)
	}

	// Missing check id: 404 inside the transaction, nothing written.
	missing := map[string]any{
		"items":    []map[string]any{{"id": checkIDs[3], "result": "passed"}, {"id": 999999, "result": "passed"}},
		"evidence": []string{"missing.jpg"},
	}
	if status, _ := doJSON(t, http.MethodPost, server.URL+"/api/v1/checks/batch-review", token, missing); status != http.StatusNotFound {
		t.Fatalf("missing item status = %d, want 404", status)
	}
	assertPendingCount(t, db, 1)

	// Single-item review still works and flips the last pending check.
	single := map[string]any{"result": "passed", "evidence": []string{"single.jpg"}, "remark": "单项"}
	if status, envelope := doJSON(t, http.MethodPatch, fmt.Sprintf("%s/api/v1/checks/%d/review", server.URL, checkIDs[3]), token, single); status != http.StatusOK {
		t.Fatalf("single review status = %d envelope=%+v", status, envelope)
	}
	assertPendingCount(t, db, 0)
}

func TestBatchReviewConcurrentOverlap(t *testing.T) {
	db, server, token := setupBatchServer(t)
	var checkIDs []uint64
	if err := db.Model(&model.SafetyCheck{}).Order("id asc").Pluck("id", &checkIDs).Error; err != nil {
		t.Fatal(err)
	}

	// Two batches overlap on the middle items. Row locks must serialize them:
	// one wins, the other is rejected wholesale (409) with no partial writes.
	batchA := map[string]any{
		"items":    []map[string]any{{"id": checkIDs[0], "result": "passed"}, {"id": checkIDs[1], "result": "passed"}},
		"evidence": []string{"concurrent-a.jpg"},
	}
	batchB := map[string]any{
		"items":    []map[string]any{{"id": checkIDs[1], "result": "failed"}, {"id": checkIDs[2], "result": "failed"}, {"id": checkIDs[3], "result": "failed"}},
		"evidence": []string{"concurrent-b.jpg"},
	}
	type callResult struct {
		status int
		err    error
	}
	start := make(chan struct{})
	results := make(chan callResult, 2)
	fire := func(body map[string]any) {
		go func() {
			<-start
			status, _, err := doJSONErr(http.MethodPost, server.URL+"/api/v1/checks/batch-review", token, body)
			results <- callResult{status, err}
		}()
	}
	fire(batchA)
	fire(batchB)
	close(start)
	first, second := <-results, <-results
	if first.err != nil {
		t.Fatal(first.err)
	}
	if second.err != nil {
		t.Fatal(second.err)
	}
	successes, conflicts := 0, 0
	for _, result := range []callResult{first, second} {
		switch result.status {
		case http.StatusOK:
			successes++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected concurrent status %d", result.status)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent batches: successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}
	// The loser must not have written any of its conclusions.
	var failedFromB int64
	if err := db.Model(&model.SafetyCheck{}).Where("result = ? AND evidence::text LIKE ?", "failed", "%concurrent-b.jpg%").Count(&failedFromB).Error; err != nil {
		t.Fatal(err)
	}
	if failedFromB != 0 {
		t.Fatalf("losing batch left %d failed conclusions", failedFromB)
	}
	var reviewedCount int64
	db.Model(&model.SafetyCheck{}).Where("result <> ?", "pending").Count(&reviewedCount)
	if reviewedCount != 2 {
		t.Fatalf("reviewed checks = %d, want exactly the 2 items of the winning batch", reviewedCount)
	}
}

func assertPendingCount(t *testing.T, db *gorm.DB, want int) {
	t.Helper()
	var pending int64
	if err := db.Model(&model.SafetyCheck{}).Where("result = ?", "pending").Count(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if pending != int64(want) {
		t.Fatalf("pending checks = %d, want %d", pending, want)
	}
}
