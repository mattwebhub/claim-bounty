package postgres

import (
	"errors"
	"strings"
	"testing"
)

func TestDatabaseFailurePreservesClassificationWithoutLeakingProviderText(t *testing.T) {
	t.Parallel()
	cause := errors.New("password=secret sql=SELECT private_column FROM users")
	err := databaseFailure("query project", cause)
	if !errors.Is(err, cause) {
		t.Fatal("database error did not preserve its cause for internal classification")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "SELECT") {
		t.Fatalf("database error leaked provider details: %q", err)
	}
	if got, want := err.Error(), "postgres: query project failed"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
