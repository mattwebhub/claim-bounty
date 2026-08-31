package system

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

type EmailProtector struct {
	aead      cipher.AEAD
	lookupKey []byte
}

func NewEmailProtector(encryptionKeyBase64, lookupKeyBase64 string) (*EmailProtector, error) {
	encryptionKey, err := base64.StdEncoding.DecodeString(encryptionKeyBase64)
	if err != nil || len(encryptionKey) != 32 {
		return nil, errors.New("system: email encryption key must be base64-encoded 32 bytes")
	}
	lookupKey, err := base64.StdEncoding.DecodeString(lookupKeyBase64)
	if err != nil || len(lookupKey) < 32 || hmac.Equal(encryptionKey, lookupKey) {
		return nil, errors.New("system: email lookup key must be a distinct base64-encoded key of at least 32 bytes")
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, errors.New("system: initialize email encryption")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New("system: initialize authenticated email encryption")
	}
	return &EmailProtector{aead: aead, lookupKey: append([]byte(nil), lookupKey...)}, nil
}

func (protector *EmailProtector) EncryptEmail(_ context.Context, email string) ([]byte, [32]byte, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return nil, [32]byte{}, errors.New("system: email is required")
	}
	nonce := make([]byte, protector.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, [32]byte{}, errors.New("system: email nonce generation failed")
	}
	ciphertext := append([]byte{1}, nonce...)
	ciphertext = protector.aead.Seal(ciphertext, nonce, []byte(normalized), []byte("claimbounty-email-v1"))
	return ciphertext, protector.LookupEmail(normalized), nil
}

func (protector *EmailProtector) DecryptEmail(_ context.Context, ciphertext []byte) (string, error) {
	nonceSize := protector.aead.NonceSize()
	if len(ciphertext) <= 1+nonceSize || ciphertext[0] != 1 {
		return "", errors.New("system: invalid encrypted email")
	}
	plaintext, err := protector.aead.Open(nil, ciphertext[1:1+nonceSize], ciphertext[1+nonceSize:], []byte("claimbounty-email-v1"))
	if err != nil {
		return "", errors.New("system: email authentication failed")
	}
	return string(plaintext), nil
}

func (protector *EmailProtector) LookupEmail(email string) [32]byte {
	mac := hmac.New(sha256.New, protector.lookupKey)
	_, _ = mac.Write([]byte(strings.ToLower(strings.TrimSpace(email))))
	var digest [32]byte
	copy(digest[:], mac.Sum(nil))
	return digest
}

type SecureValues struct{}

func (SecureValues) NewIdentifier(context.Context) (domain.Identifier, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("system: secure identifier generation failed")
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	raw := fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(b[0:4]), hex.EncodeToString(b[4:6]), hex.EncodeToString(b[6:8]), hex.EncodeToString(b[8:10]), hex.EncodeToString(b[10:16]))
	return domain.NewIdentifier(raw)
}

func (SecureValues) NewOpaqueToken(_ context.Context, size int) (string, error) {
	if size < 16 || size > 128 {
		return "", errors.New("system: invalid token size")
	}
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("system: secure token generation failed")
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (SecureValues) NewChallengeCode(context.Context) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", errors.New("system: secure challenge generation failed")
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (SecureValues) NewObjectKey(ctx context.Context, prefix string) (string, error) {
	token, err := (SecureValues{}).NewOpaqueToken(ctx, 32)
	if err != nil {
		return "", err
	}
	if prefix != "quarantine" && prefix != "exports" {
		return "", errors.New("system: invalid object prefix")
	}
	return prefix + "/" + token, nil
}

func (SecureValues) NewPublicReference(context.Context) (string, error) {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	b := make([]byte, 12)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", errors.New("system: secure reference generation failed")
		}
		b[i] = alphabet[n.Int64()]
	}
	return "CB-" + string(b), nil
}

type AllowlistPolicy struct {
	emails           map[string]struct{}
	version          string
	allowlistVersion string
}

func NewAllowlistPolicy(emails []string, version, allowlistVersion string) (*AllowlistPolicy, error) {
	if !domain.ValidPolicyVersion(version) || !domain.ValidPolicyVersion(allowlistVersion) {
		return nil, errors.New("system: admin policy versions are invalid")
	}
	values := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email != "" {
			values[email] = struct{}{}
		}
	}
	return &AllowlistPolicy{emails: values, version: version, allowlistVersion: allowlistVersion}, nil
}

func (policy *AllowlistPolicy) Authorize(_ context.Context, email, policyVersion string) error {
	if policyVersion != "" && policyVersion != policy.version {
		return domain.ErrForbidden
	}
	if _, ok := policy.emails[strings.ToLower(strings.TrimSpace(email))]; !ok {
		return domain.ErrForbidden
	}
	return nil
}

func (policy *AllowlistPolicy) Version() string          { return policy.version }
func (policy *AllowlistPolicy) AllowlistVersion() string { return policy.allowlistVersion }
