package services_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
)

var claimNow = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

func TestUploadFilePersistsTheExactWriteOnceObjectGeneration(t *testing.T) {
	t.Parallel()

	order := newClaimOrder(t)
	content := []byte("%PDF-1.7\n")
	digest := strings.Repeat("a", 64)
	repository := &claimRepositoryStub{ownedOrder: order}
	storage := &privateStoreStub{putMetadata: ports.ObjectMetadata{SizeBytes: int64(len(content)), ETag: "etag-1", SHA256: digest, Generation: "version-0001"}}
	service, err := services.NewIntakeService(repository, storage, secureValuesStub{}, claimClock{}, testRetentionContract())
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.UploadFile(context.Background(), services.UploadFileCommand{
		Session: submitterSession(t), OrderID: order.ID, ExpectedVersion: order.Version,
		Role: "primary_paper", OriginalDisplayName: "paper.pdf", SizeBytes: int64(len(content)), SHA256: digest,
		DeclaredMediaType: "application/pdf", IdempotencyKey: "upload-request-0001", Body: bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if storage.putKey != "quarantine/object-1" || storage.putExpectedSHA != digest || !bytes.Equal(storage.putBody, content) {
		t.Fatalf("write-once request = key %q, sha %q, body %q", storage.putKey, storage.putExpectedSHA, storage.putBody)
	}
	if result.File.ObjectGeneration != "version-0001" || repository.uploaded.File.ObjectGeneration != "version-0001" {
		t.Fatalf("recorded generations = result %q, persisted %q", result.File.ObjectGeneration, repository.uploaded.File.ObjectGeneration)
	}
	if repository.uploaded.Write.ExpectedVersion != 1 || result.Order.Version != 2 {
		t.Fatalf("persisted expected/result versions = %d/%d", repository.uploaded.Write.ExpectedVersion, result.Order.Version)
	}
}

func TestUploadFileDeletesTheExactGenerationWhenMetadataDoesNotMatch(t *testing.T) {
	t.Parallel()

	order := newClaimOrder(t)
	content := []byte("%PDF")
	storage := &privateStoreStub{putMetadata: ports.ObjectMetadata{SizeBytes: int64(len(content)), ETag: "etag", SHA256: strings.Repeat("b", 64), Generation: "version-bad"}}
	service, err := services.NewIntakeService(&claimRepositoryStub{ownedOrder: order}, storage, secureValuesStub{}, claimClock{}, testRetentionContract())
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.UploadFile(context.Background(), services.UploadFileCommand{
		Session: submitterSession(t), OrderID: order.ID, ExpectedVersion: 1,
		Role: "primary_paper", OriginalDisplayName: "paper.pdf", SizeBytes: int64(len(content)), SHA256: strings.Repeat("a", 64),
		DeclaredMediaType: "application/pdf", IdempotencyKey: "upload-request-0002", Body: bytes.NewReader(content),
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("UploadFile() error = %v, want validation error", err)
	}
	if storage.deletedKey != "quarantine/object-1" || storage.deletedGeneration != "version-bad" {
		t.Fatalf("cleanup target = %q@%q", storage.deletedKey, storage.deletedGeneration)
	}
}

func TestConcurrentIdempotentUploadsDeleteTheLosingObjectVersion(t *testing.T) {
	t.Parallel()

	order := newClaimOrder(t)
	content := []byte("%PDF")
	digest := strings.Repeat("a", 64)
	repository := newConcurrentUploadRepository(order)
	storage := newConcurrentPrivateStore()
	values := &concurrentSecureValues{}
	service, err := services.NewIntakeService(repository, storage, values, claimClock{}, testRetentionContract())
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		upload services.UploadedFile
		err    error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			upload, err := service.UploadFile(context.Background(), services.UploadFileCommand{
				Session: submitterSession(t), OrderID: order.ID, ExpectedVersion: order.Version,
				Role: "primary_paper", OriginalDisplayName: "paper.pdf", SizeBytes: int64(len(content)), SHA256: digest,
				DeclaredMediaType: "application/pdf", IdempotencyKey: "concurrent-upload-0001", Body: bytes.NewReader(content),
			})
			results <- result{upload: upload, err: err}
		}()
	}
	ready.Wait()
	close(start)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("concurrent UploadFile() errors = %v, %v", first.err, second.err)
	}
	if first.upload.File.ID != second.upload.File.ID || first.upload.File.ObjectGeneration != second.upload.File.ObjectGeneration {
		t.Fatalf("concurrent replay files differ: %#v and %#v", first.upload.File, second.upload.File)
	}
	remaining, deleted := storage.snapshot()
	if len(remaining) != 1 || deleted != 1 {
		t.Fatalf("storage has %d live versions and %d deleted versions, want 1/1", len(remaining), deleted)
	}
	tracked := first.upload.File.StorageKey + "@" + first.upload.File.ObjectGeneration
	if _, ok := remaining[tracked]; !ok {
		t.Fatalf("remaining object = %v, want tracked %q", remaining, tracked)
	}
}

func TestCreateOrderUsesConfiguredServerRetentionSnapshot(t *testing.T) {
	t.Parallel()

	service, err := services.NewIntakeService(&claimRepositoryStub{}, &privateStoreStub{}, secureValuesStub{}, claimClock{}, testRetentionContract())
	if err != nil {
		t.Fatal(err)
	}
	order, err := service.CreateOrder(context.Background(), services.CreateOrderCommand{
		Session: submitterSession(t), Title: "Study", Purpose: "Verify the result",
		TargetClaim: domain.TargetClaim{Text: "The treatment improved scores"}, IdempotencyKey: "create-retention-0001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.PIIRetention.PolicyVersion != "intake-30d-v1" || order.PIIRetention.Disposition != "hard_delete" ||
		!order.PIIRetention.SourceDeleteAfter.Equal(claimNow.Add(14*24*time.Hour)) || !order.PIIRetention.ApplyAfter.Equal(claimNow.Add(30*24*time.Hour)) {
		t.Fatalf("configured retention snapshot = %+v", order.PIIRetention)
	}
}

func TestSubmitFreezesConfiguredServerRetentionDeadlines(t *testing.T) {
	t.Parallel()

	order := newClaimOrder(t)
	file, err := domain.NewOrderFile(mustClaimID(t, "33333333-3333-4333-8333-333333333333"), "primary_paper", "paper.pdf", 4, strings.Repeat("c", 64), "application/pdf", "quarantine/object", claimNow)
	if err != nil {
		t.Fatal(err)
	}
	file, err = file.ConfirmUpload("etag", "version-1", claimNow)
	if err != nil {
		t.Fatal(err)
	}
	order, err = order.ReserveFile(file, order.Version, claimNow)
	if err != nil {
		t.Fatal(err)
	}
	service, err := services.NewIntakeService(&claimRepositoryStub{ownedOrder: order}, &privateStoreStub{}, secureValuesStub{}, claimClock{}, testRetentionContract())
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.SubmitOrder(context.Background(), services.SubmitOrderCommand{
		Session: submitterSession(t), OrderID: order.ID, ExpectedVersion: order.Version,
		TermsAccepted: true, TermsVersion: "terms-v1", UploadsAuthorized: true, AnalysisUseAuthorized: true,
		ExternalRedistributionAuthorized: false, IdempotencyKey: "submit-retention-0001",
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error = %v", err)
	}
	if updated.PIIRetention.PolicyVersion != "intake-30d-v1" || updated.PIIRetention.Disposition != "hard_delete" {
		t.Fatalf("retention identity = %+v", updated.PIIRetention)
	}
	if !updated.PIIRetention.SourceDeleteAfter.Equal(claimNow.Add(14*24*time.Hour)) || !updated.PIIRetention.ApplyAfter.Equal(claimNow.Add(30*24*time.Hour)) {
		t.Fatalf("retention deadlines = %+v", updated.PIIRetention)
	}
}

func TestAdministrationRechecksTheLiveAllowlistForEveryRequest(t *testing.T) {
	t.Parallel()

	policy := &adminPolicyStub{}
	service, err := services.NewAdministrationService(&claimRepositoryStub{}, &privateStoreStub{}, validatorStub{}, policy, secureValuesStub{}, claimClock{}, trustedRoutine())
	if err != nil {
		t.Fatal(err)
	}
	session := administratorSession(t)
	if _, err := service.ListOrders(context.Background(), session, ports.OrderPageRequest{Limit: 20}); err != nil {
		t.Fatalf("first ListOrders() error = %v", err)
	}
	policy.revoked = true
	if _, err := service.ListOrders(context.Background(), session, ports.OrderPageRequest{Limit: 20}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("revoked ListOrders() error = %v, want forbidden", err)
	}
	if policy.authorizeCalls != 2 {
		t.Fatalf("Authorize() calls = %d, want 2", policy.authorizeCalls)
	}
}

func TestAdministrationRejectsClientRoutineOutsideTrustedRegistry(t *testing.T) {
	t.Parallel()

	order := claimOrderReadyForIntake(t)
	repository := &claimRepositoryStub{adminOrder: order}
	service, err := services.NewAdministrationService(repository, &privateStoreStub{}, validatorStub{}, &adminPolicyStub{}, secureValuesStub{}, claimClock{}, trustedRoutine())
	if err != nil {
		t.Fatal(err)
	}
	audit := []byte(`{"caseId":"` + order.ID.String() + `","purpose":"` + order.Purpose + `","targetClaim":{"text":"` + order.TargetClaim.Text + `","source":{"artifact":"` + order.Files[0].CaseBundlePath() + `","location":"` + order.TargetClaim.SourceLocation + `"}},"permissions":{"executeSuppliedCode":false,"externalSearch":false},"privacy":{"containsParticipantLevelData":false,"containsDirectIdentifiers":false},"authority":{"termsVersion":"` + order.TermsVersion + `"},"retention":{"policyVersion":"retention-v1","piiDeleteAfter":"2026-09-30T12:00:00Z","piiDisposition":"hard_delete"}}`)
	_, _, err = service.UpdateIntake(context.Background(), services.AdminIntakeCommand{
		Session: administratorSession(t), OrderID: order.ID, ExpectedVersion: order.Version,
		AuditRequest: audit, ScientificPolicy: []byte(`{}`), ExecutionPolicy: []byte(`{}`),
		RoutineRevision:    "sha256:" + strings.Repeat("a", 64),
		RoutineValidatedAt: claimNow,
		RoutineEvidenceSHA: strings.Repeat("c", 64),
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("UpdateIntake() error = %v, want validation error for untrusted routine record", err)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("SaveOrder() calls = %d, want 0", repository.saveCalls)
	}
}

func TestAdministrationCanCompleteAnOptionalCustomerSourceLocation(t *testing.T) {
	t.Parallel()

	order := claimOrderReadyForIntake(t)
	order.TargetClaim.SourceLocation = ""
	repository := &claimRepositoryStub{adminOrder: order}
	service, err := services.NewAdministrationService(repository, &privateStoreStub{}, validatorStub{}, &adminPolicyStub{}, secureValuesStub{}, claimClock{}, trustedRoutine())
	if err != nil {
		t.Fatal(err)
	}
	audit := []byte(`{"caseId":"` + order.ID.String() + `","purpose":"` + order.Purpose + `","targetClaim":{"text":"` + order.TargetClaim.Text + `","source":{"artifact":"` + order.Files[0].CaseBundlePath() + `","location":"Table 2"}},"permissions":{"executeSuppliedCode":false,"externalSearch":false},"privacy":{"containsParticipantLevelData":false,"containsDirectIdentifiers":false},"authority":{"termsVersion":"` + order.TermsVersion + `"},"retention":{"policyVersion":"intake-30d-v1","sourceDeleteAfter":"2026-09-13T12:00:00Z","piiDeleteAfter":"2026-09-20T12:00:00Z","piiDisposition":"hard_delete"}}`)
	updated, _, err := service.UpdateIntake(context.Background(), services.AdminIntakeCommand{
		Session: administratorSession(t), OrderID: order.ID, ExpectedVersion: order.Version,
		AuditRequest: audit, ScientificPolicy: []byte(`{}`), ExecutionPolicy: []byte(`{}`),
		RoutineRevision: "sha256:" + strings.Repeat("a", 64), RoutineValidatedAt: claimNow,
		RoutineEvidenceSHA: strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatalf("UpdateIntake() error = %v, want admin-completed optional source location", err)
	}
	if updated.Status != "ready_for_export" || repository.saveCalls != 1 {
		t.Fatalf("updated status/save calls = %q/%d", updated.Status, repository.saveCalls)
	}
	wantSource := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	wantPII := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	if !updated.PIIRetention.SourceDeleteAfter.Equal(wantSource) || !updated.PIIRetention.ApplyAfter.Equal(wantPII) {
		t.Fatalf("shortened retention snapshot = %+v, want %s/%s", updated.PIIRetention, wantSource, wantPII)
	}
}

func TestAdministrationMustPreserveSubmissionRetentionIdentity(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, policyVersion, disposition string
	}{
		{name: "policy version", policyVersion: "different-policy-v1", disposition: "hard_delete"},
		{name: "disposition", policyVersion: "intake-30d-v1", disposition: "irreversible_anonymize"},
	} {
		t.Run(test.name, func(t *testing.T) {
			order := claimOrderReadyForIntake(t)
			repository := &claimRepositoryStub{adminOrder: order}
			service, err := services.NewAdministrationService(repository, &privateStoreStub{}, validatorStub{}, &adminPolicyStub{}, secureValuesStub{}, claimClock{}, trustedRoutine())
			if err != nil {
				t.Fatal(err)
			}
			audit := []byte(fmt.Sprintf(`{"caseId":%q,"purpose":%q,"targetClaim":{"text":%q,"source":{"artifact":%q,"location":%q}},"permissions":{"executeSuppliedCode":false,"externalSearch":false},"privacy":{"containsParticipantLevelData":false,"containsDirectIdentifiers":false},"authority":{"termsVersion":%q},"retention":{"policyVersion":%q,"sourceDeleteAfter":"2026-09-13T12:00:00Z","piiDeleteAfter":"2026-09-20T12:00:00Z","piiDisposition":%q}}`, order.ID.String(), order.Purpose, order.TargetClaim.Text, order.Files[0].CaseBundlePath(), order.TargetClaim.SourceLocation, order.TermsVersion, test.policyVersion, test.disposition))
			_, _, err = service.UpdateIntake(context.Background(), services.AdminIntakeCommand{
				Session: administratorSession(t), OrderID: order.ID, ExpectedVersion: order.Version,
				AuditRequest: audit, ScientificPolicy: []byte(`{}`), ExecutionPolicy: []byte(`{}`),
				RoutineRevision: "sha256:" + strings.Repeat("a", 64), RoutineValidatedAt: claimNow,
				RoutineEvidenceSHA: strings.Repeat("b", 64),
			})
			var validation *domain.ValidationError
			if !errors.As(err, &validation) || len(validation.Issues()) != 1 || validation.Issues()[0].Code != "snapshot_mismatch" {
				t.Fatalf("UpdateIntake() error = %v, want frozen retention identity mismatch", err)
			}
			if repository.saveCalls != 0 {
				t.Fatalf("SaveOrder() calls = %d, want 0", repository.saveCalls)
			}
		})
	}
}

func TestAdministrationCannotExtendSubmissionFrozenRetention(t *testing.T) {
	t.Parallel()

	order := claimOrderReadyForIntake(t)
	repository := &claimRepositoryStub{adminOrder: order}
	service, err := services.NewAdministrationService(repository, &privateStoreStub{}, validatorStub{}, &adminPolicyStub{}, secureValuesStub{}, claimClock{}, trustedRoutine())
	if err != nil {
		t.Fatal(err)
	}
	audit := []byte(`{"caseId":"` + order.ID.String() + `","purpose":"` + order.Purpose + `","targetClaim":{"text":"` + order.TargetClaim.Text + `","source":{"artifact":"` + order.Files[0].CaseBundlePath() + `","location":"` + order.TargetClaim.SourceLocation + `"}},"permissions":{"executeSuppliedCode":false,"externalSearch":false},"privacy":{"containsParticipantLevelData":false,"containsDirectIdentifiers":false},"authority":{"termsVersion":"` + order.TermsVersion + `"},"retention":{"policyVersion":"intake-30d-v1","sourceDeleteAfter":"2026-09-30T12:00:00Z","piiDeleteAfter":"2026-09-30T12:00:00Z","piiDisposition":"hard_delete"}}`)
	_, _, err = service.UpdateIntake(context.Background(), services.AdminIntakeCommand{
		Session: administratorSession(t), OrderID: order.ID, ExpectedVersion: order.Version,
		AuditRequest: audit, ScientificPolicy: []byte(`{}`), ExecutionPolicy: []byte(`{}`),
		RoutineRevision: "sha256:" + strings.Repeat("a", 64), RoutineValidatedAt: claimNow,
		RoutineEvidenceSHA: strings.Repeat("b", 64),
	})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || len(validation.Issues()) != 1 || validation.Issues()[0].Code != "exceeds_server_ceiling" {
		t.Fatalf("UpdateIntake() error = %v, want server-ceiling validation", err)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("SaveOrder() calls = %d, want 0", repository.saveCalls)
	}
}

func TestIdentityBoundsAuthenticatedScopeBySessionIPAndEmail(t *testing.T) {
	t.Parallel()

	repository := &identityRepositoryStub{}
	service, err := services.NewIdentityService(repository, verificationMailerStub{}, secureValuesStub{}, claimClock{}, &adminPolicyStub{}, strings.Repeat("p", 32))
	if err != nil {
		t.Fatal(err)
	}
	err = service.EnforceSessionRateLimit(context.Background(), submitterSession(t), services.SessionRateLimit{Scope: "order_upload", IPPrefix: "192.0.2.0/24", Now: claimNow})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := repository.scopes, []string{"order_upload_session", "order_upload_ip", "order_upload_email"}; !slices.Equal(got, want) {
		t.Fatalf("rate limit scopes = %v, want %v", got, want)
	}
	for index := range repository.windows {
		if repository.windows[index] != 15*time.Minute || repository.limits[index] != 20 {
			t.Fatalf("rate policy = %s/%d, want 15m/20", repository.windows[index], repository.limits[index])
		}
	}
}

func TestWorkersRetryAfterTransientStepFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	repository := &transientWorkerRepository{claimRepositoryStub: &claimRepositoryStub{}, cancel: cancel}
	workers, err := services.NewWorkers(repository, &privateStoreStub{}, workerInspectorStub{}, workerBuilderStub{}, claimClock{}, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := workers.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v, want transient failure to be retried", err)
	}
	if repository.calls != 2 {
		t.Fatalf("AnonymizeExpired() calls = %d, want retry after first failure", repository.calls)
	}
}

type claimClock struct{}

func (claimClock) Now() time.Time { return claimNow }

func testRetentionContract() services.RetentionContract {
	return services.RetentionContract{PolicyVersion: "intake-30d-v1", SourceDuration: 14 * 24 * time.Hour, PIIDuration: 30 * 24 * time.Hour}
}

type identityRepositoryStub struct {
	scopes  []string
	windows []time.Duration
	limits  []int
}

func (repository *identityRepositoryStub) EnforceRateLimit(_ context.Context, scope string, _ [32]byte, _ time.Time, window time.Duration, limit int) error {
	repository.scopes = append(repository.scopes, scope)
	repository.windows = append(repository.windows, window)
	repository.limits = append(repository.limits, limit)
	return nil
}
func (*identityRepositoryStub) CreateChallenge(context.Context, ports.Challenge) error { return nil }
func (*identityRepositoryStub) ExchangeChallenge(context.Context, string, domain.Audience, [32]byte, domain.Identifier, [32]byte, [32]byte, string, time.Time, time.Time) (ports.SessionCredential, error) {
	return ports.SessionCredential{}, nil
}
func (*identityRepositoryStub) GetSession(context.Context, [32]byte, time.Time) (ports.SessionCredential, error) {
	return ports.SessionCredential{}, nil
}
func (*identityRepositoryStub) RotateCSRF(context.Context, domain.Identifier, [32]byte, [32]byte, time.Time) error {
	return nil
}
func (*identityRepositoryStub) RevokeSession(context.Context, [32]byte, time.Time) error { return nil }

type verificationMailerStub struct{}

func (verificationMailerStub) SendVerification(context.Context, string, domain.Audience, string, time.Time) error {
	return nil
}

func trustedRoutine() services.TrustedRoutineContract {
	return services.TrustedRoutineContract{
		Revision:       "sha256:" + strings.Repeat("a", 64),
		ValidatedAt:    claimNow,
		EvidenceSHA256: strings.Repeat("b", 64),
	}
}

type secureValuesStub struct{}

func (secureValuesStub) NewIdentifier(context.Context) (domain.Identifier, error) {
	return claimID("44444444-4444-4444-8444-444444444444")
}
func (secureValuesStub) NewOpaqueToken(context.Context, int) (string, error) {
	return strings.Repeat("t", 43), nil
}
func (secureValuesStub) NewChallengeCode(context.Context) (string, error) { return "123456", nil }
func (secureValuesStub) NewObjectKey(context.Context, string) (string, error) {
	return "quarantine/object-1", nil
}
func (secureValuesStub) NewPublicReference(context.Context) (string, error) {
	return "CB-NEW123ABC456", nil
}

type privateStoreStub struct {
	putMetadata                   ports.ObjectMetadata
	putKey, putExpectedSHA        string
	putBody                       []byte
	deletedKey, deletedGeneration string
	openReader                    ports.ObjectReader
	openMetadata                  ports.ObjectMetadata
}

func (store *privateStoreStub) Open(context.Context, string, string) (ports.ObjectReader, ports.ObjectMetadata, error) {
	if store.openReader == nil {
		store.openReader = io.NopCloser(bytes.NewReader(nil))
	}
	return store.openReader, store.openMetadata, nil
}
func (store *privateStoreStub) PutWriteOnce(_ context.Context, key string, reader io.Reader, _ int64, _ string, expectedSHA string) (ports.ObjectMetadata, error) {
	store.putKey, store.putExpectedSHA = key, expectedSHA
	store.putBody, _ = io.ReadAll(reader)
	return store.putMetadata, nil
}
func (store *privateStoreStub) DeleteVersion(_ context.Context, key, generation string) error {
	store.deletedKey, store.deletedGeneration = key, generation
	return nil
}

type concurrentSecureValues struct {
	secureValuesStub
	mu            sync.Mutex
	identifierSeq int
	objectSeq     int
}

func (values *concurrentSecureValues) NewIdentifier(context.Context) (domain.Identifier, error) {
	values.mu.Lock()
	defer values.mu.Unlock()
	values.identifierSeq++
	return claimID(fmt.Sprintf("44444444-4444-4444-8444-%012d", values.identifierSeq))
}

func (values *concurrentSecureValues) NewObjectKey(context.Context, string) (string, error) {
	values.mu.Lock()
	defer values.mu.Unlock()
	values.objectSeq++
	return fmt.Sprintf("quarantine/object-%d", values.objectSeq), nil
}

type concurrentPrivateStore struct {
	privateStoreStub
	mu      sync.Mutex
	objects map[string]struct{}
	deleted int
}

func newConcurrentPrivateStore() *concurrentPrivateStore {
	return &concurrentPrivateStore{objects: map[string]struct{}{}}
}

func (store *concurrentPrivateStore) PutWriteOnce(_ context.Context, key string, reader io.Reader, size int64, _ string, expectedSHA string) (ports.ObjectMetadata, error) {
	body, err := io.ReadAll(reader)
	if err != nil || int64(len(body)) != size {
		return ports.ObjectMetadata{}, errors.New("test storage: body mismatch")
	}
	generation := "version-" + strings.TrimPrefix(key, "quarantine/object-")
	store.mu.Lock()
	store.objects[key+"@"+generation] = struct{}{}
	store.mu.Unlock()
	return ports.ObjectMetadata{SizeBytes: size, ETag: "etag-" + generation, SHA256: expectedSHA, Generation: generation}, nil
}

func (store *concurrentPrivateStore) DeleteVersion(_ context.Context, key, generation string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	object := key + "@" + generation
	if _, ok := store.objects[object]; !ok {
		return errors.New("test storage: deleting unknown version")
	}
	delete(store.objects, object)
	store.deleted++
	return nil
}

func (store *concurrentPrivateStore) snapshot() (map[string]struct{}, int) {
	store.mu.Lock()
	defer store.mu.Unlock()
	objects := make(map[string]struct{}, len(store.objects))
	for object := range store.objects {
		objects[object] = struct{}{}
	}
	return objects, store.deleted
}

type concurrentUploadRepository struct {
	*claimRepositoryStub
	ready     sync.WaitGroup
	mu        sync.Mutex
	persisted *ports.UploadedFileWrite
}

func newConcurrentUploadRepository(order domain.Order) *concurrentUploadRepository {
	repository := &concurrentUploadRepository{claimRepositoryStub: &claimRepositoryStub{ownedOrder: order}}
	repository.ready.Add(2)
	return repository
}

func (repository *concurrentUploadRepository) SaveUploadedFile(_ context.Context, upload ports.UploadedFileWrite) (domain.Order, domain.OrderFile, error) {
	repository.ready.Done()
	repository.ready.Wait()
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if repository.persisted == nil {
		persisted := upload
		repository.persisted = &persisted
	}
	return repository.persisted.Write.Order, repository.persisted.File, nil
}

type adminPolicyStub struct {
	revoked        bool
	authorizeCalls int
}

func (policy *adminPolicyStub) Authorize(context.Context, string, string) error {
	policy.authorizeCalls++
	if policy.revoked {
		return domain.ErrForbidden
	}
	return nil
}
func (*adminPolicyStub) Version() string          { return "authorization-v1" }
func (*adminPolicyStub) AllowlistVersion() string { return "allowlist-v1" }

type validatorStub struct{}

func (validatorStub) ValidateAuditRequest([]byte) error     { return nil }
func (validatorStub) ValidateScientificPolicy([]byte) error { return nil }
func (validatorStub) ValidateExecutionPolicy([]byte) error  { return nil }
func (validatorStub) ValidateCaseManifest([]byte) error     { return nil }

type workerInspectorStub struct{}

func (workerInspectorStub) Inspect(context.Context, io.Reader, int64, string) (string, bool, string, error) {
	return "application/pdf", true, "", nil
}

type workerBuilderStub struct{}

func (workerBuilderStub) Build(context.Context, domain.Order, domain.Export) (ports.ObjectMetadata, error) {
	return ports.ObjectMetadata{}, nil
}

type claimRepositoryStub struct {
	ownedOrder domain.Order
	adminOrder domain.Order
	uploaded   ports.UploadedFileWrite
	saveCalls  int
}

func (repo *claimRepositoryStub) CreateOrder(_ context.Context, write ports.IdempotentOrderWrite) (domain.Order, error) {
	return write.Order, nil
}
func (repo *claimRepositoryStub) GetOwnedOrder(context.Context, domain.Identifier, domain.Identifier) (domain.Order, error) {
	return repo.ownedOrder, nil
}
func (repo *claimRepositoryStub) GetOrder(context.Context, domain.Identifier) (domain.Order, []domain.Export, error) {
	return repo.adminOrder, nil, nil
}
func (*claimRepositoryStub) ListOrders(context.Context, ports.OrderPageRequest) (ports.OrderPage, error) {
	return ports.OrderPage{}, nil
}
func (repo *claimRepositoryStub) SaveOrder(context.Context, domain.Order, uint64) error {
	repo.saveCalls++
	return nil
}
func (*claimRepositoryStub) SaveOrderIdempotent(_ context.Context, write ports.IdempotentOrderWrite) (domain.Order, error) {
	return write.Order, nil
}
func (*claimRepositoryStub) GetIdempotentOrder(context.Context, domain.Identifier, string, string, [32]byte) (domain.Order, bool, error) {
	return domain.Order{}, false, nil
}
func (repo *claimRepositoryStub) SaveUploadedFile(_ context.Context, upload ports.UploadedFileWrite) (domain.Order, domain.OrderFile, error) {
	repo.uploaded = upload
	return upload.Write.Order, upload.File, nil
}
func (*claimRepositoryStub) GetIdempotentFile(context.Context, domain.Identifier, string, [32]byte) (domain.Order, domain.OrderFile, bool, error) {
	return domain.Order{}, domain.OrderFile{}, false, nil
}
func (*claimRepositoryStub) RemoveFile(context.Context, domain.Order, domain.OrderFile, uint64, domain.Identifier) error {
	return nil
}
func (*claimRepositoryStub) QueueExport(_ context.Context, request ports.ExportQueueRequest) (domain.Export, error) {
	return request.Export, nil
}
func (*claimRepositoryStub) GetIdempotentExport(context.Context, domain.Identifier, string, [32]byte) (domain.Export, bool, error) {
	return domain.Export{}, false, nil
}
func (*claimRepositoryStub) GetExport(context.Context, domain.Identifier, domain.Identifier) (domain.Export, error) {
	return domain.Export{}, domain.ErrExportNotFound
}
func (*claimRepositoryStub) ClaimInspection(context.Context, time.Time) (domain.Order, domain.OrderFile, bool, error) {
	return domain.Order{}, domain.OrderFile{}, false, nil
}
func (*claimRepositoryStub) FinishInspection(context.Context, domain.Order, domain.OrderFile, uint64, ports.ExpiredObject) error {
	return nil
}
func (*claimRepositoryStub) ClaimExport(context.Context, time.Time) (domain.Order, domain.Export, bool, error) {
	return domain.Order{}, domain.Export{}, false, nil
}
func (*claimRepositoryStub) FinishExport(context.Context, domain.Order, domain.Export, uint64) error {
	return nil
}
func (*claimRepositoryStub) FailExport(context.Context, domain.Identifier, domain.Identifier, string, time.Time) error {
	return nil
}
func (*claimRepositoryStub) ClaimDeletion(context.Context, time.Time) (int64, ports.ExpiredObject, bool, error) {
	return 0, ports.ExpiredObject{}, false, nil
}
func (*claimRepositoryStub) FinishDeletion(context.Context, int64, time.Time) error { return nil }
func (*claimRepositoryStub) AnonymizeExpired(context.Context, time.Time, int) ([]ports.ExpiredObject, error) {
	return nil, nil
}
func (*claimRepositoryStub) CleanupExpiredIdentityAndAbandoned(context.Context, time.Time, time.Time) error {
	return nil
}

type transientWorkerRepository struct {
	*claimRepositoryStub
	cancel context.CancelFunc
	calls  int
}

func (repository *transientWorkerRepository) AnonymizeExpired(context.Context, time.Time, int) ([]ports.ExpiredObject, error) {
	repository.calls++
	if repository.calls == 1 {
		return nil, errors.New("transient database failure")
	}
	repository.cancel()
	return nil, nil
}

func newClaimOrder(t *testing.T) domain.Order {
	t.Helper()
	order, err := domain.NewOrder(
		mustClaimID(t, "11111111-1111-4111-8111-111111111111"),
		mustClaimID(t, "22222222-2222-4222-8222-222222222222"),
		"researcher@example.test", "CB-ABC123DEF456", "Study", "Verify the result",
		domain.TargetClaim{Text: "The treatment improved scores", SourceLocation: "Table 2"},
		domain.Permissions{}, domain.Privacy{}, claimNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	return order
}

func claimOrderReadyForIntake(t *testing.T) domain.Order {
	t.Helper()
	order := newClaimOrder(t)
	file, err := domain.NewOrderFile(mustClaimID(t, "33333333-3333-4333-8333-333333333333"), "primary_paper", "paper.pdf", 4, strings.Repeat("c", 64), "application/pdf", "quarantine/object", claimNow)
	if err != nil {
		t.Fatal(err)
	}
	file, err = file.ConfirmUpload("etag", "version-1", claimNow)
	if err != nil {
		t.Fatal(err)
	}
	order, err = order.ReserveFile(file, order.Version, claimNow)
	if err != nil {
		t.Fatal(err)
	}
	order, err = order.Submit(order.Version, "terms-v1", true, claimNow)
	if err != nil {
		t.Fatal(err)
	}
	file, err = file.InspectionResult(true, "application/pdf", "", claimNow)
	if err != nil {
		t.Fatal(err)
	}
	order, err = order.ReplaceFile(file, order.Version, claimNow)
	if err != nil {
		t.Fatal(err)
	}
	order, err = order.ReconcileInspection(claimNow)
	if err != nil {
		t.Fatal(err)
	}
	return order
}

func submitterSession(t *testing.T) domain.Session {
	t.Helper()
	session, err := domain.NewSession(mustClaimID(t, "55555555-5555-4555-8555-555555555555"), mustClaimID(t, "22222222-2222-4222-8222-222222222222"), "researcher@example.test", domain.SubmitterAudience, "authorization-v1", claimNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func administratorSession(t *testing.T) domain.Session {
	t.Helper()
	session, err := domain.NewSession(mustClaimID(t, "66666666-6666-4666-8666-666666666666"), mustClaimID(t, "77777777-7777-4777-8777-777777777777"), "admin@example.test", domain.AdminAudience, "authorization-v1", claimNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func mustClaimID(t *testing.T, raw string) domain.Identifier {
	t.Helper()
	id, err := claimID(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func claimID(raw string) (domain.Identifier, error) { return domain.NewIdentifier(raw) }
