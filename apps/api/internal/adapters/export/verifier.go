package export

import (
	"archive/zip"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
	"golang.org/x/text/unicode/norm"
)

const (
	maxArchiveEntries   = domain.MaxOrderFiles + 5
	maxArchiveBytes     = domain.MaxOrderBytes + 10<<20
	maxArchiveJSONBytes = 1 << 20
	maxCompressionRatio = 200
)

type Verifier struct {
	validator ports.IntakeValidator
}

type ExtractedPaths struct {
	Destination      string
	CaseBundle       string
	AuditRequest     string
	ScientificPolicy string
	ExecutionPolicy  string
}

func NewVerifier(validator ports.IntakeValidator) (*Verifier, error) {
	if validator == nil {
		return nil, errors.New("export verifier: validator is required")
	}
	return &Verifier{validator: validator}, nil
}

// VerifyFile validates a local export without opening network connections or
// extracting any archive member to disk.
func (verifier *Verifier) VerifyFile(filename, expectedSHA256 string) error {
	archiveFile, archive, err := openVerifiedArchive(filename, expectedSHA256)
	if err != nil {
		return err
	}
	defer archiveFile.Close()
	return verifier.verifyArchive(archive.File)
}

func openVerifiedArchive(filename, expectedSHA256 string) (*os.File, *zip.Reader, error) {
	expected, err := parseExpectedSHA256(expectedSHA256)
	if err != nil {
		return nil, nil, err
	}
	archive, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("export verification failed: open archive: %w", err)
	}
	closeWithError := func(err error) (*os.File, *zip.Reader, error) {
		_ = archive.Close()
		return nil, nil, err
	}
	info, err := archive.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxArchiveBytes {
		return closeWithError(errors.New("export verification failed: archive must be a bounded regular file"))
	}
	hasher := sha256.New()
	written, err := io.Copy(hasher, io.LimitReader(archive, maxArchiveBytes+1))
	if err != nil || written != info.Size() || subtle.ConstantTimeCompare(hasher.Sum(nil), expected) != 1 {
		return closeWithError(errors.New("export verification failed: expected archive SHA-256 mismatch"))
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return closeWithError(errors.New("export verification failed: rewind verified archive"))
	}
	zipReader, err := zip.NewReader(archive, info.Size())
	if err != nil {
		return closeWithError(fmt.Errorf("export verification failed: open ZIP: %w", err))
	}
	return archive, zipReader, nil
}

func parseExpectedSHA256(value string) ([]byte, error) {
	digestBytes, err := hex.DecodeString(value)
	if err != nil || len(digestBytes) != sha256.Size || hex.EncodeToString(digestBytes) != value {
		return nil, errors.New("export verification failed: expected archive digest must be 64 lowercase SHA-256 hexadecimal characters")
	}
	return digestBytes, nil
}

func (verifier *Verifier) verifyArchive(files []*zip.File) error {
	if len(files) == 0 || len(files) > maxArchiveEntries {
		return fmt.Errorf("export verification failed: archive entry count must be between 1 and %d", maxArchiveEntries)
	}
	entries, err := indexArchive(files)
	if err != nil {
		return err
	}
	manifestEntry, ok := entries["case-bundle/case-manifest.json"]
	if !ok {
		return errors.New("export verification failed: case-bundle/CASE-MANIFEST.json is missing")
	}
	manifestBytes, err := readBoundedEntry(manifestEntry, maxArchiveJSONBytes)
	if err != nil {
		return fmt.Errorf("export verification failed: read manifest: %w", err)
	}
	if err := verifier.validator.ValidateCaseManifest(manifestBytes); err != nil {
		return errors.New("export verification failed: manifest does not match schema version 1.0.0")
	}
	var document manifest
	if err := json.Unmarshal(manifestBytes, &document); err != nil {
		return errors.New("export verification failed: manifest is not valid JSON")
	}
	claimedManifestSHA := document.ManifestSHA256
	document.ManifestSHA256 = ""
	canonical, err := canonicalManifest(document)
	if err != nil || digest(canonical) != claimedManifestSHA {
		return errors.New("export verification failed: manifest checksum mismatch")
	}
	if err := verifier.verifyMembers(entries, document); err != nil {
		return err
	}
	return nil
}

// VerifyAndExtract validates the entire archive before writing into a newly
// created destination. A failed extraction removes every partial output.
func (verifier *Verifier) VerifyAndExtract(filename, expectedSHA256, destination string) (ExtractedPaths, error) {
	if destination == "" {
		return ExtractedPaths{}, errors.New("export extraction failed: destination is required")
	}
	archiveFile, archive, err := openVerifiedArchive(filename, expectedSHA256)
	if err != nil {
		return ExtractedPaths{}, err
	}
	defer archiveFile.Close()
	if err := verifier.verifyArchive(archive.File); err != nil {
		return ExtractedPaths{}, err
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		return ExtractedPaths{}, fmt.Errorf("export extraction failed: create exclusive destination: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()
	root, err := os.OpenRoot(destination)
	if err != nil {
		return ExtractedPaths{}, errors.New("export extraction failed: confine destination")
	}
	defer root.Close()
	for _, entry := range archive.File {
		if err := root.MkdirAll(path.Dir(entry.Name), 0o700); err != nil {
			return ExtractedPaths{}, fmt.Errorf("export extraction failed: create directory: %w", err)
		}
		reader, err := entry.Open()
		if err != nil {
			return ExtractedPaths{}, fmt.Errorf("export extraction failed: open member: %w", err)
		}
		output, err := root.OpenFile(entry.Name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			_ = reader.Close()
			return ExtractedPaths{}, fmt.Errorf("export extraction failed: create member: %w", err)
		}
		expectedSize, err := checkedExtractSize(entry.UncompressedSize64)
		if err != nil {
			_ = reader.Close()
			_ = output.Close()
			return ExtractedPaths{}, err
		}
		written, copyErr := io.Copy(output, io.LimitReader(reader, expectedSize+1))
		closeErr := errors.Join(reader.Close(), output.Close())
		if copyErr != nil || closeErr != nil || written != expectedSize {
			return ExtractedPaths{}, errors.New("export extraction failed: member content mismatch")
		}
	}
	absolute, err := filepath.Abs(destination)
	if err != nil {
		return ExtractedPaths{}, err
	}
	complete = true
	return ExtractedPaths{
		Destination: absolute, CaseBundle: filepath.Join(absolute, "case-bundle"),
		AuditRequest:     filepath.Join(absolute, "dispatch", "audit-request.json"),
		ScientificPolicy: filepath.Join(absolute, "dispatch", "scientific-policy.json"),
		ExecutionPolicy:  filepath.Join(absolute, "dispatch", "execution-policy.json"),
	}, nil
}

func checkedExtractSize(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, errors.New("export extraction failed: member size overrun")
	}
	return int64(value), nil
}

func indexArchive(files []*zip.File) (map[string]*zip.File, error) {
	entries := make(map[string]*zip.File, len(files))
	var declaredTotal uint64
	for _, entry := range files {
		name, err := safeArchivePath(entry.Name)
		if err != nil {
			return nil, err
		}
		if _, exists := entries[name]; exists {
			return nil, fmt.Errorf("export verification failed: duplicate normalized path %q", entry.Name)
		}
		if entry.Flags&1 != 0 || (entry.Method != zip.Store && entry.Method != zip.Deflate) {
			return nil, fmt.Errorf("export verification failed: unsupported or encrypted entry %q", entry.Name)
		}
		mode := entry.Mode()
		if !mode.IsRegular() || mode&os.ModeSymlink != 0 || mode&os.ModeType != 0 {
			return nil, fmt.Errorf("export verification failed: symlink or special entry %q", entry.Name)
		}
		if entry.UncompressedSize64 > math.MaxInt64 || entry.CompressedSize64 > math.MaxInt64 {
			return nil, fmt.Errorf("export verification failed: entry size overrun %q", entry.Name)
		}
		compressed := entry.CompressedSize64
		if compressed == 0 {
			compressed = 1
		}
		minimumCompressed := (entry.UncompressedSize64 + maxCompressionRatio - 1) / maxCompressionRatio
		if compressed < minimumCompressed {
			return nil, fmt.Errorf("export verification failed: compression ratio overrun %q", entry.Name)
		}
		if entry.UncompressedSize64 > uint64(maxArchiveBytes) || declaredTotal > uint64(maxArchiveBytes)-entry.UncompressedSize64 {
			return nil, errors.New("export verification failed: archive size overrun")
		}
		declaredTotal += entry.UncompressedSize64
		entries[name] = entry
	}
	return entries, nil
}

func safeArchivePath(name string) (string, error) {
	if name == "" || len(name) > 1024 || strings.ContainsAny(name, "\\\x00") || strings.HasPrefix(name, "/") || path.Clean(name) != name || name == "." || strings.HasPrefix(name, "../") {
		return "", fmt.Errorf("export verification failed: unsafe archive path %q", name)
	}
	return canonicalArchivePath(name), nil
}

func canonicalArchivePath(name string) string {
	return strings.ToLower(norm.NFC.String(name))
}

func (verifier *Verifier) verifyMembers(entries map[string]*zip.File, document manifest) error {
	readme, ok := entries["readme.md"]
	if !ok {
		return errors.New("export verification failed: README.md is missing")
	}
	readmeBytes, err := readBoundedEntry(readme, 64<<10)
	if err != nil || string(readmeBytes) != operatorReadme {
		return errors.New("export verification failed: README.md does not match the frozen operator instructions")
	}
	expected := map[string]manifestFile{}
	for _, file := range document.Files {
		normalized, err := safeArchivePath(file.Path)
		if err != nil || strings.HasPrefix(normalized, "case-bundle/") {
			return errors.New("export verification failed: manifest contains an unsafe member path")
		}
		archivePath := canonicalArchivePath("case-bundle/" + normalized)
		if _, duplicate := expected[archivePath]; duplicate {
			return errors.New("export verification failed: manifest contains duplicate normalized member paths")
		}
		expected[archivePath] = file
	}
	dispatchDocuments := map[string]func([]byte) error{
		"auditRequest":     verifier.validator.ValidateAuditRequest,
		"scientificPolicy": verifier.validator.ValidateScientificPolicy,
		"executionPolicy":  verifier.validator.ValidateExecutionPolicy,
	}
	allowed := map[string]bool{"readme.md": true, "case-bundle/case-manifest.json": true}
	for name, validation := range dispatchDocuments {
		binding, ok := document.Dispatch[name]
		if !ok {
			return fmt.Errorf("export verification failed: manifest dispatch binding %q is missing", name)
		}
		normalized, err := safeArchivePath(binding.Path)
		if err != nil {
			return err
		}
		entry, ok := entries[normalized]
		if !ok {
			return fmt.Errorf("export verification failed: dispatch member %q is missing", binding.Path)
		}
		contents, err := readBoundedEntry(entry, maxArchiveJSONBytes)
		if err != nil || digest(contents) != binding.SHA256 {
			return fmt.Errorf("export verification failed: dispatch checksum mismatch for %q", binding.Path)
		}
		if err := validation(contents); err != nil {
			return fmt.Errorf("export verification failed: dispatch member %q does not match schema version 1.0.0", binding.Path)
		}
		allowed[normalized] = true
	}
	for normalized, file := range expected {
		entry, ok := entries[normalized]
		if !ok {
			return fmt.Errorf("export verification failed: manifest member %q is missing", file.Path)
		}
		expectedSize, err := checkedArchiveSize(file.SizeBytes)
		if err != nil || entry.UncompressedSize64 != expectedSize {
			return fmt.Errorf("export verification failed: member size mismatch for %q", file.Path)
		}
		sha, size, err := hashEntry(entry, file.SizeBytes)
		if err != nil || size != file.SizeBytes || sha != file.SHA256 {
			return fmt.Errorf("export verification failed: member checksum mismatch for %q", file.Path)
		}
		allowed[normalized] = true
	}
	for name := range entries {
		if !allowed[name] {
			return fmt.Errorf("export verification failed: undeclared archive member %q", entries[name].Name)
		}
	}
	return nil
}

func readBoundedEntry(entry *zip.File, maximum int64) ([]byte, error) {
	maximumSize, err := checkedArchiveSize(maximum)
	if err != nil || entry.UncompressedSize64 > maximumSize {
		return nil, errors.New("entry exceeds size limit")
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	contents, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil || int64(len(contents)) > maximum || uint64(len(contents)) != entry.UncompressedSize64 {
		return nil, errors.New("entry content does not match declared size")
	}
	return contents, nil
}

func checkedArchiveSize(value int64) (uint64, error) {
	if value < 0 {
		return 0, errors.New("negative archive size")
	}
	return uint64(value), nil
}

func hashEntry(entry *zip.File, maximum int64) (string, int64, error) {
	reader, err := entry.Open()
	if err != nil {
		return "", 0, err
	}
	defer reader.Close()
	hasher := sha256.New()
	written, err := io.Copy(hasher, io.LimitReader(reader, maximum+1))
	if err != nil || written > maximum {
		return "", written, errors.New("entry content exceeds manifest size")
	}
	return hex.EncodeToString(hasher.Sum(nil)), written, nil
}
