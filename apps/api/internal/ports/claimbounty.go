package ports

import (
	"context"
	"io"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

type Challenge struct {
	ID                domain.Identifier
	SubjectID         domain.Identifier
	Email             string
	Audience          domain.Audience
	TokenHash         [32]byte
	ExpiresAt         time.Time
	AttemptsRemaining int
}

type SessionCredential struct {
	Session   domain.Session
	TokenHash [32]byte
	CSRFHash  [32]byte
}

type IdentityRepository interface {
	EnforceRateLimit(context.Context, string, [32]byte, time.Time, time.Duration, int) error
	CreateChallenge(context.Context, Challenge) error
	ExchangeChallenge(context.Context, string, domain.Audience, [32]byte, domain.Identifier, [32]byte, [32]byte, string, time.Time, time.Time) (SessionCredential, error)
	GetSession(context.Context, [32]byte, time.Time) (SessionCredential, error)
	RotateCSRF(context.Context, domain.Identifier, [32]byte, [32]byte, time.Time) error
	RevokeSession(context.Context, [32]byte, time.Time) error
}

type EmailProtector interface {
	EncryptEmail(context.Context, string) ([]byte, [32]byte, error)
	DecryptEmail(context.Context, []byte) (string, error)
	LookupEmail(string) [32]byte
}

type VerificationMailer interface {
	SendVerification(context.Context, string, domain.Audience, string, time.Time) error
}

type SecureValues interface {
	NewIdentifier(context.Context) (domain.Identifier, error)
	NewOpaqueToken(context.Context, int) (string, error)
	NewChallengeCode(context.Context) (string, error)
	NewObjectKey(context.Context, string) (string, error)
	NewPublicReference(context.Context) (string, error)
}

type OrderPageRequest struct {
	Status          string
	CreatedAfter    *time.Time
	CreatedBefore   *time.Time
	PublicReference string
	Limit           uint32
	Cursor          string
}

type OrderPage struct {
	Orders     []domain.Order
	NextCursor string
}

type ExportQueueRequest struct {
	Order          domain.Order
	Export         domain.Export
	ActorID        domain.Identifier
	IdempotencyKey string
	RequestHash    [32]byte
}

type IdempotentOrderWrite struct {
	Order           domain.Order
	ExpectedVersion uint64
	ActorID         domain.Identifier
	Operation       string
	IdempotencyKey  string
	RequestHash     [32]byte
}

type UploadedFileWrite struct {
	Write IdempotentOrderWrite
	File  domain.OrderFile
}

type IntakeRepository interface {
	CreateOrder(context.Context, IdempotentOrderWrite) (domain.Order, error)
	GetOwnedOrder(context.Context, domain.Identifier, domain.Identifier) (domain.Order, error)
	GetOrder(context.Context, domain.Identifier) (domain.Order, []domain.Export, error)
	ListOrders(context.Context, OrderPageRequest) (OrderPage, error)
	SaveOrder(context.Context, domain.Order, uint64) error
	SaveOrderIdempotent(context.Context, IdempotentOrderWrite) (domain.Order, error)
	GetIdempotentOrder(context.Context, domain.Identifier, string, string, [32]byte) (domain.Order, bool, error)
	SaveUploadedFile(context.Context, UploadedFileWrite) (domain.Order, domain.OrderFile, error)
	GetIdempotentFile(context.Context, domain.Identifier, string, [32]byte) (domain.Order, domain.OrderFile, bool, error)
	RemoveFile(context.Context, domain.Order, domain.OrderFile, uint64, domain.Identifier) error
	QueueExport(context.Context, ExportQueueRequest) (domain.Export, error)
	GetIdempotentExport(context.Context, domain.Identifier, string, [32]byte) (domain.Export, bool, error)
	GetExport(context.Context, domain.Identifier, domain.Identifier) (domain.Export, error)
	ClaimInspection(context.Context, time.Time) (domain.Order, domain.OrderFile, bool, error)
	FinishInspection(context.Context, domain.Order, domain.OrderFile, uint64, ExpiredObject) error
	ClaimExport(context.Context, time.Time) (domain.Order, domain.Export, bool, error)
	FinishExport(context.Context, domain.Order, domain.Export, uint64) error
	FailExport(context.Context, domain.Identifier, domain.Identifier, string, time.Time) error
	ClaimDeletion(context.Context, time.Time) (int64, ExpiredObject, bool, error)
	FinishDeletion(context.Context, int64, time.Time) error
	CleanupExpiredIdentityAndAbandoned(context.Context, time.Time, time.Time) error
	AnonymizeExpired(context.Context, time.Time, int) ([]ExpiredObject, error)
}

type ExpiredObject struct {
	Key        string
	Generation string
}

type ObjectMetadata struct {
	SizeBytes  int64
	ETag       string
	MediaType  string
	SHA256     string
	Generation string
	ModifiedAt time.Time
}

type ObjectReader interface {
	io.Reader
	io.Closer
}

type PrivateObjectStore interface {
	Open(context.Context, string, string) (ObjectReader, ObjectMetadata, error)
	PutWriteOnce(context.Context, string, io.Reader, int64, string, string) (ObjectMetadata, error)
	DeleteVersion(context.Context, string, string) error
}

type AdminPolicy interface {
	Authorize(context.Context, string, string) error
	Version() string
	AllowlistVersion() string
}

type IntakeValidator interface {
	ValidateAuditRequest([]byte) error
	ValidateScientificPolicy([]byte) error
	ValidateExecutionPolicy([]byte) error
	ValidateCaseManifest([]byte) error
}

type FileInspector interface {
	Inspect(context.Context, io.Reader, int64, string) (detectedMediaType string, clean bool, rejectionCode string, err error)
}

type ExportBuilder interface {
	Build(context.Context, domain.Order, domain.Export) (ObjectMetadata, error)
}
