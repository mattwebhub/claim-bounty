package services

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

const (
	challengeLifetime = 10 * time.Minute
	sessionLifetime   = 12 * time.Hour
)

type RequestVerificationCommand struct {
	Email     string
	Audience  string
	IPPrefix  string
	Requested time.Time
}

type ConfirmVerificationCommand struct {
	Email     string
	Audience  string
	Code      string
	IPPrefix  string
	Confirmed time.Time
}

type EstablishedSession struct {
	Session     domain.Session
	CookieToken string
	CSRFToken   string
}

type SessionRateLimit struct {
	Scope    string
	IPPrefix string
	Now      time.Time
}

type IdentityService struct {
	repository  ports.IdentityRepository
	mailer      ports.VerificationMailer
	values      ports.SecureValues
	clock       ports.Clock
	adminPolicy ports.AdminPolicy
	pepper      string
}

func NewIdentityService(repository ports.IdentityRepository, mailer ports.VerificationMailer, values ports.SecureValues, clock ports.Clock, adminPolicy ports.AdminPolicy, pepper string) (*IdentityService, error) {
	if repository == nil || mailer == nil || values == nil || clock == nil || adminPolicy == nil || len(pepper) < 32 {
		return nil, ErrInvalidDependencies
	}
	return &IdentityService{repository: repository, mailer: mailer, values: values, clock: clock, adminPolicy: adminPolicy, pepper: pepper}, nil
}

func (service *IdentityService) RequestVerification(ctx context.Context, command RequestVerificationCommand) error {
	email, err := normalizeEmail(command.Email)
	if err != nil {
		return domain.NewValidationError(domain.FieldIssue{Field: "email", Code: "invalid_format", Message: "must be a valid email address"})
	}
	audience, err := domain.NewAudience(command.Audience)
	if err != nil {
		return err
	}
	now := command.Requested.UTC()
	if now.IsZero() {
		now = service.clock.Now()
	}
	emailHash := sha256.Sum256([]byte(email))
	ipHash := sha256.Sum256([]byte(command.IPPrefix))
	if err := service.repository.EnforceRateLimit(ctx, "verification_email", emailHash, now, 15*time.Minute, 5); err != nil {
		return err
	}
	if err := service.repository.EnforceRateLimit(ctx, "verification_ip", ipHash, now, 15*time.Minute, 20); err != nil {
		return err
	}
	if audience == domain.AdminAudience {
		if err := service.adminPolicy.Authorize(ctx, email, ""); err != nil {
			return concealAdminChallenge(err)
		}
	}
	challengeID, err := service.values.NewIdentifier(ctx)
	if err != nil {
		return err
	}
	subjectID, err := service.values.NewIdentifier(ctx)
	if err != nil {
		return err
	}
	code, err := service.values.NewChallengeCode(ctx)
	if err != nil {
		return err
	}
	expires := now.Add(challengeLifetime)
	if err := service.repository.CreateChallenge(ctx, ports.Challenge{
		ID: challengeID, SubjectID: subjectID, Email: email, Audience: audience,
		TokenHash: service.challengeHash(email, audience, code), ExpiresAt: expires, AttemptsRemaining: 5,
	}); err != nil {
		return err
	}
	return service.mailer.SendVerification(ctx, email, audience, code, expires)
}

func concealAdminChallenge(err error) error {
	if errors.Is(err, domain.ErrForbidden) {
		return nil
	}
	return err
}

func (service *IdentityService) ConfirmVerification(ctx context.Context, command ConfirmVerificationCommand) (EstablishedSession, error) {
	email, err := normalizeEmail(command.Email)
	if err != nil || len(command.Code) != 6 {
		return EstablishedSession{}, domain.ErrInvalidChallenge
	}
	audience, err := domain.NewAudience(command.Audience)
	if err != nil {
		return EstablishedSession{}, domain.ErrInvalidChallenge
	}
	if audience == domain.AdminAudience {
		if err := service.adminPolicy.Authorize(ctx, email, ""); err != nil {
			return EstablishedSession{}, domain.ErrInvalidChallenge
		}
	}
	now := command.Confirmed.UTC()
	if now.IsZero() {
		now = service.clock.Now()
	}
	ipHash := sha256.Sum256([]byte(command.IPPrefix))
	if err := service.repository.EnforceRateLimit(ctx, "verification_confirm_ip", ipHash, now, 15*time.Minute, 30); err != nil {
		return EstablishedSession{}, err
	}
	sessionID, err := service.values.NewIdentifier(ctx)
	if err != nil {
		return EstablishedSession{}, err
	}
	cookieToken, err := service.values.NewOpaqueToken(ctx, 32)
	if err != nil {
		return EstablishedSession{}, err
	}
	csrfToken, err := service.values.NewOpaqueToken(ctx, 32)
	if err != nil {
		return EstablishedSession{}, err
	}
	credential, err := service.repository.ExchangeChallenge(ctx,
		email, audience, service.challengeHash(email, audience, command.Code), sessionID,
		sha256.Sum256([]byte(cookieToken)), sha256.Sum256([]byte(csrfToken)),
		service.adminPolicy.Version(),
		now, now.Add(sessionLifetime),
	)
	if err != nil {
		return EstablishedSession{}, err
	}
	return EstablishedSession{Session: credential.Session, CookieToken: cookieToken, CSRFToken: csrfToken}, nil
}

func (service *IdentityService) RefreshSession(ctx context.Context, cookieToken string, now time.Time) (EstablishedSession, error) {
	if len(cookieToken) < 32 || len(cookieToken) > 512 {
		return EstablishedSession{}, domain.ErrUnauthorized
	}
	if now.IsZero() {
		now = service.clock.Now()
	}
	credential, err := service.repository.GetSession(ctx, sha256.Sum256([]byte(cookieToken)), now.UTC())
	if err != nil {
		return EstablishedSession{}, err
	}
	csrfToken, err := service.values.NewOpaqueToken(ctx, 32)
	if err != nil {
		return EstablishedSession{}, err
	}
	if err := service.repository.RotateCSRF(ctx, credential.Session.ID, credential.CSRFHash, sha256.Sum256([]byte(csrfToken)), now.UTC()); err != nil {
		return EstablishedSession{}, err
	}
	return EstablishedSession{Session: credential.Session, CSRFToken: csrfToken}, nil
}

func (service *IdentityService) Logout(ctx context.Context, cookieToken string, now time.Time) error {
	if len(cookieToken) < 32 || len(cookieToken) > 512 {
		return domain.ErrUnauthorized
	}
	if now.IsZero() {
		now = service.clock.Now()
	}
	return service.repository.RevokeSession(ctx, sha256.Sum256([]byte(cookieToken)), now.UTC())
}

func (service *IdentityService) EnforceSessionRateLimit(ctx context.Context, session domain.Session, request SessionRateLimit) error {
	window, limit, ok := authenticatedRatePolicy(request.Scope)
	if !ok || session.ID == "" || session.Email == "" {
		return domain.ErrForbidden
	}
	now := request.Now.UTC()
	if now.IsZero() {
		now = service.clock.Now()
	}
	keys := []struct {
		suffix string
		value  string
	}{
		{"session", session.ID.String()},
		{"ip", request.IPPrefix},
		{"email", strings.ToLower(session.Email)},
	}
	for _, key := range keys {
		digest := sha256.Sum256([]byte(key.value))
		if err := service.repository.EnforceRateLimit(ctx, request.Scope+"_"+key.suffix, digest, now, window, limit); err != nil {
			return err
		}
	}
	return nil
}

func authenticatedRatePolicy(scope string) (time.Duration, int, bool) {
	switch scope {
	case "session_refresh":
		return 5 * time.Minute, 30, true
	case "order_upload":
		return 15 * time.Minute, 20, true
	case "order_submit":
		return 15 * time.Minute, 10, true
	case "order_read":
		return 5 * time.Minute, 120, true
	case "admin_read":
		return 5 * time.Minute, 120, true
	case "admin_write":
		return 15 * time.Minute, 30, true
	case "order_write":
		return 15 * time.Minute, 60, true
	default:
		return 0, 0, false
	}
}

func (service *IdentityService) challengeHash(email string, audience domain.Audience, code string) [32]byte {
	return sha256.Sum256([]byte(service.pepper + "\x00" + email + "\x00" + string(audience) + "\x00" + code))
}

func (service *IdentityService) Authenticate(ctx context.Context, cookieToken string, now time.Time) (domain.Session, error) {
	if len(cookieToken) < 32 || len(cookieToken) > 512 {
		return domain.Session{}, domain.ErrUnauthorized
	}
	if now.IsZero() {
		now = service.clock.Now()
	}
	credential, err := service.repository.GetSession(ctx, sha256.Sum256([]byte(cookieToken)), now.UTC())
	if err != nil {
		return domain.Session{}, err
	}
	return credential.Session, nil
}

func (service *IdentityService) AuthorizeCSRF(ctx context.Context, cookieToken, csrfToken string, now time.Time) (domain.Session, error) {
	if len(csrfToken) < 32 || len(csrfToken) > 256 {
		return domain.Session{}, domain.ErrForbidden
	}
	if now.IsZero() {
		now = service.clock.Now()
	}
	credential, err := service.repository.GetSession(ctx, sha256.Sum256([]byte(cookieToken)), now.UTC())
	if err != nil {
		return domain.Session{}, err
	}
	provided := sha256.Sum256([]byte(csrfToken))
	if subtle.ConstantTimeCompare(provided[:], credential.CSRFHash[:]) != 1 {
		return domain.Session{}, domain.ErrForbidden
	}
	return credential.Session, nil
}

func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > 254 || strings.ContainsAny(value, "\r\n") {
		return "", errors.New("invalid email")
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return "", errors.New("invalid email")
	}
	parts := strings.Split(address.Address, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("invalid email")
	}
	return parts[0] + "@" + strings.ToLower(parts[1]), nil
}
