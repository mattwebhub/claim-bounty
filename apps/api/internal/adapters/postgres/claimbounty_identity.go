package postgres

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

func (s *Store) EnforceRateLimit(ctx context.Context, scope string, key [32]byte, now time.Time, window time.Duration, limit int) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var count int
	err := s.pool.QueryRow(ctx, `
INSERT INTO claimbounty_rate_limits(scope,key_hash,window_started_at,request_count) VALUES($1,$2,$3,1)
ON CONFLICT(scope,key_hash) DO UPDATE SET
 window_started_at=CASE WHEN claimbounty_rate_limits.window_started_at <= $3-$4::interval THEN $3 ELSE claimbounty_rate_limits.window_started_at END,
 request_count=CASE WHEN claimbounty_rate_limits.window_started_at <= $3-$4::interval THEN 1 ELSE claimbounty_rate_limits.request_count+1 END
RETURNING request_count`, scope, key[:], now.UTC(), window.String()).Scan(&count)
	if err != nil {
		return databaseFailure("enforce rate limit", err)
	}
	if count > limit {
		return domain.ErrRateLimited
	}
	return nil
}

func (s *Store) CreateChallenge(ctx context.Context, challenge ports.Challenge) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	ciphertext, lookup, err := s.protectEmail(ctx, challenge.Email)
	if err != nil {
		return err
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		subjectID := challenge.SubjectID.String()
		var retainedSubject string
		err := tx.QueryRow(ctx, `SELECT subject_id::text FROM claimbounty_orders WHERE submitter_email_lookup_hash=$1 AND $2='submitter' ORDER BY created_at DESC,id DESC LIMIT 1`, lookup[:], string(challenge.Audience)).Scan(&retainedSubject)
		if err == nil {
			subjectID = retainedSubject
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO email_challenges(id,subject_id,email_ciphertext,email_lookup_hash,audience,token_hash,expires_at,attempts_remaining) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, challenge.ID.String(), subjectID, ciphertext, lookup[:], string(challenge.Audience), challenge.TokenHash[:], challenge.ExpiresAt.UTC(), challenge.AttemptsRemaining)
		return err
	})
	if err != nil {
		return databaseFailure("create email challenge", err)
	}
	return nil
}

func (s *Store) ExchangeChallenge(ctx context.Context, email string, audience domain.Audience, tokenHash [32]byte, sessionID domain.Identifier, sessionHash, csrfHash [32]byte, policyVersion string, now, expires time.Time) (ports.SessionCredential, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var result ports.SessionCredential
	invalidChallenge := false
	ciphertext, lookup, protectErr := s.protectEmail(ctx, email)
	if protectErr != nil {
		return ports.SessionCredential{}, protectErr
	}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var challengeID, subjectID string
		var storedHash []byte
		var challengeExpires time.Time
		var attempts int
		err := tx.QueryRow(ctx, `SELECT id::text,subject_id::text,token_hash,expires_at,attempts_remaining FROM email_challenges WHERE email_lookup_hash=$1 AND audience=$2 AND used_at IS NULL ORDER BY expires_at DESC LIMIT 1 FOR UPDATE`, lookup[:], string(audience)).Scan(&challengeID, &subjectID, &storedHash, &challengeExpires, &attempts)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInvalidChallenge
		}
		if err != nil {
			return err
		}
		if attempts <= 0 || !challengeExpires.After(now) || !bytes.Equal(storedHash, tokenHash[:]) {
			if attempts > 0 {
				if _, err := tx.Exec(ctx, `UPDATE email_challenges SET attempts_remaining=attempts_remaining-1 WHERE id=$1`, challengeID); err != nil {
					return err
				}
			}
			invalidChallenge = true
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE email_challenges SET used_at=$2 WHERE id=$1 AND used_at IS NULL`, challengeID, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO claimbounty_sessions(id,subject_id,email_ciphertext,email_lookup_hash,audience,authorization_policy_version,token_hash,csrf_hash,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, sessionID.String(), subjectID, ciphertext, lookup[:], string(audience), policyVersion, sessionHash[:], csrfHash[:], expires.UTC()); err != nil {
			return err
		}
		subject, err := domain.NewIdentifier(subjectID)
		if err != nil {
			return err
		}
		session, err := domain.NewSession(sessionID, subject, email, audience, policyVersion, expires)
		if err != nil {
			return err
		}
		result = ports.SessionCredential{Session: session, TokenHash: sessionHash, CSRFHash: csrfHash}
		return nil
	})
	if invalidChallenge {
		return ports.SessionCredential{}, domain.ErrInvalidChallenge
	}
	if errors.Is(err, domain.ErrInvalidChallenge) {
		return ports.SessionCredential{}, err
	}
	if err != nil {
		return ports.SessionCredential{}, databaseFailure("exchange email challenge", err)
	}
	return result, nil
}

func (s *Store) GetSession(ctx context.Context, tokenHash [32]byte, now time.Time) (ports.SessionCredential, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var id, subjectID, email, audience, policyVersion string
	var emailCiphertext []byte
	var csrfHash []byte
	var expires time.Time
	err := s.pool.QueryRow(ctx, `SELECT id::text,subject_id::text,email_ciphertext,audience,authorization_policy_version,csrf_hash,expires_at FROM claimbounty_sessions WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>$2`, tokenHash[:], now.UTC()).Scan(&id, &subjectID, &emailCiphertext, &audience, &policyVersion, &csrfHash, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.SessionCredential{}, domain.ErrUnauthorized
	}
	if err != nil {
		return ports.SessionCredential{}, databaseFailure("get session", err)
	}
	email, err = s.revealEmail(ctx, emailCiphertext)
	if err != nil {
		return ports.SessionCredential{}, databaseFailure("restore session email", err)
	}
	sessionID, err := domain.NewIdentifier(id)
	if err != nil {
		return ports.SessionCredential{}, databaseFailure("restore session", err)
	}
	subject, err := domain.NewIdentifier(subjectID)
	if err != nil {
		return ports.SessionCredential{}, databaseFailure("restore session", err)
	}
	parsedAudience, err := domain.NewAudience(audience)
	if err != nil {
		return ports.SessionCredential{}, databaseFailure("restore session", err)
	}
	session, err := domain.NewSession(sessionID, subject, email, parsedAudience, policyVersion, expires)
	if err != nil {
		return ports.SessionCredential{}, databaseFailure("restore session", err)
	}
	var csrf [32]byte
	if len(csrfHash) != len(csrf) {
		return ports.SessionCredential{}, databaseFailure("restore session", errors.New("invalid csrf digest"))
	}
	copy(csrf[:], csrfHash)
	return ports.SessionCredential{Session: session, TokenHash: tokenHash, CSRFHash: csrf}, nil
}

func (s *Store) RotateCSRF(ctx context.Context, sessionID domain.Identifier, previousHash, csrfHash [32]byte, now time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	tag, err := s.pool.Exec(ctx, `UPDATE claimbounty_sessions SET csrf_hash=$3 WHERE id=$1 AND csrf_hash=$2 AND revoked_at IS NULL AND expires_at>$4`, sessionID.String(), previousHash[:], csrfHash[:], now.UTC())
	if err != nil {
		return databaseFailure("rotate csrf", err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrStateConflict
	}
	return nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash [32]byte, now time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	tag, err := s.pool.Exec(ctx, `UPDATE claimbounty_sessions SET revoked_at=$2 WHERE token_hash=$1 AND revoked_at IS NULL`, tokenHash[:], now.UTC())
	if err != nil {
		return databaseFailure("revoke session", err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrUnauthorized
	}
	return nil
}
