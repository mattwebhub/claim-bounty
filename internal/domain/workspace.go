package domain

import (
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	CurrentWorkspaceSchemaVersion uint32 = 1
	MaxWorkspaceObjects                  = 1000
	// MaxWorkspaceVersion is the largest integer that JSON clients can
	// represent exactly, keeping the ETag and response body consistent.
	MaxWorkspaceVersion          uint64 = 1<<53 - 1
	maxWorkspaceObjectIDRunes           = 100
	maxWorkspaceObjectKindRunes         = 50
	maxWorkspaceObjectLabelRunes        = 500
	maxWorkspaceCoordinate              = 10_000_000
	maxWorkspaceDimension               = 1_000_000
)

// WorkspaceObject is the business-neutral persisted representation of an
// item on the workspace. Selection, hover, tools, and undo history are local UI
// concerns and deliberately do not belong here.
type WorkspaceObject struct {
	id     string
	kind   string
	label  string
	x      float64
	y      float64
	width  float64
	height float64
}

func NewWorkspaceObject(
	id string,
	kind string,
	label string,
	x, y, width, height float64,
) (WorkspaceObject, error) {
	id = strings.TrimSpace(id)
	kind = strings.TrimSpace(kind)
	label = strings.TrimSpace(label)

	var issues []FieldIssue
	if id == "" {
		issues = append(issues, FieldIssue{Field: "objects[].id", Code: "required", Message: "must not be empty"})
	} else if !utf8.ValidString(id) || utf8.RuneCountInString(id) > maxWorkspaceObjectIDRunes || containsControl(id) {
		issues = append(issues, FieldIssue{Field: "objects[].id", Code: "invalid", Message: "must be valid UTF-8 with at most 100 characters and no control characters"})
	}
	if kind == "" {
		issues = append(issues, FieldIssue{Field: "objects[].kind", Code: "required", Message: "must not be empty"})
	} else if utf8.RuneCountInString(kind) > maxWorkspaceObjectKindRunes || !isSlug(kind) {
		issues = append(issues, FieldIssue{Field: "objects[].kind", Code: "invalid_format", Message: "must contain only lowercase letters, digits, and hyphens"})
	}
	if !utf8.ValidString(label) || utf8.RuneCountInString(label) > maxWorkspaceObjectLabelRunes || containsControl(label) {
		issues = append(issues, FieldIssue{Field: "objects[].label", Code: "invalid", Message: "must be valid UTF-8 with at most 500 characters and no control characters"})
	}
	if !isFiniteBounded(x, maxWorkspaceCoordinate) {
		issues = append(issues, FieldIssue{Field: "objects[].x", Code: "out_of_range", Message: "must be a finite workspace coordinate"})
	}
	if !isFiniteBounded(y, maxWorkspaceCoordinate) {
		issues = append(issues, FieldIssue{Field: "objects[].y", Code: "out_of_range", Message: "must be a finite workspace coordinate"})
	}
	if !isFiniteBounded(width, maxWorkspaceDimension) || width <= 0 {
		issues = append(issues, FieldIssue{Field: "objects[].width", Code: "out_of_range", Message: "must be positive and within the workspace limit"})
	}
	if !isFiniteBounded(height, maxWorkspaceDimension) || height <= 0 {
		issues = append(issues, FieldIssue{Field: "objects[].height", Code: "out_of_range", Message: "must be positive and within the workspace limit"})
	}
	if err := NewValidationError(issues...); err != nil {
		return WorkspaceObject{}, err
	}

	return WorkspaceObject{id: id, kind: kind, label: label, x: x, y: y, width: width, height: height}, nil
}

func (o WorkspaceObject) ID() string      { return o.id }
func (o WorkspaceObject) Kind() string    { return o.kind }
func (o WorkspaceObject) Label() string   { return o.label }
func (o WorkspaceObject) X() float64      { return o.x }
func (o WorkspaceObject) Y() float64      { return o.y }
func (o WorkspaceObject) Width() float64  { return o.width }
func (o WorkspaceObject) Height() float64 { return o.height }

type WorkspaceDocument struct {
	schemaVersion uint32
	objects       []WorkspaceObject
}

func EmptyWorkspaceDocument() WorkspaceDocument {
	return WorkspaceDocument{schemaVersion: CurrentWorkspaceSchemaVersion}
}

func NewWorkspaceDocument(schemaVersion uint32, objects []WorkspaceObject) (WorkspaceDocument, error) {
	var issues []FieldIssue
	if schemaVersion != CurrentWorkspaceSchemaVersion {
		issues = append(issues, FieldIssue{Field: "schemaVersion", Code: "unsupported", Message: "must use the current workspace schema version"})
	}
	if len(objects) > MaxWorkspaceObjects {
		issues = append(issues, FieldIssue{Field: "objects", Code: "too_many", Message: "must not contain more than 1000 objects"})
	}

	seen := make(map[string]struct{}, len(objects))
	for _, object := range objects {
		if object.id == "" || object.kind == "" {
			issues = append(issues, FieldIssue{Field: "objects", Code: "invalid_object", Message: "must contain only constructed workspace objects"})
			continue
		}
		if _, exists := seen[object.id]; exists {
			issues = append(issues, FieldIssue{Field: "objects", Code: "duplicate_id", Message: "object IDs must be unique"})
		}
		seen[object.id] = struct{}{}
	}
	if err := NewValidationError(issues...); err != nil {
		return WorkspaceDocument{}, err
	}

	return WorkspaceDocument{
		schemaVersion: schemaVersion,
		objects:       append([]WorkspaceObject(nil), objects...),
	}, nil
}

func (d WorkspaceDocument) SchemaVersion() uint32 { return d.schemaVersion }

func (d WorkspaceDocument) Objects() []WorkspaceObject {
	return append([]WorkspaceObject(nil), d.objects...)
}

type Workspace struct {
	projectID ProjectID
	document  WorkspaceDocument
	version   uint64
	createdAt time.Time
	updatedAt time.Time
}

func NewWorkspace(projectID ProjectID, document WorkspaceDocument, now time.Time) (Workspace, error) {
	return RestoreWorkspace(projectID, document, 1, now, now)
}

func RestoreWorkspace(
	projectID ProjectID,
	document WorkspaceDocument,
	version uint64,
	createdAt, updatedAt time.Time,
) (Workspace, error) {
	var issues []FieldIssue
	if !isUUID(projectID.String()) {
		issues = append(issues, FieldIssue{Field: "projectId", Code: "invalid_format", Message: "must be a UUID"})
	}
	if _, err := NewWorkspaceDocument(document.schemaVersion, document.objects); err != nil {
		var validation *ValidationError
		if errors.As(err, &validation) {
			issues = append(issues, validation.Issues()...)
		}
	}
	if version == 0 {
		issues = append(issues, FieldIssue{Field: "version", Code: "required", Message: "must be greater than zero"})
	} else if version > MaxWorkspaceVersion {
		issues = append(issues, FieldIssue{Field: "version", Code: "out_of_range", Message: "must be a JSON-safe integer"})
	}

	createdAt = createdAt.UTC()
	updatedAt = updatedAt.UTC()
	if createdAt.IsZero() {
		issues = append(issues, FieldIssue{Field: "createdAt", Code: "required", Message: "must not be zero"})
	}
	if updatedAt.IsZero() {
		issues = append(issues, FieldIssue{Field: "updatedAt", Code: "required", Message: "must not be zero"})
	}
	if !createdAt.IsZero() && !updatedAt.IsZero() && updatedAt.Before(createdAt) {
		issues = append(issues, FieldIssue{Field: "updatedAt", Code: "before_created_at", Message: "must not be before createdAt"})
	}
	if err := NewValidationError(issues...); err != nil {
		return Workspace{}, err
	}

	return Workspace{
		projectID: projectID,
		document: WorkspaceDocument{
			schemaVersion: document.schemaVersion,
			objects:       append([]WorkspaceObject(nil), document.objects...),
		},
		version:   version,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

// ReplaceDocument applies a complete client snapshot after checking the
// version it was based on. Persistence must repeat the check atomically.
func (w Workspace) ReplaceDocument(
	document WorkspaceDocument,
	expectedVersion uint64,
	now time.Time,
) (Workspace, error) {
	if expectedVersion == 0 || expectedVersion > MaxWorkspaceVersion {
		return Workspace{}, NewValidationError(FieldIssue{
			Field: "expectedVersion", Code: "out_of_range", Message: "must be a positive JSON-safe integer",
		})
	}
	if expectedVersion != w.version {
		return Workspace{}, NewVersionConflictError(expectedVersion, w.version)
	}
	if w.version == MaxWorkspaceVersion {
		return Workspace{}, NewVersionConflictError(expectedVersion, w.version)
	}
	return RestoreWorkspace(w.projectID, document, w.version+1, w.createdAt, now)
}

func (w Workspace) ProjectID() ProjectID { return w.projectID }
func (w Workspace) Version() uint64      { return w.version }
func (w Workspace) CreatedAt() time.Time { return w.createdAt }
func (w Workspace) UpdatedAt() time.Time { return w.updatedAt }

func (w Workspace) Document() WorkspaceDocument {
	return WorkspaceDocument{
		schemaVersion: w.document.schemaVersion,
		objects:       append([]WorkspaceObject(nil), w.document.objects...),
	}
}

func isSlug(value string) bool {
	for index, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			continue
		}
		if character == '-' && index > 0 && index < len(value)-1 {
			continue
		}
		return false
	}
	return value != ""
}

func isFiniteBounded(value, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= maximum
}
