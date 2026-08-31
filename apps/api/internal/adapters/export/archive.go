package export

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"time"

	jsoncanonicalizer "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

type Builder struct {
	storage   ports.PrivateObjectStore
	validator ports.IntakeValidator
	policy    ports.AdminPolicy
}

const operatorReadme = "# ClaimBounty manual handoff\n\nThis immutable archive is an input package for a local scientific-audit operator. Verify `case-bundle/CASE-MANIFEST.json`, then use the files under `case-bundle/` together with the three validated JSON documents under `dispatch/`. The hosted service has not run or dispatched the scientific routine.\n"

func NewBuilder(storage ports.PrivateObjectStore, validator ports.IntakeValidator, policy ports.AdminPolicy) (*Builder, error) {
	if storage == nil || validator == nil || policy == nil {
		return nil, errors.New("export: dependencies are required")
	}
	return &Builder{storage: storage, validator: validator, policy: policy}, nil
}

type manifest struct {
	SchemaVersion   string                  `json:"schemaVersion"`
	OrderID         string                  `json:"orderId"`
	ExportID        string                  `json:"exportId"`
	PublicReference string                  `json:"publicReference"`
	CreatedAt       string                  `json:"createdAt"`
	ManifestPath    string                  `json:"manifestPath"`
	ManifestSHA256  string                  `json:"manifestSha256,omitempty"`
	RoutineContract routineContract         `json:"routineContract"`
	PolicyVersions  map[string]string       `json:"policyVersions"`
	Authority       authority               `json:"authority"`
	Dispatch        map[string]dispatchFile `json:"dispatch"`
	Files           []manifestFile          `json:"files"`
}

type routineContract struct {
	RoutineID  string            `json:"routineId"`
	Revision   string            `json:"revision"`
	Validation routineValidation `json:"validation"`
}
type routineValidation struct {
	Status         string `json:"status"`
	ValidatedAt    string `json:"validatedAt"`
	EvidenceSHA256 string `json:"evidenceSha256"`
}
type authority struct {
	UploadsAuthorized                bool   `json:"uploadsAuthorized"`
	AnalysisUseAuthorized            bool   `json:"analysisUseAuthorized"`
	ExternalRedistributionAuthorized bool   `json:"externalRedistributionAuthorized"`
	AuthorizationPolicyVersion       string `json:"authorizationPolicyVersion"`
	AdminAllowlistVersion            string `json:"adminAllowlistVersion"`
}
type dispatchFile struct {
	Path          string `json:"path"`
	SchemaVersion string `json:"schemaVersion"`
	SHA256        string `json:"sha256"`
}
type manifestFile struct {
	FileID              string `json:"fileId"`
	Role                string `json:"role"`
	Path                string `json:"path"`
	OriginalDisplayName string `json:"originalDisplayName"`
	SizeBytes           int64  `json:"sizeBytes"`
	SHA256              string `json:"sha256"`
	ObjectVersion       string `json:"objectVersion"`
	StorageImmutability string `json:"storageImmutability"`
	DetectedMediaType   string `json:"detectedMediaType"`
	ScanResult          string `json:"scanResult"`
	ScannedAt           string `json:"scannedAt"`
}

func (builder *Builder) Build(ctx context.Context, order domain.Order, export domain.Export) (ports.ObjectMetadata, error) {
	if order.Intake == nil || len(order.Files) == 0 || export.StorageKey == "" {
		return ports.ObjectMetadata{}, errors.New("export: snapshot is incomplete")
	}
	files := append([]domain.OrderFile(nil), order.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].ID.String() < files[j].ID.String() })
	manifestFiles := make([]manifestFile, 0, len(files))
	paths := map[string]struct{}{
		canonicalArchivePath("README.md"):                       {},
		canonicalArchivePath("case-bundle/CASE-MANIFEST.json"):  {},
		canonicalArchivePath("dispatch/audit-request.json"):     {},
		canonicalArchivePath("dispatch/scientific-policy.json"): {},
		canonicalArchivePath("dispatch/execution-policy.json"):  {},
	}
	for _, file := range files {
		if file.Status != "clean" || file.ScannedAt == nil || file.ObjectGeneration == "" || file.DetectedMediaType == "" {
			return ports.ObjectMetadata{}, domain.ErrFileNotClean
		}
		archivePath := file.ArchivePath()
		canonicalPath := canonicalArchivePath(archivePath)
		if _, exists := paths[canonicalPath]; exists {
			return ports.ObjectMetadata{}, errors.New("export: archive path collision")
		}
		paths[canonicalPath] = struct{}{}
		manifestFiles = append(manifestFiles, manifestFile{FileID: file.ID.String(), Role: file.Role, Path: file.CaseBundlePath(), OriginalDisplayName: file.OriginalDisplayName, SizeBytes: file.SizeBytes, SHA256: file.SHA256, ObjectVersion: file.ObjectGeneration, StorageImmutability: "write_once", DetectedMediaType: file.DetectedMediaType, ScanResult: "clean", ScannedAt: formatTime(*file.ScannedAt)})
	}
	auth, err := manifestAuthority(order.Intake.AuditRequest, builder.policy.Version(), builder.policy.AllowlistVersion())
	if err != nil {
		return ports.ObjectMetadata{}, err
	}
	dispatch := map[string]dispatchFile{
		"auditRequest":     {Path: "dispatch/audit-request.json", SchemaVersion: "1.0.0", SHA256: digest(order.Intake.AuditRequest)},
		"scientificPolicy": {Path: "dispatch/scientific-policy.json", SchemaVersion: "1.0.0", SHA256: digest(order.Intake.ScientificPolicy)},
		"executionPolicy":  {Path: "dispatch/execution-policy.json", SchemaVersion: "1.0.0", SHA256: digest(order.Intake.ExecutionPolicy)},
	}
	resultManifest := manifest{SchemaVersion: "1.0.0", OrderID: order.ID.String(), ExportID: export.ID.String(), PublicReference: order.PublicReference, CreatedAt: formatTime(export.CreatedAt), ManifestPath: "case-bundle/CASE-MANIFEST.json", RoutineContract: routineContract{RoutineID: export.RoutineID, Revision: export.RoutineRevision, Validation: routineValidation{Status: "validated", ValidatedAt: formatTime(export.RoutineValidatedAt), EvidenceSHA256: export.RoutineEvidenceSHA}}, PolicyVersions: map[string]string{"auditRequest": "1.0.0", "scientificPolicy": "1.0.0", "executionPolicy": "1.0.0"}, Authority: auth, Dispatch: dispatch, Files: manifestFiles}
	canonical, err := canonicalManifest(resultManifest)
	if err != nil {
		return ports.ObjectMetadata{}, errors.New("export: manifest encoding failed")
	}
	resultManifest.ManifestSHA256 = digest(canonical)
	manifestBytes, err := canonicalManifest(resultManifest)
	if err != nil || builder.validator.ValidateCaseManifest(manifestBytes) != nil {
		return ports.ObjectMetadata{}, errors.New("export: generated manifest failed validation")
	}

	temporary, err := os.CreateTemp("", "claimbounty-export-*.zip")
	if err != nil {
		return ports.ObjectMetadata{}, errors.New("export: temporary archive creation failed")
	}
	name := temporary.Name()
	defer func() { _ = temporary.Close(); _ = os.Remove(name) }()
	archive := zip.NewWriter(temporary)
	if err := writeBytes(archive, "README.md", []byte(operatorReadme)); err != nil {
		return ports.ObjectMetadata{}, err
	}
	if err := writeBytes(archive, "case-bundle/CASE-MANIFEST.json", manifestBytes); err != nil {
		return ports.ObjectMetadata{}, err
	}
	if err := writeBytes(archive, "dispatch/audit-request.json", order.Intake.AuditRequest); err != nil {
		return ports.ObjectMetadata{}, err
	}
	if err := writeBytes(archive, "dispatch/scientific-policy.json", order.Intake.ScientificPolicy); err != nil {
		return ports.ObjectMetadata{}, err
	}
	if err := writeBytes(archive, "dispatch/execution-policy.json", order.Intake.ExecutionPolicy); err != nil {
		return ports.ObjectMetadata{}, err
	}
	for index, file := range files {
		if err := builder.writeObject(ctx, archive, files[index].ArchivePath(), file); err != nil {
			return ports.ObjectMetadata{}, err
		}
	}
	if err := archive.Close(); err != nil {
		return ports.ObjectMetadata{}, errors.New("export: archive finalization failed")
	}
	if err := temporary.Sync(); err != nil {
		return ports.ObjectMetadata{}, errors.New("export: archive sync failed")
	}
	stat, err := temporary.Stat()
	if err != nil {
		return ports.ObjectMetadata{}, errors.New("export: archive stat failed")
	}
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		return ports.ObjectMetadata{}, errors.New("export: archive rewind failed")
	}
	h := sha256.New()
	if _, err := io.Copy(h, temporary); err != nil {
		return ports.ObjectMetadata{}, errors.New("export: archive hash failed")
	}
	archiveSHA := hex.EncodeToString(h.Sum(nil))
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		return ports.ObjectMetadata{}, errors.New("export: archive rewind failed")
	}
	return builder.storage.PutWriteOnce(ctx, export.StorageKey, temporary, stat.Size(), "application/zip", archiveSHA)
}

func (builder *Builder) writeObject(ctx context.Context, archive *zip.Writer, archivePath string, file domain.OrderFile) error {
	reader, metadata, err := builder.storage.Open(ctx, file.StorageKey, file.ObjectGeneration)
	if err != nil {
		return err
	}
	defer reader.Close()
	if metadata.Generation != file.ObjectGeneration || metadata.SizeBytes != file.SizeBytes || metadata.SHA256 != file.SHA256 {
		return domain.ErrFileNotClean
	}
	w, err := createEntry(archive, archivePath)
	if err != nil {
		return err
	}
	h := sha256.New()
	written, err := io.CopyN(io.MultiWriter(w, h), reader, file.SizeBytes)
	if err != nil || written != file.SizeBytes || hex.EncodeToString(h.Sum(nil)) != file.SHA256 {
		return domain.ErrFileNotClean
	}
	var extra [1]byte
	if n, err := reader.Read(extra[:]); n != 0 || (err != nil && !errors.Is(err, io.EOF)) {
		return domain.ErrFileNotClean
	}
	return nil
}

func manifestAuthority(raw []byte, authorizationPolicyVersion, allowlistVersion string) (authority, error) {
	var document struct {
		Permissions struct {
			ExternalRedistributionAuthorized bool `json:"externalRedistributionAuthorized"`
		} `json:"permissions"`
		Authority struct {
			UploadsAuthorized          bool   `json:"uploadsAuthorized"`
			AuthorizationPolicyVersion string `json:"authorizationPolicyVersion"`
			AdminAllowlistVersion      string `json:"adminAllowlistVersion"`
		} `json:"authority"`
	}
	if err := json.Unmarshal(raw, &document); err != nil || !document.Authority.UploadsAuthorized || document.Authority.AdminAllowlistVersion != allowlistVersion || document.Authority.AuthorizationPolicyVersion != authorizationPolicyVersion {
		return authority{}, errors.New("export: authority is missing or stale")
	}
	return authority{UploadsAuthorized: true, AnalysisUseAuthorized: true, ExternalRedistributionAuthorized: document.Permissions.ExternalRedistributionAuthorized, AuthorizationPolicyVersion: document.Authority.AuthorizationPolicyVersion, AdminAllowlistVersion: document.Authority.AdminAllowlistVersion}, nil
}

func writeBytes(archive *zip.Writer, name string, contents []byte) error {
	w, err := createEntry(archive, name)
	if err != nil {
		return err
	}
	_, err = w.Write(contents)
	return err
}
func createEntry(archive *zip.Writer, name string) (io.Writer, error) {
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	header.SetMode(0o600)
	return archive.CreateHeader(header)
}
func digest(value []byte) string        { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func canonicalManifest(document manifest) ([]byte, error) {
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	return canonicalizeJSON(raw)
}

func canonicalizeJSON(raw []byte) ([]byte, error) { return jsoncanonicalizer.Transform(raw) }
