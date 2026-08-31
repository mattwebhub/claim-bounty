//go:build integration

package postgres_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/postgres"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/system"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
)

func TestClaimBountyPersistenceSessionOwnershipOutboxAndInspection(t *testing.T) {
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseURL, cleanup := isolatedSchema(t, ctx, baseURL)
	defer cleanup()
	if err := postgres.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, databaseURL, 4, 5*time.Second, claimTestEmailProtector(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	subject := claimID(t, "11111111-1111-4111-8111-111111111111")
	challengeID := claimID(t, "22222222-2222-4222-8222-222222222222")
	challengeHash := sha256.Sum256([]byte("challenge"))
	if err := store.CreateChallenge(ctx, ports.Challenge{ID: challengeID, SubjectID: subject, Email: "owner@example.test", Audience: domain.SubmitterAudience, TokenHash: challengeHash, ExpiresAt: now.Add(10 * time.Minute), AttemptsRemaining: 5}); err != nil {
		t.Fatal(err)
	}
	sessionHash := sha256.Sum256([]byte("session"))
	csrfHash := sha256.Sum256([]byte("csrf"))
	credential, err := store.ExchangeChallenge(ctx, "owner@example.test", domain.SubmitterAudience, challengeHash, claimID(t, "33333333-3333-4333-8333-333333333333"), sessionHash, csrfHash, "authorization-v1", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if credential.Session.SubjectID != subject {
		t.Fatalf("session subject = %s", credential.Session.SubjectID)
	}
	storedCredential, err := store.GetSession(ctx, sessionHash, now)
	if err != nil {
		t.Fatal(err)
	}
	csrfCandidates := [][32]byte{sha256.Sum256([]byte("csrf-refresh-a")), sha256.Sum256([]byte("csrf-refresh-b"))}
	rotateResults := make(chan error, len(csrfCandidates))
	rotateStart := make(chan struct{})
	for _, candidate := range csrfCandidates {
		go func(candidate [32]byte) {
			<-rotateStart
			rotateResults <- store.RotateCSRF(ctx, credential.Session.ID, storedCredential.CSRFHash, candidate, now)
		}(candidate)
	}
	close(rotateStart)
	succeeded, conflicted := 0, 0
	for range csrfCandidates {
		switch err := <-rotateResults; {
		case err == nil:
			succeeded++
		case errors.Is(err, domain.ErrStateConflict):
			conflicted++
		default:
			t.Fatalf("concurrent RotateCSRF() error = %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("concurrent RotateCSRF() results = %d success/%d conflict, want 1/1", succeeded, conflicted)
	}
	storedCredential, err = store.GetSession(ctx, sessionHash, now)
	if err != nil {
		t.Fatal(err)
	}
	if storedCredential.CSRFHash != csrfCandidates[0] && storedCredential.CSRFHash != csrfCandidates[1] {
		t.Fatalf("stored CSRF hash = %x, want one successful candidate", storedCredential.CSRFHash)
	}
	order, err := domain.NewOrder(claimID(t, "44444444-4444-4444-8444-444444444444"), subject, "owner@example.test", "CB-01J7Y8K2Q9ZX", "Study", "Audit", domain.TargetClaim{Text: "Claim"}, domain.Permissions{}, domain.Privacy{}, now)
	if err != nil {
		t.Fatal(err)
	}
	write := ports.IdempotentOrderWrite{Order: order, ActorID: subject, Operation: "create_order", IdempotencyKey: "create-order-0001", RequestHash: sha256.Sum256([]byte("create"))}
	order, err = store.CreateOrder(ctx, write)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeSession(ctx, sessionHash, now); err != nil {
		t.Fatal(err)
	}
	secondChallengeHash := sha256.Sum256([]byte("second-challenge"))
	if err := store.CreateChallenge(ctx, ports.Challenge{
		ID: claimID(t, "22222222-2222-4222-8222-222222222223"), SubjectID: claimID(t, "22222222-2222-4222-8222-222222222224"),
		Email: "owner@example.test", Audience: domain.SubmitterAudience, TokenHash: secondChallengeHash,
		ExpiresAt: now.Add(10 * time.Minute), AttemptsRemaining: 5,
	}); err != nil {
		t.Fatal(err)
	}
	secondSessionHash := sha256.Sum256([]byte("second-session"))
	secondCredential, err := store.ExchangeChallenge(ctx, "owner@example.test", domain.SubmitterAudience, secondChallengeHash,
		claimID(t, "33333333-3333-4333-8333-333333333334"), secondSessionHash, sha256.Sum256([]byte("second-csrf")),
		"authorization-v1", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if secondCredential.Session.SubjectID != subject {
		t.Fatalf("reverified subject = %s, want retained order owner %s", secondCredential.Session.SubjectID, subject)
	}
	if _, err := store.GetOwnedOrder(ctx, secondCredential.Session.SubjectID, order.ID); err != nil {
		t.Fatalf("reverified owner cannot access existing order: %v", err)
	}
	other := claimID(t, "55555555-5555-4555-8555-555555555555")
	if _, err := store.GetOwnedOrder(ctx, other, order.ID); err != domain.ErrOrderNotFound {
		t.Fatalf("cross-subject read error = %v", err)
	}
	file, err := domain.NewOrderFile(claimID(t, "66666666-6666-4666-8666-666666666666"), "primary_paper", "study.pdf", 4, strings.Repeat("a", 64), "application/pdf", "quarantine/object", now)
	if err != nil {
		t.Fatal(err)
	}
	file, _ = file.ConfirmUpload("etag", "version-1", now)
	reserved, err := order.ReserveFile(file, order.Version, now)
	if err != nil {
		t.Fatal(err)
	}
	reserved, savedFile, err := store.SaveUploadedFile(ctx, ports.UploadedFileWrite{Write: ports.IdempotentOrderWrite{Order: reserved, ExpectedVersion: order.Version, ActorID: subject, Operation: "reserve_upload", IdempotencyKey: "upload-file-00001", RequestHash: sha256.Sum256([]byte("upload"))}, File: file})
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := reserved.Submit(reserved.Version, "terms-v1", true, now)
	if err != nil {
		t.Fatal(err)
	}
	submitted, err = store.SaveOrderIdempotent(ctx, ports.IdempotentOrderWrite{Order: submitted, ExpectedVersion: reserved.Version, ActorID: subject, Operation: "submit_order", IdempotencyKey: "submit-order-001", RequestHash: sha256.Sum256([]byte("submit"))})
	if err != nil {
		t.Fatal(err)
	}
	claimedOrder, claimedFile, ok, err := store.ClaimInspection(ctx, now)
	if err != nil || !ok {
		t.Fatalf("ClaimInspection = %v, %v (cause: %v)", ok, err, errors.Unwrap(err))
	}
	if claimedFile.ID != savedFile.ID || claimedFile.Status != "scanning" {
		t.Fatalf("claimed file = %#v", claimedFile)
	}
	clean, err := claimedFile.InspectionResult(true, "application/pdf", "", now)
	if err != nil {
		t.Fatal(err)
	}
	finished, err := claimedOrder.ReplaceFile(clean, claimedOrder.Version, now)
	if err != nil {
		t.Fatal(err)
	}
	finished, err = finished.ReconcileInspection(now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.FinishInspection(ctx, finished, clean, claimedOrder.Version, ports.ExpiredObject{Key: claimedFile.StorageKey, Generation: claimedFile.ObjectGeneration}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetOwnedOrder(ctx, subject, submitted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Files[0].Status != "clean" || loaded.Status != "needs_information" {
		t.Fatalf("finished order = %#v", loaded)
	}
}

func TestClaimBountyConcurrentFirstWriteIdempotencyReturnsCommittedOrder(t *testing.T) {
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseURL, cleanup := isolatedSchema(t, ctx, baseURL)
	defer cleanup()
	if err := postgres.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, databaseURL, 4, 5*time.Second, claimTestEmailProtector(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	actor := claimID(t, "11111111-1111-4111-8111-111111111111")
	orders := make([]domain.Order, 2)
	orders[0], err = domain.NewOrder(claimID(t, "22222222-2222-4222-8222-222222222221"), actor, "owner@example.test", "CB-ABC123DEF456", "Study", "Audit", domain.TargetClaim{Text: "Claim"}, domain.Permissions{}, domain.Privacy{}, now)
	if err != nil {
		t.Fatal(err)
	}
	orders[1], err = domain.NewOrder(claimID(t, "22222222-2222-4222-8222-222222222222"), actor, "owner@example.test", "CB-XYZ123ABC789", "Study", "Audit", domain.TargetClaim{Text: "Claim"}, domain.Permissions{}, domain.Privacy{}, now)
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		order domain.Order
		err   error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, order := range orders {
		go func(order domain.Order) {
			ready.Done()
			<-start
			stored, err := store.CreateOrder(ctx, ports.IdempotentOrderWrite{Order: order, ActorID: actor, Operation: "create_order", IdempotencyKey: "concurrent-create-1", RequestHash: sha256.Sum256([]byte("same-request"))})
			results <- result{order: stored, err: err}
		}(order)
	}
	ready.Wait()
	close(start)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("concurrent CreateOrder() errors = %v, %v", first.err, second.err)
	}
	if first.order.ID != second.order.ID {
		t.Fatalf("concurrent replay returned %s and %s", first.order.ID, second.order.ID)
	}
}

func TestClaimBountySubmitAndExportIdempotencySurvivesConcurrencyAndResponseLoss(t *testing.T) {
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseURL, cleanup := isolatedSchema(t, ctx, baseURL)
	defer cleanup()
	if err := postgres.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, databaseURL, 6, 5*time.Second, claimTestEmailProtector(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	submitter := claimID(t, "11111111-1111-4111-8111-111111111111")
	order, err := domain.NewOrder(
		claimID(t, "22222222-2222-4222-8222-222222222222"), submitter,
		"owner@example.test", "CB-CONCUR123456", "Study", "Audit",
		domain.TargetClaim{Text: "Claim", SourceLocation: "Table 2"},
		domain.Permissions{}, domain.Privacy{}, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	order, err = store.CreateOrder(ctx, ports.IdempotentOrderWrite{
		Order: order, ActorID: submitter, Operation: "create_order",
		IdempotencyKey: "create-concurrent-order", RequestHash: sha256.Sum256([]byte("create")),
	})
	if err != nil {
		t.Fatal(err)
	}
	file, err := domain.NewOrderFile(
		claimID(t, "33333333-3333-4333-8333-333333333333"), "primary_paper",
		"paper.pdf", 4, strings.Repeat("a", 64), "application/pdf", "quarantine/concurrent", now,
	)
	if err != nil {
		t.Fatal(err)
	}
	file, err = file.ConfirmUpload("etag", "version-source", now)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := order.ReserveFile(file, order.Version, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	reserved, _, err = store.SaveUploadedFile(ctx, ports.UploadedFileWrite{
		Write: ports.IdempotentOrderWrite{
			Order: reserved, ExpectedVersion: order.Version, ActorID: submitter,
			Operation: "reserve_upload", IdempotencyKey: "upload-concurrent-paper",
			RequestHash: sha256.Sum256([]byte("upload")),
		},
		File: file,
	})
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := reserved.SubmitWithAuthorizations(
		reserved.Version, "terms-v1", true, true, true, false, now.Add(2*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	submitHash := sha256.Sum256([]byte("submit-authorizations"))
	submitWrite := ports.IdempotentOrderWrite{
		Order: submitted, ExpectedVersion: reserved.Version, ActorID: submitter,
		Operation: "submit_order", IdempotencyKey: "submit-concurrent-order", RequestHash: submitHash,
	}

	type submitResult struct {
		order domain.Order
		err   error
	}
	startSubmit := make(chan struct{})
	submitResults := make(chan submitResult, 2)
	var submitReady sync.WaitGroup
	submitReady.Add(2)
	for range 2 {
		go func() {
			submitReady.Done()
			<-startSubmit
			result, resultErr := store.SaveOrderIdempotent(ctx, submitWrite)
			submitResults <- submitResult{order: result, err: resultErr}
		}()
	}
	submitReady.Wait()
	close(startSubmit)
	firstSubmit, secondSubmit := <-submitResults, <-submitResults
	if firstSubmit.err != nil || secondSubmit.err != nil {
		t.Fatalf("concurrent submit errors = %v, %v", firstSubmit.err, secondSubmit.err)
	}
	if firstSubmit.order.ID != secondSubmit.order.ID || firstSubmit.order.Version != secondSubmit.order.Version {
		t.Fatalf("concurrent submit replay returned different orders")
	}
	replayedSubmit, ok, err := store.GetIdempotentOrder(ctx, submitter, "submit_order", "submit-concurrent-order", submitHash)
	if err != nil || !ok || replayedSubmit.ID != submitted.ID {
		t.Fatalf("response-loss submit replay = %t/%s (error %v)", ok, replayedSubmit.ID, err)
	}
	persisted, err := store.GetOwnedOrder(ctx, submitter, submitted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !persisted.Authorizations.UploadsAuthorized || !persisted.Authorizations.AnalysisUseAuthorized || persisted.Authorizations.ExternalRedistributionAuthorized || persisted.Authorizations.ConfirmedAt == nil {
		t.Fatalf("persisted customer authorizations = %#v", persisted.Authorizations)
	}

	readyAt := now.Add(3 * time.Second)
	if _, err := database.ExecContext(ctx, `UPDATE claimbounty_files SET status='clean',detected_media_type='application/pdf',scanned_at=$2,updated_at=$2 WHERE order_id=$1`, submitted.ID.String(), readyAt); err != nil {
		t.Fatal(err)
	}
	readyVersion := persisted.Version + 1
	if _, err := database.ExecContext(ctx, `UPDATE claimbounty_orders SET status='ready_for_export',version=$2,updated_at=$3 WHERE id=$1`, submitted.ID.String(), readyVersion, readyAt); err != nil {
		t.Fatal(err)
	}
	routineRevision := "sha256:" + strings.Repeat("b", 64)
	routineEvidence := strings.Repeat("c", 64)
	administrator := claimID(t, "44444444-4444-4444-8444-444444444444")
	if _, err := database.ExecContext(ctx, `INSERT INTO claimbounty_intakes(order_id,audit_request,scientific_policy,execution_policy,routine_revision,routine_validated_at,routine_evidence_sha256,frozen_by,frozen_at) VALUES($1,'{}','{}','{}',$2,$3,$4,$5,$3)`, submitted.ID.String(), routineRevision, readyAt, routineEvidence, administrator.String()); err != nil {
		t.Fatal(err)
	}
	ready, _, err := store.GetOrder(ctx, submitted.ID)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := ready.QueueExport(ready.Version, now.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	exportHash := sha256.Sum256([]byte("export-readiness-snapshot"))
	exports := []domain.Export{
		{ID: claimID(t, "55555555-5555-4555-8555-555555555551"), OrderID: ready.ID, Status: "queued", RoutineID: "claim-bounty-operations/run-claimbounty-scientific-audit", RoutineRevision: routineRevision, RoutineValidatedAt: readyAt, RoutineEvidenceSHA: routineEvidence, RetentionPolicy: ready.PIIRetention.PolicyVersion, StorageKey: "exports/concurrent-a", CreatedAt: now.Add(4 * time.Second)},
		{ID: claimID(t, "55555555-5555-4555-8555-555555555552"), OrderID: ready.ID, Status: "queued", RoutineID: "claim-bounty-operations/run-claimbounty-scientific-audit", RoutineRevision: routineRevision, RoutineValidatedAt: readyAt, RoutineEvidenceSHA: routineEvidence, RetentionPolicy: ready.PIIRetention.PolicyVersion, StorageKey: "exports/concurrent-b", CreatedAt: now.Add(4 * time.Second)},
	}
	type exportResult struct {
		export domain.Export
		err    error
	}
	startExport := make(chan struct{})
	exportResults := make(chan exportResult, 2)
	var exportReady sync.WaitGroup
	exportReady.Add(2)
	for _, item := range exports {
		go func(item domain.Export) {
			exportReady.Done()
			<-startExport
			result, resultErr := store.QueueExport(ctx, ports.ExportQueueRequest{
				Order: queued, Export: item, ActorID: administrator,
				IdempotencyKey: "export-concurrent-order", RequestHash: exportHash,
			})
			exportResults <- exportResult{export: result, err: resultErr}
		}(item)
	}
	exportReady.Wait()
	close(startExport)
	firstExport, secondExport := <-exportResults, <-exportResults
	if firstExport.err != nil || secondExport.err != nil {
		t.Fatalf("concurrent export errors = %v, %v", firstExport.err, secondExport.err)
	}
	if firstExport.export.ID != secondExport.export.ID {
		t.Fatalf("concurrent export replay returned %s and %s", firstExport.export.ID, secondExport.export.ID)
	}
	replayedExport, ok, err := store.GetIdempotentExport(ctx, administrator, "export-concurrent-order", exportHash)
	if err != nil || !ok || replayedExport.ID != firstExport.export.ID {
		t.Fatalf("response-loss export replay = %t/%s (error %v)", ok, replayedExport.ID, err)
	}
	var exportCount, exportJobs int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM claimbounty_exports WHERE order_id=$1`, ready.ID.String()).Scan(&exportCount); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM claimbounty_outbox WHERE kind='build_export' AND order_id=$1`, ready.ID.String()).Scan(&exportJobs); err != nil {
		t.Fatal(err)
	}
	if exportCount != 1 || exportJobs != 1 {
		t.Fatalf("concurrent export persisted %d exports and %d jobs, want 1/1", exportCount, exportJobs)
	}
}

func TestClaimBountyRestoresBothContractRetentionDispositions(t *testing.T) {
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseURL, cleanup := isolatedSchema(t, ctx, baseURL)
	defer cleanup()
	if err := postgres.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, databaseURL, 4, 5*time.Second, claimTestEmailProtector(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	actor := claimID(t, "11111111-1111-4111-8111-111111111111")
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	for index, disposition := range []string{"hard_delete", "irreversible_anonymize"} {
		orderID := []string{"22222222-2222-4222-8222-222222222221", "22222222-2222-4222-8222-222222222222"}[index]
		reference := []string{"CB-RETENTION001", "CB-RETENTION002"}[index]
		order, err := domain.NewOrder(claimID(t, orderID), actor, "owner@example.test", reference, "Study", "Audit", domain.TargetClaim{Text: "Claim"}, domain.Permissions{}, domain.Privacy{}, now)
		if err != nil {
			t.Fatal(err)
		}
		order.PIIRetention.PolicyVersion = "configured-retention-v2"
		order.PIIRetention.SourceDeleteAfter = now.Add(time.Duration(index+7) * 24 * time.Hour)
		order.PIIRetention.ApplyAfter = now.Add(time.Duration(index+21) * 24 * time.Hour)
		order, err = store.CreateOrder(ctx, ports.IdempotentOrderWrite{Order: order, ActorID: actor, Operation: "create_order", IdempotencyKey: "retention-disposition-" + disposition, RequestHash: sha256.Sum256([]byte(disposition))})
		if err != nil {
			t.Fatal(err)
		}
		order.PIIRetention.Disposition = disposition
		if err := store.SaveOrder(ctx, order, order.Version); err != nil {
			t.Fatal(err)
		}
		restored, err := store.GetOwnedOrder(ctx, actor, order.ID)
		if err != nil {
			t.Fatalf("restore %s: %v", disposition, err)
		}
		if restored.PIIRetention.PolicyVersion != order.PIIRetention.PolicyVersion || restored.PIIRetention.Disposition != disposition ||
			!restored.PIIRetention.SourceDeleteAfter.Equal(order.PIIRetention.SourceDeleteAfter) || !restored.PIIRetention.ApplyAfter.Equal(order.PIIRetention.ApplyAfter) {
			t.Fatalf("restored retention = %+v, want %+v", restored.PIIRetention, order.PIIRetention)
		}
	}
}

func TestClaimBountyIdenticalFilesPromoteAndDeleteIndependentVersions(t *testing.T) {
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseURL, cleanup := isolatedSchema(t, ctx, baseURL)
	defer cleanup()
	if err := postgres.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, databaseURL, 4, 5*time.Second, claimTestEmailProtector(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	storage := newVersionedStorageStub()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	actor := claimID(t, "11111111-1111-4111-8111-111111111111")
	contents := []byte("%PDF identical bytes")
	digestBytes := sha256.Sum256(contents)
	digest := hex.EncodeToString(digestBytes[:])
	orderIDs := []string{"22222222-2222-4222-8222-222222222221", "22222222-2222-4222-8222-222222222222"}
	fileIDs := []string{"33333333-3333-4333-8333-333333333331", "33333333-3333-4333-8333-333333333332"}
	references := []string{"CB-DUPLICATE001", "CB-DUPLICATE002"}
	for index := range orderIDs {
		order, err := domain.NewOrder(claimID(t, orderIDs[index]), actor, "owner@example.test", references[index], "Study", "Audit", domain.TargetClaim{Text: "Claim"}, domain.Permissions{}, domain.Privacy{}, now)
		if err != nil {
			t.Fatal(err)
		}
		order, err = store.CreateOrder(ctx, ports.IdempotentOrderWrite{Order: order, ActorID: actor, Operation: "create_order", IdempotencyKey: []string{"duplicate-create-0001", "duplicate-create-0002"}[index], RequestHash: sha256.Sum256([]byte(orderIDs[index]))})
		if err != nil {
			t.Fatal(err)
		}
		quarantineKey, quarantineVersion := "quarantine/duplicate-"+fileIDs[index], "quarantine-version-"+fileIDs[index]
		storage.seed(quarantineKey, quarantineVersion, contents, digest)
		file, err := domain.NewOrderFile(claimID(t, fileIDs[index]), "primary_paper", "paper.pdf", int64(len(contents)), digest, "application/pdf", quarantineKey, now)
		if err != nil {
			t.Fatal(err)
		}
		file, err = file.ConfirmUpload("etag", quarantineVersion, now)
		if err != nil {
			t.Fatal(err)
		}
		reserved, err := order.ReserveFile(file, order.Version, now)
		if err != nil {
			t.Fatal(err)
		}
		reserved, _, err = store.SaveUploadedFile(ctx, ports.UploadedFileWrite{Write: ports.IdempotentOrderWrite{Order: reserved, ExpectedVersion: order.Version, ActorID: actor, Operation: "reserve_upload", IdempotencyKey: []string{"duplicate-upload-0001", "duplicate-upload-0002"}[index], RequestHash: sha256.Sum256([]byte(fileIDs[index]))}, File: file})
		if err != nil {
			t.Fatal(err)
		}
		submitted, err := reserved.Submit(reserved.Version, "terms-v1", true, now)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.SaveOrderIdempotent(ctx, ports.IdempotentOrderWrite{Order: submitted, ExpectedVersion: reserved.Version, ActorID: actor, Operation: "submit_order", IdempotencyKey: []string{"duplicate-submit-0001", "duplicate-submit-0002"}[index], RequestHash: sha256.Sum256([]byte("submit-" + fileIDs[index]))}); err != nil {
			t.Fatal(err)
		}
	}
	workers, err := services.NewWorkers(store, storage, pdfInspectorStub{}, exportBuilderStub{}, fixedClock{now}, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	workerCtx, stopWorker := context.WithCancel(ctx)
	workerErr := make(chan error, 1)
	go func() { workerErr <- workers.Run(workerCtx) }()
	acceptedKeys := make([]string, 2)
	deadline := time.Now().Add(5 * time.Second)
	for {
		allClean := true
		for index, rawID := range orderIDs {
			order, err := store.GetOwnedOrder(ctx, actor, claimID(t, rawID))
			if err != nil || len(order.Files) != 1 || order.Files[0].Status != "clean" {
				allClean = false
				break
			}
			acceptedKeys[index] = order.Files[0].StorageKey
		}
		if allClean {
			break
		}
		if time.Now().After(deadline) {
			stopWorker()
			t.Fatal("worker did not promote both identical files")
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopWorker()
	if err := <-workerErr; err != nil {
		t.Fatal(err)
	}
	if acceptedKeys[0] == acceptedKeys[1] || acceptedKeys[0] != "accepted/sha256/"+digest+"/"+fileIDs[0] || acceptedKeys[1] != "accepted/sha256/"+digest+"/"+fileIDs[1] {
		t.Fatalf("accepted keys = %#v", acceptedKeys)
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.ExecContext(ctx, `UPDATE claimbounty_orders SET status='expired',source_retention_expires_at=$1,retention_expires_at=$2 WHERE id=ANY($3::uuid[])`, now, now.Add(time.Hour), orderIDs); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AnonymizeExpired(ctx, now, 10); err != nil {
		t.Fatal(err)
	}
	deletedAccepted := 0
	for attempts := 0; attempts < 10 && deletedAccepted < 2; attempts++ {
		id, object, ok, err := store.ClaimDeletion(ctx, now)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		if strings.HasPrefix(object.Key, "accepted/sha256/") {
			other := acceptedKeys[0]
			if object.Key == other {
				other = acceptedKeys[1]
			}
			if deletedAccepted == 0 && !storage.hasKey(other) {
				t.Fatal("deleting one accepted version removed the other file")
			}
			deletedAccepted++
		}
		if err := storage.DeleteVersion(ctx, object.Key, object.Generation); err != nil {
			t.Fatal(err)
		}
		if err := store.FinishDeletion(ctx, id, now); err != nil {
			t.Fatal(err)
		}
	}
	if deletedAccepted != 2 || storage.hasKey(acceptedKeys[0]) || storage.hasKey(acceptedKeys[1]) {
		t.Fatalf("accepted cleanup count/remaining = %d/%t/%t", deletedAccepted, storage.hasKey(acceptedKeys[0]), storage.hasKey(acceptedKeys[1]))
	}
}

func claimTestEmailProtector(t *testing.T) *system.EmailProtector {
	t.Helper()
	protector, err := system.NewEmailProtector(
		base64.StdEncoding.EncodeToString([]byte("test-db-email-encryption-key-000")),
		base64.StdEncoding.EncodeToString([]byte("test-db-email-lookup-hmac-key-00")),
	)
	if err != nil {
		t.Fatal(err)
	}
	return protector
}

func claimID(t *testing.T, raw string) domain.Identifier {
	t.Helper()
	id, err := domain.NewIdentifier(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

type storedVersion struct {
	contents []byte
	metadata ports.ObjectMetadata
}

type versionedStorageStub struct {
	mu      sync.Mutex
	objects map[string]storedVersion
}

func newVersionedStorageStub() *versionedStorageStub {
	return &versionedStorageStub{objects: make(map[string]storedVersion)}
}

func storageVersionKey(key, generation string) string { return key + "@" + generation }

func (storage *versionedStorageStub) seed(key, generation string, contents []byte, digest string) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.objects[storageVersionKey(key, generation)] = storedVersion{contents: append([]byte(nil), contents...), metadata: ports.ObjectMetadata{SizeBytes: int64(len(contents)), SHA256: digest, Generation: generation}}
}

func (storage *versionedStorageStub) Open(_ context.Context, key, generation string) (ports.ObjectReader, ports.ObjectMetadata, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	stored, ok := storage.objects[storageVersionKey(key, generation)]
	if !ok {
		return nil, ports.ObjectMetadata{}, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(stored.contents)), stored.metadata, nil
}

func (storage *versionedStorageStub) PutWriteOnce(_ context.Context, key string, reader io.Reader, size int64, mediaType, expectedSHA string) (ports.ObjectMetadata, error) {
	contents, err := io.ReadAll(reader)
	if err != nil || int64(len(contents)) != size {
		return ports.ObjectMetadata{}, errors.New("invalid promoted object")
	}
	sum := sha256.Sum256(contents)
	if hex.EncodeToString(sum[:]) != expectedSHA {
		return ports.ObjectMetadata{}, errors.New("promoted checksum mismatch")
	}
	generation := "accepted-version-" + strings.TrimPrefix(key[strings.LastIndex(key, "/"):], "/")
	metadata := ports.ObjectMetadata{SizeBytes: size, SHA256: expectedSHA, MediaType: mediaType, Generation: generation, ETag: "accepted-etag"}
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.objects[storageVersionKey(key, generation)] = storedVersion{contents: append([]byte(nil), contents...), metadata: metadata}
	return metadata, nil
}

func (storage *versionedStorageStub) DeleteVersion(_ context.Context, key, generation string) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	delete(storage.objects, storageVersionKey(key, generation))
	return nil
}

func (storage *versionedStorageStub) hasKey(key string) bool {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	for stored := range storage.objects {
		if strings.HasPrefix(stored, key+"@") {
			return true
		}
	}
	return false
}

type pdfInspectorStub struct{}

func (pdfInspectorStub) Inspect(context.Context, io.Reader, int64, string) (string, bool, string, error) {
	return "application/pdf", true, "", nil
}

type exportBuilderStub struct{}

func (exportBuilderStub) Build(context.Context, domain.Order, domain.Export) (ports.ObjectMetadata, error) {
	return ports.ObjectMetadata{}, errors.New("unexpected export")
}

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }
