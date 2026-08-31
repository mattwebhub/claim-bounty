package bootstrap

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi"
)

func TestClaimBountyCompiledProcessCreatesOwnedOrder(t *testing.T) {
	session := processSession(t)
	identity := processIdentity{session: session}
	intake := &processIntake{session: session}
	routes, err := httpapi.NewClaimRoutes(identity, intake, processAdmin{}, slog.New(slog.DiscardHandler), 1<<20, "https://claim.example.test")
	if err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(t)
	cfg.HTTP.AllowedOrigins = []string{"https://claim.example.test"}
	application, err := NewApplication(cfg, slog.New(slog.DiscardHandler), Module{Name: "claimbounty", Routes: routes})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"title":"Study","purpose":"Audit","targetClaim":{"text":"Claim"},"permissions":{"executeSuppliedCode":false,"externalSearch":false},"privacy":{"containsParticipantLevelData":false,"containsDirectIdentifiers":false}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://claim.example.test")
	request.Header.Set("X-Csrf-Token", strings.Repeat("c", 32))
	request.Header.Set("Idempotency-Key", "create-order-0001")
	request.AddCookie(&http.Cookie{Name: "__Host-claimbounty-session", Value: strings.Repeat("s", 32)})
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("ETag") != "\"1\"" || !bytes.Contains(recorder.Body.Bytes(), []byte(`"publicReference":"CB-01J7Y8K2Q9ZX"`)) {
		t.Fatalf("response headers/body = %v %s", recorder.Header(), recorder.Body.String())
	}
	if intake.created.SubjectID != session.SubjectID {
		t.Fatalf("created subject = %s", intake.created.SubjectID)
	}
}

type processIdentity struct{ session domain.Session }

func (processIdentity) RequestVerification(context.Context, services.RequestVerificationCommand) error {
	return nil
}
func (identity processIdentity) ConfirmVerification(context.Context, services.ConfirmVerificationCommand) (services.EstablishedSession, error) {
	return services.EstablishedSession{Session: identity.session, CookieToken: strings.Repeat("s", 32), CSRFToken: strings.Repeat("c", 32)}, nil
}
func (identity processIdentity) Authenticate(context.Context, string, time.Time) (domain.Session, error) {
	return identity.session, nil
}
func (identity processIdentity) AuthorizeCSRF(context.Context, string, string, time.Time) (domain.Session, error) {
	return identity.session, nil
}
func (identity processIdentity) RefreshSession(context.Context, string, time.Time) (services.EstablishedSession, error) {
	return services.EstablishedSession{Session: identity.session, CSRFToken: strings.Repeat("c", 32)}, nil
}
func (processIdentity) Logout(context.Context, string, time.Time) error { return nil }
func (processIdentity) EnforceSessionRateLimit(context.Context, domain.Session, services.SessionRateLimit) error {
	return nil
}

type processIntake struct {
	session domain.Session
	created domain.Order
}

func (intake *processIntake) CreateOrder(_ context.Context, command services.CreateOrderCommand) (domain.Order, error) {
	order, err := domain.NewOrder(processID("11111111-1111-4111-8111-111111111111"), command.Session.SubjectID, command.Session.Email, "CB-01J7Y8K2Q9ZX", command.Title, command.Purpose, command.TargetClaim, command.Permissions, command.Privacy, time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC))
	intake.created = order
	return order, err
}
func (*processIntake) GetOwnedOrder(context.Context, domain.Session, domain.Identifier) (domain.Order, error) {
	return domain.Order{}, domain.ErrOrderNotFound
}
func (*processIntake) UploadFile(context.Context, services.UploadFileCommand) (services.UploadedFile, error) {
	return services.UploadedFile{}, nil
}
func (*processIntake) DeleteFile(context.Context, services.DeleteFileCommand) error { return nil }
func (*processIntake) SubmitOrder(context.Context, services.SubmitOrderCommand) (domain.Order, error) {
	return domain.Order{}, nil
}

type processAdmin struct{}

func (processAdmin) ListOrders(context.Context, domain.Session, services.OrderPageRequest) (services.OrderPage, error) {
	return services.OrderPage{}, nil
}
func (processAdmin) GetOrder(context.Context, domain.Session, domain.Identifier) (domain.Order, []domain.Export, error) {
	return domain.Order{}, nil, nil
}
func (processAdmin) UpdateIntake(context.Context, services.AdminIntakeCommand) (domain.Order, []domain.Export, error) {
	return domain.Order{}, nil, nil
}
func (processAdmin) CreateExport(context.Context, services.CreateExportCommand) (domain.Export, error) {
	return domain.Export{}, nil
}
func (processAdmin) GetExport(context.Context, domain.Session, domain.Identifier, domain.Identifier) (domain.Export, error) {
	return domain.Export{}, nil
}
func (processAdmin) OpenFile(context.Context, domain.Session, domain.Identifier, domain.Identifier) (services.ObjectReader, domain.OrderFile, error) {
	return io.NopCloser(strings.NewReader("")), domain.OrderFile{}, nil
}
func (processAdmin) OpenExport(context.Context, domain.Session, domain.Identifier) (services.ObjectReader, domain.Export, error) {
	return io.NopCloser(strings.NewReader("")), domain.Export{}, nil
}

func processSession(t *testing.T) domain.Session {
	t.Helper()
	session, err := domain.NewSession(processID("22222222-2222-4222-8222-222222222222"), processID("33333333-3333-4333-8333-333333333333"), "owner@example.test", domain.SubmitterAudience, "authorization-v1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return session
}
func processID(raw string) domain.Identifier { id, _ := domain.NewIdentifier(raw); return id }
