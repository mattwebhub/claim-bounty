package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi/middleware"
	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi/response"
)

const sessionCookieName = "__Host-claimbounty-session"

type IdentityActions interface {
	RequestVerification(context.Context, services.RequestVerificationCommand) error
	ConfirmVerification(context.Context, services.ConfirmVerificationCommand) (services.EstablishedSession, error)
	Authenticate(context.Context, string, time.Time) (domain.Session, error)
	AuthorizeCSRF(context.Context, string, string, time.Time) (domain.Session, error)
	RefreshSession(context.Context, string, time.Time) (services.EstablishedSession, error)
	EnforceSessionRateLimit(context.Context, domain.Session, services.SessionRateLimit) error
	Logout(context.Context, string, time.Time) error
}

type ClaimRoutes struct {
	identity        IdentityActions
	intake          IntakeActions
	administration  AdministrationActions
	logger          *slog.Logger
	maxJSONBytes    int64
	canonicalOrigin string
	trustedProxies  []*net.IPNet
}

func NewClaimRoutes(identity IdentityActions, intake IntakeActions, administration AdministrationActions, logger *slog.Logger, maxJSONBytes int64, canonicalOrigin string, trustedProxyCIDRs ...string) (*ClaimRoutes, error) {
	if identity == nil || intake == nil || administration == nil || strings.TrimSpace(canonicalOrigin) == "" {
		return nil, errors.New("httpapi: claim route dependencies are required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if maxJSONBytes <= 0 {
		maxJSONBytes = 1 << 20
	}
	trustedProxies := make([]*net.IPNet, 0, len(trustedProxyCIDRs))
	for _, raw := range trustedProxyCIDRs {
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return nil, errors.New("httpapi: trusted proxy CIDR is invalid")
		}
		trustedProxies = append(trustedProxies, network)
	}
	return &ClaimRoutes{identity: identity, intake: intake, administration: administration, logger: logger, maxJSONBytes: maxJSONBytes, canonicalOrigin: canonicalOrigin, trustedProxies: trustedProxies}, nil
}

func (routes *ClaimRoutes) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/email-challenges", noStore(routes.requestChallenge))
	mux.HandleFunc("POST /api/v1/email-challenges/verify", noStore(routes.verifyChallenge))
	mux.HandleFunc("GET /api/v1/session", noStore(routes.getSession))
	mux.HandleFunc("DELETE /api/v1/session", noStore(routes.logout))
	mux.HandleFunc("POST /api/v1/orders", noStore(routes.createOrder))
	mux.HandleFunc("GET /api/v1/orders/{orderId}", noStore(routes.getOrder))
	mux.HandleFunc("POST /api/v1/orders/{orderId}/files", noStore(routes.uploadFile))
	mux.HandleFunc("DELETE /api/v1/orders/{orderId}/files/{fileId}", noStore(routes.deleteFile))
	mux.HandleFunc("POST /api/v1/orders/{orderId}/submit", noStore(routes.submitOrder))
	mux.HandleFunc("GET /api/v1/admin/orders", noStore(routes.listAdminOrders))
	mux.HandleFunc("GET /api/v1/admin/orders/{orderId}", noStore(routes.getAdminOrder))
	mux.HandleFunc("PATCH /api/v1/admin/orders/{orderId}/intake", noStore(routes.updateAdminIntake))
	mux.HandleFunc("GET /api/v1/admin/orders/{orderId}/files/{fileId}/content", noStore(routes.downloadAdminFile))
	mux.HandleFunc("POST /api/v1/admin/orders/{orderId}/exports", noStore(routes.createExport))
	mux.HandleFunc("GET /api/v1/admin/orders/{orderId}/exports/{exportId}", noStore(routes.getExport))
	mux.HandleFunc("GET /api/v1/admin/exports/{exportId}/download", noStore(routes.downloadExport))
}

func noStore(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		privateNoStore(w)
		next(w, r)
	}
}

func (routes *ClaimRoutes) AllowedOrigins() []string {
	return []string{routes.canonicalOrigin}
}

type challengeRequest struct {
	Email    *string `json:"email"`
	Audience *string `json:"audience"`
}

func (routes *ClaimRoutes) requestChallenge(w http.ResponseWriter, r *http.Request) {
	privateNoStore(w)
	if !routes.validOrigin(r) {
		routes.writeError(w, r, domain.ErrForbidden)
		return
	}
	var body challengeRequest
	if err := response.DecodeJSON(w, r, &body, routes.maxJSONBytes); err != nil {
		routes.writeError(w, r, err)
		return
	}
	if body.Email == nil || body.Audience == nil {
		routes.writeError(w, r, domain.NewValidationError(domain.FieldIssue{Field: "email", Code: "required", Message: "email and audience are required"}))
		return
	}
	if err := routes.identity.RequestVerification(r.Context(), services.RequestVerificationCommand{Email: *body.Email, Audience: *body.Audience, IPPrefix: routes.clientPrefix(r), Requested: time.Now().UTC()}); err != nil {
		if !errors.Is(err, domain.ErrRateLimited) {
			routes.logger.WarnContext(r.Context(), "verification request was not delivered", "error", err)
		}
	}
	_ = response.WriteData(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

type verifyRequest struct {
	Email    *string `json:"email"`
	Audience *string `json:"audience"`
	Code     *string `json:"code"`
}

func (routes *ClaimRoutes) verifyChallenge(w http.ResponseWriter, r *http.Request) {
	privateNoStore(w)
	if !routes.validOrigin(r) {
		routes.writeError(w, r, domain.ErrForbidden)
		return
	}
	var body verifyRequest
	if err := response.DecodeJSON(w, r, &body, routes.maxJSONBytes); err != nil {
		routes.writeError(w, r, err)
		return
	}
	if body.Email == nil || body.Audience == nil || body.Code == nil {
		routes.writeError(w, r, domain.ErrInvalidChallenge)
		return
	}
	established, err := routes.identity.ConfirmVerification(r.Context(), services.ConfirmVerificationCommand{Email: *body.Email, Audience: *body.Audience, Code: *body.Code, IPPrefix: routes.clientPrefix(r), Confirmed: time.Now().UTC()})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: established.CookieToken, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, Expires: established.Session.ExpiresAt})
	_ = response.WriteData(w, http.StatusOK, sessionDTO(established.Session, established.CSRFToken))
}

func (routes *ClaimRoutes) getSession(w http.ResponseWriter, r *http.Request) {
	privateNoStore(w)
	token, err := sessionToken(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	now := time.Now().UTC()
	session, err := routes.identity.Authenticate(r.Context(), token, now)
	if err == nil {
		err = routes.identity.EnforceSessionRateLimit(r.Context(), session, services.SessionRateLimit{Scope: "session_refresh", IPPrefix: routes.clientPrefix(r), Now: now})
	}
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	established, err := routes.identity.RefreshSession(r.Context(), token, now)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	_ = response.WriteData(w, http.StatusOK, sessionDTO(established.Session, established.CSRFToken))
}

func (routes *ClaimRoutes) logout(w http.ResponseWriter, r *http.Request) {
	privateNoStore(w)
	_, err := routes.unsafeSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	token, err := sessionToken(r)
	if err == nil {
		err = routes.identity.Logout(r.Context(), token, time.Now().UTC())
	}
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0).UTC()})
	w.Header().Set("Clear-Site-Data", `"cookies"`)
	w.WriteHeader(http.StatusNoContent)
}

func privateNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
}

func sessionDTO(session domain.Session, csrf string) any {
	return struct {
		Audience                   string `json:"audience"`
		CSRFToken                  string `json:"csrfToken"`
		AuthorizationPolicyVersion string `json:"authorizationPolicyVersion"`
		ExpiresAt                  string `json:"expiresAt"`
	}{string(session.Audience), csrf, session.AuthorizationPolicyVersion, session.ExpiresAt.UTC().Format(time.RFC3339Nano)}
}
func sessionToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", domain.ErrUnauthorized
	}
	return cookie.Value, nil
}
func (routes *ClaimRoutes) readSession(r *http.Request) (domain.Session, error) {
	token, err := sessionToken(r)
	if err != nil {
		return domain.Session{}, err
	}
	now := time.Now().UTC()
	session, err := routes.identity.Authenticate(r.Context(), token, now)
	if err != nil {
		return domain.Session{}, err
	}
	if err := routes.identity.EnforceSessionRateLimit(r.Context(), session, services.SessionRateLimit{Scope: requestRateScope(r), IPPrefix: routes.clientPrefix(r), Now: now}); err != nil {
		return domain.Session{}, err
	}
	return session, nil
}
func (routes *ClaimRoutes) unsafeSession(r *http.Request) (domain.Session, error) {
	if !routes.validOrigin(r) {
		return domain.Session{}, domain.ErrForbidden
	}
	token, err := sessionToken(r)
	if err != nil {
		return domain.Session{}, err
	}
	now := time.Now().UTC()
	session, err := routes.identity.AuthorizeCSRF(r.Context(), token, r.Header.Get("X-Csrf-Token"), now)
	if err != nil {
		return domain.Session{}, err
	}
	if err := routes.identity.EnforceSessionRateLimit(r.Context(), session, services.SessionRateLimit{Scope: requestRateScope(r), IPPrefix: routes.clientPrefix(r), Now: now}); err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func requestRateScope(r *http.Request) string {
	pattern := r.Pattern
	if pattern == "" {
		pattern = r.URL.Path
	}
	if strings.HasPrefix(pattern, "GET /api/v1/admin/") || strings.HasPrefix(pattern, "/api/v1/admin/") && r.Method == http.MethodGet {
		return "admin_read"
	}
	if strings.Contains(pattern, "/api/v1/admin/") {
		return "admin_write"
	}
	if strings.Contains(pattern, "/files") && r.Method == http.MethodPost {
		return "order_upload"
	}
	if strings.HasSuffix(pattern, "/submit") {
		return "order_submit"
	}
	if r.Method == http.MethodGet {
		return "order_read"
	}
	return "order_write"
}
func (routes *ClaimRoutes) validOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == routes.canonicalOrigin
}
func (routes *ClaimRoutes) clientPrefix(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "unknown"
	}
	for _, proxy := range routes.trustedProxies {
		if proxy.Contains(ip) {
			forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
			if candidate := net.ParseIP(forwarded); candidate != nil {
				ip = candidate
			}
			break
		}
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.Mask(net.CIDRMask(24, 32)).String() + "/24"
	}
	return ip.Mask(net.CIDRMask(56, 128)).String() + "/56"
}
func requestHash(value any) [32]byte {
	encoded, _ := json.Marshal(value)
	return sha256.Sum256(encoded)
}
func (routes *ClaimRoutes) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message, details := mapError(err)
	requestID := middleware.RequestIDFromContext(r.Context())
	if status >= 500 {
		routes.logger.ErrorContext(r.Context(), "HTTP request failed", "error", err, "request_id", requestID)
	}
	_ = response.WriteError(w, status, code, message, requestID, details)
}
