//go:build integration

package postgres_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/postgres"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

func TestClaimBountyChallengeExpiryLockoutReplayAndSessionExpiry(t *testing.T) {
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
	protector := claimTestEmailProtector(t)
	store, err := postgres.Open(ctx, databaseURL, 4, 5*time.Second, protector)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	subject := claimID(t, "11111111-1111-4111-8111-111111111111")
	email := "owner@example.test"
	createChallenge := func(id string, token [32]byte, expires time.Time) {
		t.Helper()
		if err := store.CreateChallenge(ctx, ports.Challenge{
			ID: claimID(t, id), SubjectID: subject, Email: email,
			Audience: domain.SubmitterAudience, TokenHash: token,
			ExpiresAt: expires, AttemptsRemaining: 5,
		}); err != nil {
			t.Fatal(err)
		}
	}
	exchange := func(sessionID string, token, sessionHash [32]byte, at time.Time) (ports.SessionCredential, error) {
		t.Helper()
		return store.ExchangeChallenge(
			ctx, email, domain.SubmitterAudience, token, claimID(t, sessionID),
			sessionHash, sha256.Sum256([]byte("csrf-"+sessionID)), "authorization-v1",
			at, at.Add(time.Hour),
		)
	}

	expired := sha256.Sum256([]byte("expired"))
	createChallenge("22222222-2222-4222-8222-222222222221", expired, now.Add(-time.Second))
	if _, err := exchange("33333333-3333-4333-8333-333333333331", expired, sha256.Sum256([]byte("expired-session")), now); !errors.Is(err, domain.ErrInvalidChallenge) {
		t.Fatalf("expired challenge error = %v, want invalid challenge", err)
	}

	locked := sha256.Sum256([]byte("locked"))
	wrong := sha256.Sum256([]byte("wrong"))
	createChallenge("22222222-2222-4222-8222-222222222222", locked, now.Add(10*time.Minute))
	for attempt := 0; attempt < 5; attempt++ {
		if _, err := exchange("33333333-3333-4333-8333-333333333332", wrong, sha256.Sum256([]byte("locked-session")), now); !errors.Is(err, domain.ErrInvalidChallenge) {
			t.Fatalf("wrong-code attempt %d error = %v, want invalid challenge", attempt+1, err)
		}
	}
	if _, err := exchange("33333333-3333-4333-8333-333333333332", locked, sha256.Sum256([]byte("locked-session")), now); !errors.Is(err, domain.ErrInvalidChallenge) {
		t.Fatalf("locked challenge error = %v, want invalid challenge", err)
	}

	valid := sha256.Sum256([]byte("valid"))
	sessionHash := sha256.Sum256([]byte("valid-session"))
	csrfHash := sha256.Sum256([]byte("valid-csrf"))
	createChallenge("22222222-2222-4222-8222-222222222223", valid, now.Add(11*time.Minute))
	if _, err := store.ExchangeChallenge(
		ctx, email, domain.SubmitterAudience, valid,
		claimID(t, "33333333-3333-4333-8333-333333333333"), sessionHash, csrfHash,
		"authorization-v1", now, now.Add(time.Hour),
	); err != nil {
		t.Fatalf("valid challenge error = %v", err)
	}
	if _, err := exchange("33333333-3333-4333-8333-333333333334", valid, sha256.Sum256([]byte("replayed-session")), now); !errors.Is(err, domain.ErrInvalidChallenge) {
		t.Fatalf("replayed challenge error = %v, want invalid challenge", err)
	}
	if _, err := store.GetSession(ctx, sessionHash, now.Add(time.Hour)); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expired session error = %v, want unauthorized", err)
	}

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var challengeCiphertext, challengeLookup, storedChallengeHash []byte
	if err := database.QueryRowContext(ctx, `SELECT email_ciphertext,email_lookup_hash,token_hash FROM email_challenges WHERE id='22222222-2222-4222-8222-222222222223'`).Scan(&challengeCiphertext, &challengeLookup, &storedChallengeHash); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(challengeCiphertext, []byte(email)) || len(challengeLookup) != sha256.Size || !bytes.Equal(storedChallengeHash, valid[:]) {
		t.Fatalf("challenge secrets were not encrypted and hash-bound")
	}
	var sessionCiphertext, sessionLookup, storedSessionHash, storedCSRFHash []byte
	if err := database.QueryRowContext(ctx, `SELECT email_ciphertext,email_lookup_hash,token_hash,csrf_hash FROM claimbounty_sessions WHERE id='33333333-3333-4333-8333-333333333333'`).Scan(&sessionCiphertext, &sessionLookup, &storedSessionHash, &storedCSRFHash); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sessionCiphertext, []byte(email)) || len(sessionLookup) != sha256.Size || !bytes.Equal(storedSessionHash, sessionHash[:]) || !bytes.Equal(storedCSRFHash, csrfHash[:]) {
		t.Fatalf("session secrets were not encrypted and hash-bound")
	}
	rateKey := sha256.Sum256([]byte("192.0.2.0/24"))
	if err := store.EnforceRateLimit(ctx, "challenge_ip", rateKey, now, 15*time.Minute, 5); err != nil {
		t.Fatal(err)
	}
	var storedRateKey []byte
	if err := database.QueryRowContext(ctx, `SELECT key_hash FROM claimbounty_rate_limits WHERE scope='challenge_ip'`).Scan(&storedRateKey); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(storedRateKey, rateKey[:]) || bytes.Contains(storedRateKey, []byte("192.0.2.0/24")) {
		t.Fatalf("rate-limit identity was not stored as the expected hash")
	}
}

func TestClaimBountyRetentionErasesPIIAndQueuesEveryObjectVersion(t *testing.T) {
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
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	protector := claimTestEmailProtector(t)
	store, err := postgres.Open(ctx, databaseURL, 4, 5*time.Second, protector)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	subject := "11111111-1111-4111-8111-111111111111"
	orderID := "22222222-2222-4222-8222-222222222222"
	email := "retained-owner@example.test"
	emailCiphertext, emailLookup, err := protector.EncryptEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	challengeHash := sha256.Sum256([]byte("challenge"))
	sessionHash := sha256.Sum256([]byte("session"))
	csrfHash := sha256.Sum256([]byte("csrf"))
	exec := func(statement string, arguments ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, statement, arguments...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO email_challenges(id,subject_id,email_ciphertext,email_lookup_hash,audience,token_hash,expires_at,attempts_remaining)
VALUES('33333333-3333-4333-8333-333333333333',$1,$2,$3,'submitter',$4,$5,5)`,
		subject, emailCiphertext, emailLookup[:], challengeHash[:], now.Add(time.Hour))
	exec(`INSERT INTO claimbounty_sessions(id,subject_id,email_ciphertext,email_lookup_hash,audience,authorization_policy_version,token_hash,csrf_hash,expires_at)
VALUES('44444444-4444-4444-8444-444444444444',$1,$2,$3,'submitter','authorization-v1',$4,$5,$6)`,
		subject, emailCiphertext, emailLookup[:], sessionHash[:], csrfHash[:], now.Add(time.Hour))
	exec(`INSERT INTO claimbounty_orders(id,subject_id,submitter_email_ciphertext,submitter_email_lookup_hash,public_reference,status,version,title,purpose,target_claim_text,target_claim_location,execute_supplied_code,external_search,contains_participant_data,contains_direct_identifiers,terms_version,created_at,updated_at,submitted_at,retention_policy_version,retention_disposition,source_retention_expires_at,retention_expires_at)
VALUES($1,$2,$3,$4,'CB-RETENTION001','exported',7,'Study','Audit','Claim','Table 2',false,false,false,false,'terms-v1',$5,$5,$5,'retention-v1','hard_delete',$5,$6)`,
		orderID, subject, emailCiphertext, emailLookup[:], now.Add(-time.Hour), now.Add(24*time.Hour))
	exec(`INSERT INTO claimbounty_files(id,order_id,role,original_display_name,size_bytes,sha256,declared_media_type,detected_media_type,status,storage_key,storage_etag,object_generation,scanned_at,created_at,updated_at)
VALUES('55555555-5555-4555-8555-555555555555',$1,'primary_paper','paper.pdf',4,$2,'application/pdf','application/pdf','clean','quarantine/source','etag-source','source-version',$3,$3,$3)`,
		orderID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", now.Add(-time.Hour))
	exec(`INSERT INTO claimbounty_exports(id,order_id,status,routine_id,routine_revision,routine_validated_at,routine_evidence_sha256,retention_policy_version,preserve_run_outputs,sha256,size_bytes,storage_key,object_generation,created_at,completed_at)
VALUES('66666666-6666-4666-8666-666666666666',$1,'ready','claim-bounty-operations/run-claimbounty-scientific-audit',$2,$3,$4,'retention-v1',false,$5,8,'exports/archive','export-version',$3,$3)`,
		orderID, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		now.Add(-time.Hour), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")

	if _, err := store.AnonymizeExpired(ctx, now, 10); err != nil {
		t.Fatalf("AnonymizeExpired() error = %v (cause: %v)", err, errors.Unwrap(err))
	}
	for name, query := range map[string]string{
		"orders":     `SELECT count(*) FROM claimbounty_orders WHERE id='22222222-2222-4222-8222-222222222222'`,
		"sessions":   `SELECT count(*) FROM claimbounty_sessions`,
		"challenges": `SELECT count(*) FROM email_challenges`,
	} {
		var count int
		if err := database.QueryRowContext(ctx, query).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s remaining after source deadline = %d (error %v), want 1", name, count, err)
		}
	}
	var sourceDeletedAt *time.Time
	if err := database.QueryRowContext(ctx, `SELECT source_deleted_at FROM claimbounty_orders WHERE id=$1`, orderID).Scan(&sourceDeletedAt); err != nil || sourceDeletedAt != nil {
		t.Fatalf("source deletion marker = %v (error %v), want pending until object deletion succeeds", sourceDeletedAt, err)
	}
	var tombstones int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM claimbounty_tombstones WHERE order_id=$1`, orderID).Scan(&tombstones); err != nil || tombstones != 0 {
		t.Fatalf("tombstones before PII deadline = %d (error %v), want 0", tombstones, err)
	}
	rows, err := database.QueryContext(ctx, `SELECT storage_key,object_generation FROM claimbounty_outbox WHERE kind='delete_object' ORDER BY storage_key`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var objects []ports.ExpiredObject
	for rows.Next() {
		var object ports.ExpiredObject
		if err := rows.Scan(&object.Key, &object.Generation); err != nil {
			t.Fatal(err)
		}
		objects = append(objects, object)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 || objects[0] != (ports.ExpiredObject{Key: "exports/archive", Generation: "export-version"}) || objects[1] != (ports.ExpiredObject{Key: "quarantine/source", Generation: "source-version"}) {
		t.Fatalf("queued object cleanup = %#v", objects)
	}
	rows.Close()

	failedID, _, ok, err := store.ClaimDeletion(ctx, now)
	if err != nil || !ok {
		t.Fatalf("ClaimDeletion() before simulated storage failure = %d/%v/%v", failedID, ok, err)
	}
	if _, err := store.AnonymizeExpired(ctx, now.Add(time.Minute), 10); err != nil {
		t.Fatal(err)
	}
	var deletionJobs int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM claimbounty_outbox WHERE kind='delete_object' AND retention_order_id=$1`, orderID).Scan(&deletionJobs); err != nil || deletionJobs != 2 {
		t.Fatalf("retention deletion jobs after failure = %d (error %v), want 2 without duplicates", deletionJobs, err)
	}
	if err := database.QueryRowContext(ctx, `SELECT source_deleted_at FROM claimbounty_orders WHERE id=$1`, orderID).Scan(&sourceDeletedAt); err != nil || sourceDeletedAt != nil {
		t.Fatalf("source deletion marker after failed delete = %v (error %v), want pending", sourceDeletedAt, err)
	}
	recoveredAt := now.Add(6 * time.Minute)
	for completed := 0; completed < 2; completed++ {
		id, _, ok, err := store.ClaimDeletion(ctx, recoveredAt)
		if err != nil || !ok {
			t.Fatalf("ClaimDeletion() recovery %d = %d/%v/%v", completed, id, ok, err)
		}
		if err := store.FinishDeletion(ctx, id, recoveredAt); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.QueryRowContext(ctx, `SELECT source_deleted_at FROM claimbounty_orders WHERE id=$1`, orderID).Scan(&sourceDeletedAt); err != nil || sourceDeletedAt == nil || !sourceDeletedAt.Equal(recoveredAt) {
		t.Fatalf("source deletion marker after recovery = %v (error %v), want %s", sourceDeletedAt, err, recoveredAt)
	}

	if _, err := store.AnonymizeExpired(ctx, now.Add(24*time.Hour), 10); err != nil {
		t.Fatalf("AnonymizeExpired() at PII deadline error = %v (cause: %v)", err, errors.Unwrap(err))
	}
	for name, query := range map[string]string{
		"orders":     `SELECT count(*) FROM claimbounty_orders WHERE id='22222222-2222-4222-8222-222222222222'`,
		"sessions":   `SELECT count(*) FROM claimbounty_sessions`,
		"challenges": `SELECT count(*) FROM email_challenges`,
	} {
		var count int
		if err := database.QueryRowContext(ctx, query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s remaining after PII deadline = %d (error %v), want 0", name, count, err)
		}
	}
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM claimbounty_tombstones WHERE order_id=$1 AND final_status='exported'`, orderID).Scan(&tombstones); err != nil || tombstones != 1 {
		t.Fatalf("tombstones after PII deadline = %d (error %v), want 1", tombstones, err)
	}
}

func TestClaimBountyAbandonedDraftUsesSevenDayCutoffAndExactVersionCleanup(t *testing.T) {
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
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	protector := claimTestEmailProtector(t)
	store, err := postgres.Open(ctx, databaseURL, 4, 5*time.Second, protector)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	emailCiphertext, emailLookup, err := protector.EncryptEmail(ctx, "draft-owner@example.test")
	if err != nil {
		t.Fatal(err)
	}
	insertOrder := func(id, reference, status string, createdAt, updatedAt time.Time) {
		t.Helper()
		if _, err := database.ExecContext(ctx, `INSERT INTO claimbounty_orders(id,subject_id,submitter_email_ciphertext,submitter_email_lookup_hash,public_reference,status,version,title,purpose,target_claim_text,target_claim_location,execute_supplied_code,external_search,contains_participant_data,contains_direct_identifiers,created_at,updated_at,retention_policy_version,retention_disposition,source_retention_expires_at,retention_expires_at) VALUES($1,'11111111-1111-4111-8111-111111111111',$2,$3,$4,$5,1,'Study','Audit','Claim','',false,false,false,false,$6,$7,'intake-30d-v1','hard_delete',$8,$8)`, id, emailCiphertext, emailLookup[:], reference, status, createdAt, updatedAt, now.Add(30*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	abandonedID := "22222222-2222-4222-8222-222222222221"
	activeID := "22222222-2222-4222-8222-222222222222"
	insertOrder(abandonedID, "CB-ABANDON00001", "uploading", now.Add(-9*24*time.Hour), now.Add(-8*24*time.Hour))
	insertOrder(activeID, "CB-ACTIVE000001", "draft", now.Add(-6*24*time.Hour), now.Add(-6*24*time.Hour))
	if _, err := database.ExecContext(ctx, `INSERT INTO claimbounty_files(id,order_id,role,original_display_name,size_bytes,sha256,declared_media_type,status,storage_key,storage_etag,object_generation,created_at,updated_at) VALUES('33333333-3333-4333-8333-333333333333',$1,'primary_paper','paper.pdf',4,$2,'application/pdf','uploaded','quarantine/abandoned','etag','abandoned-version',$3,$3)`, abandonedID, strings.Repeat("a", 64), now.Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	if err := store.CleanupExpiredIdentityAndAbandoned(ctx, now, now.Add(-7*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	var abandonedStatus, activeStatus string
	var sourceDeadline, piiDeadline time.Time
	if err := database.QueryRowContext(ctx, `SELECT status,source_retention_expires_at,retention_expires_at FROM claimbounty_orders WHERE id=$1`, abandonedID).Scan(&abandonedStatus, &sourceDeadline, &piiDeadline); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT status FROM claimbounty_orders WHERE id=$1`, activeID).Scan(&activeStatus); err != nil {
		t.Fatal(err)
	}
	if abandonedStatus != "expired" || !sourceDeadline.Equal(now) || !piiDeadline.Equal(now) || activeStatus != "draft" {
		t.Fatalf("abandoned/active cleanup = %q/%s/%s and %q", abandonedStatus, sourceDeadline, piiDeadline, activeStatus)
	}
	if _, err := store.AnonymizeExpired(ctx, now, 10); err != nil {
		t.Fatal(err)
	}
	var abandonedCount, activeCount, tombstoneCount int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM claimbounty_orders WHERE id=$1`, abandonedID).Scan(&abandonedCount); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM claimbounty_orders WHERE id=$1`, activeID).Scan(&activeCount); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM claimbounty_tombstones WHERE order_id=$1 AND final_status='expired'`, abandonedID).Scan(&tombstoneCount); err != nil {
		t.Fatal(err)
	}
	var deletedKey, deletedVersion string
	if err := database.QueryRowContext(ctx, `SELECT storage_key,object_generation FROM claimbounty_outbox WHERE kind='delete_object'`).Scan(&deletedKey, &deletedVersion); err != nil {
		t.Fatal(err)
	}
	if abandonedCount != 0 || activeCount != 1 || tombstoneCount != 1 || deletedKey != "quarantine/abandoned" || deletedVersion != "abandoned-version" {
		t.Fatalf("abandoned cleanup counts/object = %d/%d/%d %q@%q", abandonedCount, activeCount, tombstoneCount, deletedKey, deletedVersion)
	}
}
