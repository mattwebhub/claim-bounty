package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
)

func TestExportContentDigestUsesRFC9530SHA256Format(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte("archive bytes"))
	got, err := contentDigestHeader(hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	want := "sha-256=:" + base64.StdEncoding.EncodeToString(sum[:]) + ":"
	if got != want {
		t.Fatalf("Content-Digest = %q, want %q", got, want)
	}
	if _, err := contentDigestHeader(strings.Repeat("A", 64)); err == nil {
		t.Fatal("contentDigestHeader accepted non-lowercase hexadecimal")
	}
}

func TestSourceAndExportDownloadsReturnRepresentationContentDigest(t *testing.T) {
	t.Parallel()

	orderID := claimHTTPID(t, "11111111-1111-4111-8111-111111111111")
	fileID := claimHTTPID(t, "44444444-4444-4444-8444-444444444444")
	exportID := claimHTTPID(t, "55555555-5555-4555-8555-555555555555")
	source := []byte("source bytes")
	archive := []byte("archive bytes")
	sourceSHA := sha256.Sum256(source)
	archiveSHA := sha256.Sum256(archive)
	admin := &downloadAdministrationStub{
		file:   domain.OrderFile{ID: fileID, OriginalDisplayName: "study.pdf", SizeBytes: int64(len(source)), SHA256: hex.EncodeToString(sourceSHA[:])},
		export: domain.Export{ID: exportID, SizeBytes: int64(len(archive)), SHA256: hex.EncodeToString(archiveSHA[:])},
		source: source, archive: archive,
	}
	routes, err := NewClaimRoutes(&claimIdentityStub{session: claimSubmitterSession(t)}, &claimIntakeStub{}, admin, slog.New(slog.DiscardHandler), 1<<20, "https://claim.example")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux)

	for _, test := range []struct {
		name, path, sha string
		body            []byte
	}{
		{name: "source", path: "/api/v1/admin/orders/" + orderID.String() + "/files/" + fileID.String() + "/content", sha: admin.file.SHA256, body: source},
		{name: "export", path: "/api/v1/admin/exports/" + exportID.String() + "/download", sha: admin.export.SHA256, body: archive},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token-value-000000000000000"})
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			wantDigest, err := contentDigestHeader(test.sha)
			if err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Digest") != wantDigest || !bytes.Equal(recorder.Body.Bytes(), test.body) {
				t.Fatalf("download = status %d, digest %q, body %q", recorder.Code, recorder.Header().Get("Content-Digest"), recorder.Body.Bytes())
			}
		})
	}
}

func TestClaimBountyCreateOrderRequiresOriginSessionCSRFAndIdempotency(t *testing.T) {
	t.Parallel()

	identity := &claimIdentityStub{session: claimSubmitterSession(t)}
	intake := &claimIntakeStub{order: claimHTTPOrder(t)}
	handler := claimHandler(t, identity, intake)
	body := `{"title":"Study","purpose":"Verify the result","targetClaim":{"text":"The treatment improved scores","sourceLocation":"Table 2"},"permissions":{"executeSuppliedCode":false,"externalSearch":false},"privacy":{"containsParticipantLevelData":false,"containsDirectIdentifiers":false}}`

	for _, test := range []struct {
		name, origin, csrf, idempotency string
		wantStatus                      int
	}{
		{"missing origin", "", "csrf-token-value-0000000000000000", "create-order-key-0001", http.StatusForbidden},
		{"wrong origin", "https://attacker.example", "csrf-token-value-0000000000000000", "create-order-key-0001", http.StatusForbidden},
		{"missing csrf", "https://claim.example", "", "create-order-key-0001", http.StatusForbidden},
		{"missing idempotency", "https://claim.example", "csrf-token-value-0000000000000000", "", http.StatusUnprocessableEntity},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", test.origin)
			request.Header.Set("X-Csrf-Token", test.csrf)
			request.Header.Set("Idempotency-Key", test.idempotency)
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token-value-000000000000000"})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("response = %d %s, want %d", recorder.Code, recorder.Body.String(), test.wantStatus)
			}
		})
	}
	if intake.createCalls != 0 {
		t.Fatalf("invalid requests reached CreateOrder() %d times", intake.createCalls)
	}
}

func TestClaimBountyCreateOrderReturnsStrongETagAndForwardsIdempotency(t *testing.T) {
	t.Parallel()

	identity := &claimIdentityStub{session: claimSubmitterSession(t)}
	intake := &claimIntakeStub{order: claimHTTPOrder(t)}
	handler := claimHandler(t, identity, intake)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(`{"title":"Study","purpose":"Verify the result","targetClaim":{"text":"The treatment improved scores","sourceLocation":"Table 2"},"permissions":{"executeSuppliedCode":false,"externalSearch":false},"privacy":{"containsParticipantLevelData":false,"containsDirectIdentifiers":false}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://claim.example")
	request.Header.Set("X-Csrf-Token", "csrf-token-value-0000000000000000")
	request.Header.Set("Idempotency-Key", "create-order-key-0001")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token-value-000000000000000"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated || recorder.Header().Get("ETag") != `"1"` {
		t.Fatalf("response = %d, ETag %q, body %s", recorder.Code, recorder.Header().Get("ETag"), recorder.Body.String())
	}
	if recorder.Header().Get("Location") != "/api/v1/orders/11111111-1111-4111-8111-111111111111" {
		t.Fatalf("Location = %q", recorder.Header().Get("Location"))
	}
	if intake.createCalls != 1 || intake.createCommand.IdempotencyKey != "create-order-key-0001" {
		t.Fatalf("CreateOrder() calls/key = %d/%q", intake.createCalls, intake.createCommand.IdempotencyKey)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"sourceDeleteAfter":"2026-09-29T12:00:00Z"`) || !strings.Contains(body, `"piiDeleteAfter":"2026-09-29T12:00:00Z"`) {
		t.Fatalf("response does not expose persisted source/PII retention deadlines: %s", body)
	}
}

func TestClaimBountyUploadRejectsTrailingMultipartParts(t *testing.T) {
	t.Parallel()

	identity := &claimIdentityStub{session: claimSubmitterSession(t)}
	intake := &claimIntakeStub{order: claimHTTPOrder(t)}
	handler := claimHandler(t, identity, intake)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range map[string]string{
		"role": "primary_paper", "originalDisplayName": "paper.pdf", "sizeBytes": "4",
		"expectedSha256": strings.Repeat("a", 64), "declaredMediaType": "application/pdf",
	} {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	first, err := writer.CreateFormFile("file", "paper.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = first.Write([]byte("%PDF"))
	second, err := writer.CreateFormFile("file", "ignored.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = second.Write([]byte("extra"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/11111111-1111-4111-8111-111111111111/files", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Origin", "https://claim.example")
	request.Header.Set("X-Csrf-Token", "csrf-token-value-0000000000000000")
	request.Header.Set("Idempotency-Key", "upload-order-key-0001")
	request.Header.Set("If-Match", `"1"`)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token-value-000000000000000"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("response = %d %s, want rejection of a second file part", recorder.Code, recorder.Body.String())
	}
}

func TestClaimBountyClientPrefixTrustsOnlyConfiguredProxy(t *testing.T) {
	t.Parallel()

	routes, err := NewClaimRoutes(&claimIdentityStub{session: claimSubmitterSession(t)}, &claimIntakeStub{}, claimAdministrationStub{}, slog.New(slog.DiscardHandler), 1<<20, "https://claim.example", "10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	direct := httptest.NewRequest(http.MethodGet, "/", nil)
	direct.RemoteAddr = "192.0.2.44:1234"
	direct.Header.Set("X-Forwarded-For", "203.0.113.8")
	if got := routes.clientPrefix(direct); got != "192.0.2.0/24" {
		t.Fatalf("direct spoofed prefix = %q", got)
	}
	proxied := httptest.NewRequest(http.MethodGet, "/", nil)
	proxied.RemoteAddr = "10.1.2.3:1234"
	proxied.Header.Set("X-Forwarded-For", "203.0.113.8")
	if got := routes.clientPrefix(proxied); got != "203.0.113.0/24" {
		t.Fatalf("trusted forwarded prefix = %q", got)
	}
}

func TestClaimBountyLogoutRequiresCSRFRevokesAndPreventsCaching(t *testing.T) {
	t.Parallel()
	identity := &claimIdentityStub{session: claimSubmitterSession(t)}
	handler := claimHandler(t, identity, &claimIntakeStub{})
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/session", nil)
	request.Header.Set("Origin", "https://claim.example")
	request.Header.Set("X-Csrf-Token", "csrf-token-value-0000000000000000")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token-value-000000000000000"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || identity.logoutCalls != 1 {
		t.Fatalf("logout response/calls = %d/%d", recorder.Code, identity.logoutCalls)
	}
	if recorder.Header().Get("Cache-Control") != "private, no-store" || recorder.Header().Get("Clear-Site-Data") != `"cookies"` {
		t.Fatalf("logout security headers = %#v", recorder.Header())
	}
}

func claimHandler(t *testing.T, identity IdentityActions, intake IntakeActions) http.Handler {
	t.Helper()
	routes, err := NewClaimRoutes(identity, intake, claimAdministrationStub{}, slog.New(slog.DiscardHandler), 1<<20, "https://claim.example")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux)
	return mux
}

type claimIdentityStub struct {
	session     domain.Session
	logoutCalls int
}

func (*claimIdentityStub) RequestVerification(context.Context, services.RequestVerificationCommand) error {
	return nil
}
func (stub *claimIdentityStub) ConfirmVerification(context.Context, services.ConfirmVerificationCommand) (services.EstablishedSession, error) {
	return services.EstablishedSession{Session: stub.session}, nil
}
func (stub *claimIdentityStub) Authenticate(context.Context, string, time.Time) (domain.Session, error) {
	return stub.session, nil
}
func (stub *claimIdentityStub) AuthorizeCSRF(_ context.Context, _, csrf string, _ time.Time) (domain.Session, error) {
	if csrf != "csrf-token-value-0000000000000000" {
		return domain.Session{}, domain.ErrForbidden
	}
	return stub.session, nil
}
func (stub *claimIdentityStub) RefreshSession(context.Context, string, time.Time) (services.EstablishedSession, error) {
	return services.EstablishedSession{Session: stub.session}, nil
}
func (stub *claimIdentityStub) Logout(context.Context, string, time.Time) error {
	stub.logoutCalls++
	return nil
}
func (*claimIdentityStub) EnforceSessionRateLimit(context.Context, domain.Session, services.SessionRateLimit) error {
	return nil
}

type claimIntakeStub struct {
	order         domain.Order
	createCommand services.CreateOrderCommand
	createCalls   int
}

func (stub *claimIntakeStub) CreateOrder(_ context.Context, command services.CreateOrderCommand) (domain.Order, error) {
	stub.createCalls++
	stub.createCommand = command
	return stub.order, nil
}
func (stub *claimIntakeStub) GetOwnedOrder(context.Context, domain.Session, domain.Identifier) (domain.Order, error) {
	return stub.order, nil
}
func (*claimIntakeStub) UploadFile(context.Context, services.UploadFileCommand) (services.UploadedFile, error) {
	return services.UploadedFile{}, nil
}
func (*claimIntakeStub) DeleteFile(context.Context, services.DeleteFileCommand) error { return nil }
func (stub *claimIntakeStub) SubmitOrder(context.Context, services.SubmitOrderCommand) (domain.Order, error) {
	return stub.order, nil
}

type claimAdministrationStub struct{}

func (claimAdministrationStub) ListOrders(context.Context, domain.Session, services.OrderPageRequest) (services.OrderPage, error) {
	return services.OrderPage{}, nil
}
func (claimAdministrationStub) GetOrder(context.Context, domain.Session, domain.Identifier) (domain.Order, []domain.Export, error) {
	return domain.Order{}, nil, nil
}
func (claimAdministrationStub) UpdateIntake(context.Context, services.AdminIntakeCommand) (domain.Order, []domain.Export, error) {
	return domain.Order{}, nil, nil
}
func (claimAdministrationStub) CreateExport(context.Context, services.CreateExportCommand) (domain.Export, error) {
	return domain.Export{}, nil
}
func (claimAdministrationStub) GetExport(context.Context, domain.Session, domain.Identifier, domain.Identifier) (domain.Export, error) {
	return domain.Export{}, nil
}
func (claimAdministrationStub) OpenFile(context.Context, domain.Session, domain.Identifier, domain.Identifier) (services.ObjectReader, domain.OrderFile, error) {
	return nil, domain.OrderFile{}, nil
}
func (claimAdministrationStub) OpenExport(context.Context, domain.Session, domain.Identifier) (services.ObjectReader, domain.Export, error) {
	return nil, domain.Export{}, nil
}

type downloadAdministrationStub struct {
	claimAdministrationStub
	file            domain.OrderFile
	export          domain.Export
	source, archive []byte
}

func (stub *downloadAdministrationStub) OpenFile(context.Context, domain.Session, domain.Identifier, domain.Identifier) (services.ObjectReader, domain.OrderFile, error) {
	return io.NopCloser(bytes.NewReader(stub.source)), stub.file, nil
}

func (stub *downloadAdministrationStub) OpenExport(context.Context, domain.Session, domain.Identifier) (services.ObjectReader, domain.Export, error) {
	return io.NopCloser(bytes.NewReader(stub.archive)), stub.export, nil
}

func claimSubmitterSession(t *testing.T) domain.Session {
	t.Helper()
	session, err := domain.NewSession(claimHTTPID(t, "22222222-2222-4222-8222-222222222222"), claimHTTPID(t, "33333333-3333-4333-8333-333333333333"), "researcher@example.test", domain.SubmitterAudience, "authorization-v1", time.Date(2026, 8, 30, 13, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func claimHTTPOrder(t *testing.T) domain.Order {
	t.Helper()
	order, err := domain.NewOrder(claimHTTPID(t, "11111111-1111-4111-8111-111111111111"), claimHTTPID(t, "33333333-3333-4333-8333-333333333333"), "researcher@example.test", "CB-ABC123DEF456", "Study", "Verify the result", domain.TargetClaim{Text: "The treatment improved scores", SourceLocation: "Table 2"}, domain.Permissions{}, domain.Privacy{}, time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return order
}

func claimHTTPID(t *testing.T, raw string) domain.Identifier {
	t.Helper()
	id, err := domain.NewIdentifier(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
