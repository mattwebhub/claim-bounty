package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

const (
	testOrderID = "11111111-1111-4111-8111-111111111111"
	testFileID  = "22222222-2222-4222-8222-222222222222"
)

func TestOrderFileEnforcesUploadTrustBoundary(t *testing.T) {
	t.Parallel()

	id := mustClaimIdentifier(t, testFileID)
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	valid := func() (domain.OrderFile, error) {
		return domain.NewOrderFile(id, "primary_paper", "paper.pdf", 1024, strings.Repeat("a", 64), "application/pdf", "quarantine/object", now)
	}
	if _, err := valid(); err != nil {
		t.Fatalf("valid NewOrderFile() error = %v", err)
	}

	for _, test := range []struct {
		name, role, filename, digest, mediaType string
		size                                    int64
	}{
		{"zero byte", "primary_paper", "paper.pdf", strings.Repeat("a", 64), "application/pdf", 0},
		{"primary ceiling", "primary_paper", "paper.pdf", strings.Repeat("a", 64), "application/pdf", 50*1024*1024 + 1},
		{"invalid digest", "primary_paper", "paper.pdf", strings.Repeat("A", 64), "application/pdf", 1},
		{"path separator", "primary_paper", "../paper.pdf", strings.Repeat("a", 64), "application/pdf", 1},
		{"primary content type", "primary_paper", "paper.pdf", strings.Repeat("a", 64), "text/plain", 1},
		{"unknown role", "executable", "run.sh", strings.Repeat("a", 64), "text/plain", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewOrderFile(id, test.role, test.filename, test.size, test.digest, test.mediaType, "quarantine/object", now)
			if !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("NewOrderFile() error = %v, want validation error", err)
			}
		})
	}
}

func TestOrderFileRejectsDisallowedRoleExtensions(t *testing.T) {
	t.Parallel()

	id := mustClaimIdentifier(t, testFileID)
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		role, filename, mediaType string
	}{
		{"data", "photo.jpg", "image/jpeg"},
		{"code", "analysis.class", "application/java-vm"},
		{"supplement", "supplement.csv", "text/csv"},
		{"environment", "notes.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	} {
		t.Run(test.role+"/"+test.filename, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewOrderFile(id, test.role, test.filename, 10, strings.Repeat("d", 64), test.mediaType, "quarantine/object", now)
			if !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("NewOrderFile(%q, %q) error = %v, want validation error", test.role, test.filename, err)
			}
		})
	}
}

func TestOrderFileAcceptsFrozenAllowedRoleExtensions(t *testing.T) {
	t.Parallel()

	id := mustClaimIdentifier(t, testFileID)
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		role, filename, mediaType string
	}{
		{"supplement", "supplement.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"supplement", "notes.md", "text/markdown"},
		{"data", "analysis.RData", "application/octet-stream"},
		{"code", "query.sql", "application/sql"},
		{"code", "reproduce.sh", "application/x-sh"},
		{"other_evidence", "sources.zip", "application/zip"},
	} {
		t.Run(test.role+"/"+test.filename, func(t *testing.T) {
			t.Parallel()
			if _, err := domain.NewOrderFile(id, test.role, test.filename, 10, strings.Repeat("e", 64), test.mediaType, "quarantine/object", now); err != nil {
				t.Fatalf("NewOrderFile(%q, %q) error = %v, want frozen allowlist acceptance", test.role, test.filename, err)
			}
		})
	}
}

func TestOrderFileBindsScannerSignatureToRoleAndExtension(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		role, name, declared, detected string
		accepted                       bool
	}{
		{"supplement", "notes.md", "text/markdown", "text/plain; charset=utf-8", true},
		{"supplement", "appendix.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip", true},
		{"data", "analysis.RData", "application/octet-stream", "application/octet-stream", true},
		{"code", "analysis.sql", "application/sql", "text/plain", true},
		{"code", "run.sh", "text/x-shellscript", "application/pdf", false},
		{"other_evidence", "sources.zip", "application/zip", "application/zip", true},
		{"other_evidence", "sources.zip", "application/zip", "application/x-msdownload", false},
	} {
		t.Run(test.role+"/"+test.name+"/"+test.detected, func(t *testing.T) {
			file, err := domain.NewOrderFile(mustClaimIdentifier(t, testFileID), test.role, test.name, 12, strings.Repeat("a", 64), test.declared, "quarantine/object", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if got := file.AcceptsDetectedMediaType(test.detected); got != test.accepted {
				t.Fatalf("AcceptsDetectedMediaType() = %t, want %t", got, test.accepted)
			}
		})
	}
}

func TestFrozenRoleFormAndMediaMatrix(t *testing.T) {
	t.Parallel()
	allowed := []struct{ role, name, declared, detected string }{
		{"primary_paper", "paper.pdf", "application/pdf", "application/pdf"},
		{"supplement", "appendix.pdf", "application/pdf", "application/pdf"}, {"supplement", "appendix.txt", "text/plain", "text/plain"}, {"supplement", "appendix.md", "text/markdown", "text/plain"}, {"supplement", "appendix.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip"},
		{"preregistration", "plan.pdf", "application/pdf", "application/pdf"}, {"preregistration", "plan.txt", "text/plain", "text/plain"}, {"preregistration", "plan.md", "text/markdown", "text/plain"}, {"preregistration", "plan.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip"},
		{"data", "data.csv", "text/csv", "text/csv"}, {"data", "data.tsv", "text/tab-separated-values", "text/plain"}, {"data", "data.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/zip"}, {"data", "data.json", "application/json", "application/json"}, {"data", "data.parquet", "application/vnd.apache.parquet", "application/octet-stream"}, {"data", "data.dta", "application/x-stata", "application/octet-stream"}, {"data", "data.sav", "application/x-spss-sav", "application/octet-stream"}, {"data", "data.rds", "application/x-r-data", "application/x-r-data"}, {"data", "data.RData", "application/x-r-data", "application/x-r-data"},
		{"code", "analysis.r", "text/x-r-source", "text/x-r-source"}, {"code", "analysis.py", "text/x-python", "text/x-python"}, {"code", "analysis.ipynb", "application/json", "application/json"}, {"code", "analysis.do", "text/plain", "text/plain"}, {"code", "analysis.sql", "application/sql", "application/sql"}, {"code", "analysis.sh", "text/x-shellscript", "text/x-shellscript"}, {"code", "code.zip", "application/zip", "application/zip"},
		{"environment", "Dockerfile", "text/plain", "text/plain"}, {"environment", "package.lock", "text/plain", "text/plain"}, {"environment", "requirements.txt", "text/plain", "text/plain"}, {"environment", "requirements-test.txt", "text/plain", "text/plain"}, {"environment", "renv.lock", "application/json", "application/json"}, {"environment", "environment.yaml", "application/yaml", "application/yaml"}, {"environment", "pyproject.toml", "application/toml", "application/toml"},
		{"data_dictionary", "dictionary.csv", "text/csv", "text/csv"}, {"data_dictionary", "dictionary.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/zip"}, {"data_dictionary", "dictionary.pdf", "application/pdf", "application/pdf"}, {"data_dictionary", "dictionary.txt", "text/plain", "text/plain"}, {"data_dictionary", "dictionary.md", "text/markdown", "text/plain"},
		{"other_evidence", "source.pdf", "application/pdf", "application/pdf"}, {"other_evidence", "source.txt", "text/plain", "text/plain"}, {"other_evidence", "source.md", "text/markdown", "text/plain"}, {"other_evidence", "source.zip", "application/zip", "application/zip"},
	}
	for _, test := range allowed {
		t.Run("allowed/"+test.role+"/"+test.name, func(t *testing.T) {
			file, err := domain.NewOrderFile(mustClaimIdentifier(t, testFileID), test.role, test.name, 12, strings.Repeat("a", 64), test.declared, "quarantine/object", time.Now())
			if err != nil {
				t.Fatalf("NewOrderFile() error = %v", err)
			}
			if !file.AcceptsDetectedMediaType(test.detected) {
				t.Fatalf("detected media type %q was rejected", test.detected)
			}
		})
	}
	rejected := []struct{ role, name, media string }{
		{"data", "data.feather", "application/octet-stream"}, {"data", "data.gz", "application/gzip"}, {"data", "data.jsonl", "application/json"}, {"data", "data.xls", "application/vnd.ms-excel"}, {"data", "data.zip", "application/zip"},
		{"code", "Makefile", "text/plain"}, {"code", "analysis.jl", "text/plain"}, {"code", "config.json", "application/json"}, {"code", "deps.lock", "text/plain"}, {"code", "notes.md", "text/markdown"},
		{"environment", "Dockerfile.exe", "text/plain"}, {"environment", "Dockerfile.json", "application/json"}, {"environment", "notes.txt", "text/plain"}, {"environment", "config.json", "application/json"},
		{"data_dictionary", "dictionary.json", "application/json"}, {"data_dictionary", "dictionary.tsv", "text/tab-separated-values"}, {"data_dictionary", "dictionary.xls", "application/vnd.ms-excel"},
		{"other_evidence", "source.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	}
	for _, test := range rejected {
		t.Run("rejected/"+test.role+"/"+test.name, func(t *testing.T) {
			_, err := domain.NewOrderFile(mustClaimIdentifier(t, testFileID), test.role, test.name, 12, strings.Repeat("b", 64), test.media, "quarantine/object", time.Now())
			if !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("NewOrderFile() error = %v, want validation error", err)
			}
		})
	}
}

func TestOrderFileUsesFrozenCaseBundleDirectories(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		role, directory, filename, mediaType string
	}{
		{"primary_paper", "paper", "paper.pdf", "application/pdf"},
		{"supplement", "supplements", "supplement.pdf", "application/pdf"},
		{"preregistration", "preregistration", "plan.md", "text/markdown"},
		{"data", "data", "data.csv", "text/csv"},
		{"code", "code", "analysis.py", "text/x-python"},
		{"environment", "environment", "requirements.txt", "text/plain"},
		{"data_dictionary", "dictionaries", "dictionary.csv", "text/csv"},
		{"other_evidence", "sources", "source.pdf", "application/pdf"},
	} {
		t.Run(test.role, func(t *testing.T) {
			t.Parallel()
			file, err := domain.NewOrderFile(mustClaimIdentifier(t, testFileID), test.role, test.filename, 10, strings.Repeat("b", 64), test.mediaType, "quarantine/object", now)
			if err != nil {
				t.Fatalf("NewOrderFile() error = %v", err)
			}
			wantPrefix := test.directory + "/" + testFileID + "-"
			if got := file.CaseBundlePath(); !strings.HasPrefix(got, wantPrefix) {
				t.Fatalf("CaseBundlePath() = %q, want prefix %q", got, wantPrefix)
			}
		})
	}
}

func TestOrderSubmitIsVersionedAndSingleUse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	orderID := mustClaimIdentifier(t, testOrderID)
	order, err := domain.NewOrder(orderID, mustClaimIdentifier(t, "33333333-3333-4333-8333-333333333333"), "researcher@example.test", "CB-ABC123DEF456", "Study", "Verify the result", domain.TargetClaim{Text: "The treatment improved scores", SourceLocation: "Table 2"}, domain.Permissions{}, domain.Privacy{}, now)
	if err != nil {
		t.Fatal(err)
	}
	file, err := domain.NewOrderFile(mustClaimIdentifier(t, testFileID), "primary_paper", "paper.pdf", 10, strings.Repeat("c", 64), "application/pdf", "quarantine/object", now)
	if err != nil {
		t.Fatal(err)
	}
	file, err = file.ConfirmUpload("etag", "generation-1", now)
	if err != nil {
		t.Fatal(err)
	}
	order, err = order.ReserveFile(file, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	order, err = order.Submit(2, "terms-v1", true, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if order.Version != 3 || order.Status != "submitted" {
		t.Fatalf("submitted order = version %d, status %q", order.Version, order.Status)
	}
	if _, err := order.Submit(3, "terms-v1", true, now.Add(2*time.Second)); !errors.Is(err, domain.ErrStateConflict) {
		t.Fatalf("second Submit() error = %v, want state conflict", err)
	}
	if _, err := order.QueueExport(2, now); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale QueueExport() error = %v, want version conflict", err)
	}
}

func mustClaimIdentifier(t *testing.T, raw string) domain.Identifier {
	t.Helper()
	id, err := domain.NewIdentifier(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
