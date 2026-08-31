package system

import (
	"context"
	"encoding/base64"
	"testing"
)

func TestEmailProtectorRoundTripAndStableLookup(t *testing.T) {
	t.Parallel()
	protector, err := NewEmailProtector(
		base64.StdEncoding.EncodeToString([]byte("test-email-encryption-key-000000")),
		base64.StdEncoding.EncodeToString([]byte("test-email-lookup-hmac-key-00000")),
	)
	if err != nil {
		t.Fatal(err)
	}
	first, lookup, err := protector.EncryptEmail(context.Background(), " Owner@Example.Test ")
	if err != nil {
		t.Fatal(err)
	}
	second, secondLookup, err := protector.EncryptEmail(context.Background(), "owner@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if string(first) == string(second) {
		t.Fatal("encryption reused a nonce")
	}
	if lookup != secondLookup || lookup != protector.LookupEmail("OWNER@example.test") {
		t.Fatal("normalized lookup hashes differ")
	}
	plain, err := protector.DecryptEmail(context.Background(), first)
	if err != nil || plain != "owner@example.test" {
		t.Fatalf("DecryptEmail() = %q, %v", plain, err)
	}
	first[len(first)-1] ^= 1
	if _, err := protector.DecryptEmail(context.Background(), first); err == nil {
		t.Fatal("DecryptEmail() accepted modified ciphertext")
	}
}

func TestEmailProtectorRejectsWeakOrSharedKeys(t *testing.T) {
	t.Parallel()
	strong := base64.StdEncoding.EncodeToString([]byte("test-email-encryption-key-000000"))
	for _, keys := range [][2]string{{"bad", strong}, {strong, strong}} {
		if _, err := NewEmailProtector(keys[0], keys[1]); err == nil {
			t.Fatal("NewEmailProtector() accepted invalid keys")
		}
	}
}
