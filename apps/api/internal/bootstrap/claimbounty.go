package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	claimexport "github.com/mattwebhub/micro1-template/apps/api/internal/adapters/export"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/objectstore"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/postgres"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/scanner"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/system"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/validation"
	"github.com/mattwebhub/micro1-template/apps/api/internal/config"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi"
)

func openClaimStore(ctx context.Context, cfg config.Config) (*postgres.Store, error) {
	emails, err := system.NewEmailProtector(cfg.ClaimBounty.EmailEncryptionKey, cfg.ClaimBounty.EmailLookupKey)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: initialize email protection: %w", err)
	}
	startupContext, cancel := context.WithTimeout(ctx, cfg.Database.QueryTimeout)
	defer cancel()
	store, err := postgres.Open(startupContext, cfg.Database.URL, cfg.Database.MaxConnections, cfg.Database.QueryTimeout, emails)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: open ClaimBounty database: %w", err)
	}
	return store, nil
}

func buildClaimBountyModule(ctx context.Context, cfg config.Config, logger *slog.Logger) (Module, error) {
	store, err := openClaimStore(ctx, cfg)
	if err != nil {
		return Module{}, err
	}
	storageClient, err := objectstore.NewS3(ctx, cfg.ClaimBounty.S3.Endpoint, cfg.ClaimBounty.S3.Region, cfg.ClaimBounty.S3.Bucket, cfg.ClaimBounty.S3.AccessKey, cfg.ClaimBounty.S3.SecretKey, cfg.ClaimBounty.S3.Secure, cfg.ClaimBounty.S3.CreateBucket)
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: open private object storage: %w", err)
	}
	storage, err := objectstore.NewScope(storageClient, []string{"accepted/sha256/", "exports/"}, []string{"quarantine/"}, []string{"quarantine/"})
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: scope API object storage: %w", err)
	}
	policy, err := system.NewAllowlistPolicy(cfg.ClaimBounty.AdminEmails, cfg.ClaimBounty.AuthorizationVersion, cfg.ClaimBounty.AdminAllowlistVersion)
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct admin policy: %w", err)
	}
	values, clock := system.SecureValues{}, system.Clock{}
	mailer, err := system.NewSMTPMailer(system.SMTPMailerConfig{
		Address:       cfg.ClaimBounty.SMTP.Address,
		Username:      cfg.ClaimBounty.SMTP.Username,
		Password:      cfg.ClaimBounty.SMTP.Password,
		From:          cfg.ClaimBounty.SMTP.From,
		TLSMode:       cfg.ClaimBounty.SMTP.TLSMode,
		TLSServerName: cfg.ClaimBounty.SMTP.TLSServerName,
		TLSCAFile:     cfg.ClaimBounty.SMTP.TLSCAFile,
	})
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct verification mailer: %w", err)
	}
	identity, err := services.NewIdentityService(store, mailer, values, clock, policy, cfg.ClaimBounty.SessionPepper)
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct identity service: %w", err)
	}
	intake, err := services.NewIntakeService(store, storage, values, clock, services.RetentionContract{
		PolicyVersion:  cfg.ClaimBounty.RetentionPolicyVersion,
		SourceDuration: cfg.ClaimBounty.SourceRetentionMaxDuration,
		PIIDuration:    cfg.ClaimBounty.PIIRetentionMaxDuration,
	})
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct intake service: %w", err)
	}
	schemas, err := validation.New()
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: compile ClaimBounty schemas: %w", err)
	}
	administration, err := services.NewAdministrationService(store, storage, schemas, policy, values, clock, services.TrustedRoutineContract{Revision: cfg.ClaimBounty.TrustedRoutine.Revision, ValidatedAt: cfg.ClaimBounty.TrustedRoutine.ValidatedAt, EvidenceSHA256: cfg.ClaimBounty.TrustedRoutine.EvidenceSHA256})
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct administration service: %w", err)
	}
	jsonLimit := min(cfg.HTTP.MaxBodyBytes, int64(1<<20))
	routes, err := httpapi.NewClaimRoutes(identity, intake, administration, logger, jsonLimit, cfg.ClaimBounty.CanonicalOrigin, cfg.HTTP.TrustedProxyCIDRs...)
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct ClaimBounty routes: %w", err)
	}
	return Module{Name: "claimbounty", Routes: routes, Readiness: store.Check, Shutdown: func(context.Context) error { store.Close(); return nil }}, nil
}

func buildClaimBountyWorkers(ctx context.Context, cfg config.Config, logger *slog.Logger) (*services.Workers, func(), error) {
	store, err := openClaimStore(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	closeStore := func() { store.Close() }
	storageClient, err := objectstore.NewS3(ctx, cfg.ClaimBounty.S3.Endpoint, cfg.ClaimBounty.S3.Region, cfg.ClaimBounty.S3.Bucket, cfg.ClaimBounty.S3.AccessKey, cfg.ClaimBounty.S3.SecretKey, cfg.ClaimBounty.S3.Secure, cfg.ClaimBounty.S3.CreateBucket)
	if err != nil {
		closeStore()
		return nil, nil, fmt.Errorf("bootstrap: open private object storage: %w", err)
	}
	storage, err := objectstore.NewScope(storageClient, []string{"quarantine/", "accepted/sha256/", "exports/"}, []string{"accepted/sha256/", "exports/"}, []string{"quarantine/", "accepted/sha256/", "exports/"})
	if err != nil {
		closeStore()
		return nil, nil, fmt.Errorf("bootstrap: scope worker object storage: %w", err)
	}
	policy, err := system.NewAllowlistPolicy(cfg.ClaimBounty.AdminEmails, cfg.ClaimBounty.AuthorizationVersion, cfg.ClaimBounty.AdminAllowlistVersion)
	if err != nil {
		closeStore()
		return nil, nil, fmt.Errorf("bootstrap: construct admin policy: %w", err)
	}
	schemas, err := validation.New()
	if err != nil {
		closeStore()
		return nil, nil, fmt.Errorf("bootstrap: compile ClaimBounty schemas: %w", err)
	}
	builder, err := claimexport.NewBuilder(storage, schemas, policy)
	if err != nil {
		closeStore()
		return nil, nil, fmt.Errorf("bootstrap: construct export builder: %w", err)
	}
	inspector, err := scanner.NewClamAV(cfg.ClaimBounty.ClamAV.Address, cfg.ClaimBounty.ClamAV.Timeout)
	if err != nil {
		closeStore()
		return nil, nil, fmt.Errorf("bootstrap: construct file scanner: %w", err)
	}
	workers, err := services.NewWorkers(store, storage, inspector, builder, system.Clock{}, cfg.ClaimBounty.WorkerInterval, logger)
	if err != nil {
		closeStore()
		return nil, nil, fmt.Errorf("bootstrap: construct workers: %w", err)
	}
	return workers, closeStore, nil
}

func runRetentionCleanup(ctx context.Context, cfg config.Config) error {
	store, err := openClaimStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer store.Close()
	cleanup, err := services.NewRetentionCleanup(store, system.Clock{}, cfg.ClaimBounty.RetentionBatch, cfg.ClaimBounty.AbandonedAfter)
	if err != nil {
		return fmt.Errorf("bootstrap: construct retention cleanup: %w", err)
	}
	return cleanup.Run(ctx)
}
