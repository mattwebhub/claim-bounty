package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/postgres/sqlc"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

type documentRecord struct {
	SchemaVersion uint32         `json:"schemaVersion"`
	Objects       []objectRecord `json:"objects"`
}

type objectRecord struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	Label  string  `json:"label"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func marshalDocument(document domain.WorkspaceDocument) ([]byte, error) {
	record := documentRecord{SchemaVersion: document.SchemaVersion(), Objects: make([]objectRecord, 0, len(document.Objects()))}
	for _, object := range document.Objects() {
		record.Objects = append(record.Objects, objectRecord{
			ID: object.ID(), Kind: object.Kind(), Label: object.Label(), X: object.X(), Y: object.Y(), Width: object.Width(), Height: object.Height(),
		})
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("postgres: marshal workspace document: %w", err)
	}
	return encoded, nil
}

func workspaceFromRow(row sqlc.Workspace) (domain.Workspace, error) {
	id, err := domain.NewProjectID(uuidString(row.ProjectID))
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("postgres: restore workspace project ID: %w", err)
	}
	var record documentRecord
	if err := json.Unmarshal(row.Document, &record); err != nil {
		return domain.Workspace{}, fmt.Errorf("postgres: decode workspace document: %w", err)
	}
	objects := make([]domain.WorkspaceObject, 0, len(record.Objects))
	for _, item := range record.Objects {
		object, err := domain.NewWorkspaceObject(item.ID, item.Kind, item.Label, item.X, item.Y, item.Width, item.Height)
		if err != nil {
			return domain.Workspace{}, fmt.Errorf("postgres: restore workspace object: %w", err)
		}
		objects = append(objects, object)
	}
	document, err := domain.NewWorkspaceDocument(record.SchemaVersion, objects)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("postgres: restore workspace document: %w", err)
	}
	version, err := checkedVersion(row.Version)
	if err != nil {
		return domain.Workspace{}, err
	}
	workspace, err := domain.RestoreWorkspace(id, document, version, row.CreatedAt.Time, row.UpdatedAt.Time)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("postgres: restore workspace: %w", err)
	}
	return workspace, nil
}
