package objectstore

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

func TestProcessStorageScopesEnforcePrefixPolicy(t *testing.T) {
	t.Parallel()
	backend := &scopeStoreStub{}
	api, err := NewScope(backend, []string{"accepted/sha256/", "exports/"}, []string{"quarantine/"}, []string{"quarantine/"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.PutWriteOnce(context.Background(), "quarantine/object", bytes.NewReader([]byte("x")), 1, "text/plain", "digest"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.PutWriteOnce(context.Background(), "exports/object", bytes.NewReader([]byte("x")), 1, "application/zip", "digest"); err == nil {
		t.Fatal("API scope wrote an export")
	}
	if _, _, err := api.Open(context.Background(), "quarantine/object", "v1"); err == nil {
		t.Fatal("API scope read quarantine")
	}
	if _, _, err := api.Open(context.Background(), "accepted/sha256/digest", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteVersion(context.Background(), "accepted/sha256/digest", "v1"); err == nil {
		t.Fatal("API scope deleted accepted content")
	}
}

type scopeStoreStub struct{}

func (*scopeStoreStub) Open(context.Context, string, string) (ports.ObjectReader, ports.ObjectMetadata, error) {
	return io.NopCloser(bytes.NewReader(nil)), ports.ObjectMetadata{}, nil
}
func (*scopeStoreStub) PutWriteOnce(context.Context, string, io.Reader, int64, string, string) (ports.ObjectMetadata, error) {
	return ports.ObjectMetadata{}, nil
}
func (*scopeStoreStub) DeleteVersion(context.Context, string, string) error { return nil }
