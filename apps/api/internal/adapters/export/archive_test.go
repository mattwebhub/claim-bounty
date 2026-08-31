package export_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jsoncanonicalizer "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	claimexport "github.com/mattwebhub/micro1-template/apps/api/internal/adapters/export"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/validation"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

func TestBuilderReadsExactObjectVersionAndProducesDispatchArchive(t *testing.T) {
	t.Parallel()

	archive, storage := buildTestArchive(t)
	for _, name := range []string{
		"case-bundle/CASE-MANIFEST.json",
		"case-bundle/paper/33333333-3333-4333-8333-333333333333-paper.pdf",
		"dispatch/audit-request.json",
		"dispatch/scientific-policy.json",
		"dispatch/execution-policy.json",
	} {
		if _, ok := archive[name]; !ok {
			t.Errorf("archive is missing %q", name)
		}
	}
	if storage.openKey != "quarantine/source" || storage.openGeneration != "version-source-1" {
		t.Fatalf("source read = %q@%q, want exact recorded version", storage.openKey, storage.openGeneration)
	}
	if storage.putKey != "exports/result" || storage.putGeneration == "" {
		t.Fatalf("archive write = %q@%q", storage.putKey, storage.putGeneration)
	}
}

func TestBuilderIncludesFrozenOperatorReadme(t *testing.T) {
	t.Parallel()

	archive, _ := buildTestArchive(t)
	readme, ok := archive["README.md"]
	if !ok {
		t.Fatal("archive is missing frozen README.md dispatch instructions")
	}
	if !bytes.Contains(readme, []byte("case-bundle")) || !bytes.Contains(readme, []byte("dispatch")) {
		t.Fatalf("README.md does not identify manual dispatch inputs: %q", readme)
	}
}

func TestVerifierAcceptsBuilderArchive(t *testing.T) {
	t.Parallel()

	_, storage := buildTestArchive(t)
	archivePath := filepath.Join(t.TempDir(), "claimbounty.zip")
	if err := os.WriteFile(archivePath, storage.archive, 0o600); err != nil {
		t.Fatal(err)
	}
	schemas, err := validation.New()
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := claimexport.NewVerifier(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyFile(archivePath, digest(storage.archive)); err != nil {
		t.Fatalf("VerifyFile() error = %v", err)
	}
}

func TestVerifierRejectsWholeArchiveDigestBeforeZIPParsing(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "not-a-zip")
	if err := os.WriteFile(archivePath, []byte("not a ZIP archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	schemas, err := validation.New()
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := claimexport.NewVerifier(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyFile(archivePath, strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), "expected archive SHA-256 mismatch") {
		t.Fatalf("VerifyFile() error = %v, want whole-archive digest mismatch before ZIP error", err)
	}
}

func TestVerifierRejectsTamperedCanonicalManifest(t *testing.T) {
	t.Parallel()
	_, storage := buildTestArchive(t)
	tampered := rewriteArchiveEntry(t, storage.archive, "case-bundle/CASE-MANIFEST.json", func(contents []byte) []byte {
		var document map[string]any
		if err := json.Unmarshal(contents, &document); err != nil {
			t.Fatal(err)
		}
		document["publicReference"] = "CB-TAMPERED0001"
		updated, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		return updated
	})
	archivePath := filepath.Join(t.TempDir(), "tampered.zip")
	if err := os.WriteFile(archivePath, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	schemas, err := validation.New()
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := claimexport.NewVerifier(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyFile(archivePath, digest(tampered)); err == nil || !strings.Contains(err.Error(), "manifest checksum mismatch") {
		t.Fatalf("VerifyFile() error = %v, want manifest checksum mismatch", err)
	}
}

func TestVerifierRejectsDuplicateUnicodeNormalizedManifestPaths(t *testing.T) {
	t.Parallel()
	_, storage := buildTestArchive(t)
	tampered := rewriteArchiveEntry(t, storage.archive, "case-bundle/CASE-MANIFEST.json", func(contents []byte) []byte {
		var document map[string]any
		if err := json.Unmarshal(contents, &document); err != nil {
			t.Fatal(err)
		}
		files := document["files"].([]any)
		first := files[0].(map[string]any)
		first["path"] = "paper/caf\u00e9.pdf"
		cloneBytes, err := json.Marshal(first)
		if err != nil {
			t.Fatal(err)
		}
		var duplicate map[string]any
		if err := json.Unmarshal(cloneBytes, &duplicate); err != nil {
			t.Fatal(err)
		}
		duplicate["path"] = "paper/cafe\u0301.pdf"
		duplicate["role"] = "supplement"
		duplicate["fileId"] = "44444444-4444-4444-8444-444444444444"
		document["files"] = []any{first, duplicate}
		delete(document, "manifestSha256")
		withoutDigest, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		canonical, err := jsoncanonicalizer.Transform(withoutDigest)
		if err != nil {
			t.Fatal(err)
		}
		document["manifestSha256"] = digest(canonical)
		updated, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		return updated
	})
	archivePath := filepath.Join(t.TempDir(), "duplicate-manifest.zip")
	if err := os.WriteFile(archivePath, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	schemas, err := validation.New()
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := claimexport.NewVerifier(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyFile(archivePath, digest(tampered)); err == nil || !strings.Contains(err.Error(), "duplicate normalized member paths") {
		t.Fatalf("VerifyFile() error = %v, want duplicate normalized manifest path", err)
	}
}

func rewriteArchiveEntry(t *testing.T, source []byte, target string, mutate func([]byte) []byte) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(source), int64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		entry, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(entry)
		_ = entry.Close()
		if err != nil {
			t.Fatal(err)
		}
		if file.Name == target {
			contents = mutate(contents)
		}
		created, err := writer.CreateHeader(&file.FileHeader)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := created.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestVerifierExtractsProductArchiveWithoutReshaping(t *testing.T) {
	t.Parallel()
	_, storage := buildTestArchive(t)
	root := t.TempDir()
	archivePath := filepath.Join(root, "claimbounty.zip")
	if err := os.WriteFile(archivePath, storage.archive, 0o600); err != nil {
		t.Fatal(err)
	}
	schemas, err := validation.New()
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := claimexport.NewVerifier(schemas)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "verified")
	paths, err := verifier.VerifyAndExtract(archivePath, digest(storage.archive), destination)
	if err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{paths.CaseBundle, paths.AuditRequest, paths.ScientificPolicy, paths.ExecutionPolicy} {
		if _, err := os.Stat(filename); err != nil {
			t.Fatalf("extracted path %q: %v", filename, err)
		}
	}
	if _, err := verifier.VerifyAndExtract(archivePath, digest(storage.archive), destination); err == nil {
		t.Fatal("VerifyAndExtract() accepted an existing destination")
	}
}

func TestVerifierRejectsTraversalBeforeExtraction(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "traversal.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("../outside")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("unsafe"))
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	schemas, err := validation.New()
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := claimexport.NewVerifier(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyFile(archivePath, digestFile(t, archivePath)); err == nil || !strings.Contains(err.Error(), "unsafe archive path") {
		t.Fatalf("VerifyFile() error = %v, want unsafe archive path", err)
	}
}

func TestVerifierRejectsDuplicateSymlinkAndCompressionBombEntries(t *testing.T) {
	t.Parallel()

	schemas, err := validation.New()
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := claimexport.NewVerifier(schemas)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, want string
		write      func(*testing.T, *zip.Writer)
	}{
		{"duplicate normalized path", "duplicate normalized path", func(t *testing.T, archive *zip.Writer) {
			writeTestEntry(t, archive, "README.md", 0, []byte("one"))
			writeTestEntry(t, archive, "readme.md", 0, []byte("two"))
		}},
		{"NFC and NFD path collision", "duplicate normalized path", func(t *testing.T, archive *zip.Writer) {
			writeTestEntry(t, archive, "case-bundle/data/caf\u00e9.csv", 0, []byte("one"))
			writeTestEntry(t, archive, "case-bundle/data/cafe\u0301.csv", 0, []byte("two"))
		}},
		{"symlink", "symlink or special entry", func(t *testing.T, archive *zip.Writer) {
			writeTestEntry(t, archive, "case-bundle/link", os.ModeSymlink|0o777, []byte("target"))
		}},
		{"compression bomb", "compression ratio overrun", func(t *testing.T, archive *zip.Writer) {
			writeTestEntry(t, archive, "case-bundle/bomb", 0, bytes.Repeat([]byte{0}, 1<<20))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "unsafe.zip")
			file, err := os.Create(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			archive := zip.NewWriter(file)
			test.write(t, archive)
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if err := verifier.VerifyFile(archivePath, digestFile(t, archivePath)); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("VerifyFile() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func writeTestEntry(t *testing.T, archive *zip.Writer, name string, mode os.FileMode, contents []byte) {
	t.Helper()
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	if mode != 0 {
		header.SetMode(mode)
	}
	entry, err := archive.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(contents); err != nil {
		t.Fatal(err)
	}
}

func digestFile(t *testing.T, filename string) string {
	t.Helper()
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return digest(contents)
}

func buildTestArchive(t *testing.T) (map[string][]byte, *archiveStoreStub) {
	t.Helper()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	paper := []byte("%PDF-1.7\nscientific evidence")
	paperSHA := digest(paper)
	fileID := mustArchiveID(t, "33333333-3333-4333-8333-333333333333")
	file, err := domain.NewOrderFile(fileID, "primary_paper", "paper.pdf", int64(len(paper)), paperSHA, "application/pdf", "quarantine/source", now)
	if err != nil {
		t.Fatal(err)
	}
	file, err = file.ConfirmUpload("etag-source", "version-source-1", now)
	if err != nil {
		t.Fatal(err)
	}
	file, err = file.InspectionResult(true, "application/pdf", "", now)
	if err != nil {
		t.Fatal(err)
	}
	audit := readContractExample(t, "audit-request.json")
	scientificPolicy := readContractExample(t, "scientific-policy.json")
	executionPolicy := readContractExample(t, "execution-policy.json")
	order := domain.Order{
		ID: mustArchiveID(t, "11111111-1111-4111-8111-111111111111"), PublicReference: "CB-ABC123DEF456", Files: []domain.OrderFile{file},
		Intake: &domain.AdminIntake{AuditRequest: audit, ScientificPolicy: scientificPolicy, ExecutionPolicy: executionPolicy},
	}
	export := domain.Export{
		ID: mustArchiveID(t, "22222222-2222-4222-8222-222222222222"), OrderID: order.ID, Status: "building",
		RoutineID: "claim-bounty-operations/run-claimbounty-scientific-audit", RoutineRevision: "sha256:" + strings.Repeat("a", 64),
		RoutineValidatedAt: now, RoutineEvidenceSHA: strings.Repeat("b", 64), StorageKey: "exports/result", CreatedAt: now,
	}
	storage := &archiveStoreStub{source: paper, sourceSHA: paperSHA}
	schemas, err := validation.New()
	if err != nil {
		t.Fatal(err)
	}
	builder, err := claimexport.NewBuilder(storage, schemas, archivePolicyStub{})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := builder.Build(context.Background(), order, export)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if metadata.SHA256 != digest(storage.archive) || metadata.SizeBytes != int64(len(storage.archive)) {
		t.Fatalf("archive metadata = %#v", metadata)
	}
	reader, err := zip.NewReader(bytes.NewReader(storage.archive), int64(len(storage.archive)))
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string][]byte, len(reader.File))
	for _, entry := range reader.File {
		source, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		value, err := io.ReadAll(source)
		_ = source.Close()
		if err != nil {
			t.Fatal(err)
		}
		if _, duplicate := entries[entry.Name]; duplicate {
			t.Fatalf("duplicate archive path %q", entry.Name)
		}
		entries[entry.Name] = value
	}
	return entries, storage
}

func readContractExample(t *testing.T, name string) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("../../../../../contracts/examples/v1", name))
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

type archivePolicyStub struct{}

func (archivePolicyStub) Authorize(context.Context, string, string) error { return nil }
func (archivePolicyStub) Version() string                                 { return "authorization-v1" }
func (archivePolicyStub) AllowlistVersion() string                        { return "admin-allowlist-v1" }

type archiveStoreStub struct {
	source, archive         []byte
	sourceSHA               string
	openKey, openGeneration string
	putKey, putGeneration   string
}

func (store *archiveStoreStub) Open(_ context.Context, key, generation string) (ports.ObjectReader, ports.ObjectMetadata, error) {
	store.openKey, store.openGeneration = key, generation
	if key != "quarantine/source" || generation != "version-source-1" {
		return nil, ports.ObjectMetadata{}, errors.New("unexpected object version")
	}
	return io.NopCloser(bytes.NewReader(store.source)), ports.ObjectMetadata{SizeBytes: int64(len(store.source)), SHA256: store.sourceSHA, Generation: generation}, nil
}
func (store *archiveStoreStub) PutWriteOnce(_ context.Context, key string, reader io.Reader, size int64, _ string, expectedSHA string) (ports.ObjectMetadata, error) {
	store.putKey = key
	store.archive, _ = io.ReadAll(reader)
	if size != int64(len(store.archive)) || expectedSHA != digest(store.archive) {
		return ports.ObjectMetadata{}, errors.New("archive size or checksum mismatch")
	}
	store.putGeneration = "version-export-1"
	return ports.ObjectMetadata{SizeBytes: size, SHA256: expectedSHA, Generation: store.putGeneration}, nil
}
func (*archiveStoreStub) DeleteVersion(context.Context, string, string) error { return nil }

func mustArchiveID(t *testing.T, raw string) domain.Identifier {
	t.Helper()
	id, err := domain.NewIdentifier(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
