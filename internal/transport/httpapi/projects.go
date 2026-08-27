package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mattwebhub/micro1-go-template/internal/domain"
	"github.com/mattwebhub/micro1-go-template/internal/services"
	"github.com/mattwebhub/micro1-go-template/internal/transport/httpapi/middleware"
	"github.com/mattwebhub/micro1-go-template/internal/transport/httpapi/response"
)

type ProjectCreator interface {
	CreateProject(context.Context, services.CreateProjectCommand) (services.CreateProjectResult, error)
}

type ProjectReader interface {
	GetProject(context.Context, services.GetProjectQuery) (services.GetProjectResult, error)
	ListProjects(context.Context, services.ListProjectsQuery) (services.ListProjectsResult, error)
}

type WorkspaceReaderWriter interface {
	GetWorkspace(context.Context, services.GetWorkspaceQuery) (services.GetWorkspaceResult, error)
	SaveWorkspace(context.Context, services.SaveWorkspaceCommand) (services.SaveWorkspaceResult, error)
}

type ProjectRoutes struct {
	creator      ProjectCreator
	projects     ProjectReader
	workspaces   WorkspaceReaderWriter
	logger       *slog.Logger
	maxBodyBytes int64
}

func NewProjectRoutes(creator ProjectCreator, projects ProjectReader, workspaces WorkspaceReaderWriter, logger *slog.Logger, maxBodyBytes int64) (*ProjectRoutes, error) {
	if creator == nil || projects == nil || workspaces == nil {
		return nil, errors.New("httpapi: project route dependencies are required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ProjectRoutes{creator: creator, projects: projects, workspaces: workspaces, logger: logger, maxBodyBytes: maxBodyBytes}, nil
}

func (routes *ProjectRoutes) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/projects", routes.createProject)
	mux.HandleFunc("GET /api/v1/projects", routes.listProjects)
	mux.HandleFunc("GET /api/v1/projects/{projectId}", routes.getProject)
	mux.HandleFunc("GET /api/v1/projects/{projectId}/workspace", routes.getWorkspace)
	mux.HandleFunc("PUT /api/v1/projects/{projectId}/workspace", routes.saveWorkspace)
}

type createProjectRequest struct {
	Name *string `json:"name"`
}

func (routes *ProjectRoutes) createProject(w http.ResponseWriter, r *http.Request) {
	var body createProjectRequest
	if err := response.DecodeJSON(w, r, &body, routes.maxBodyBytes); err != nil {
		routes.writeError(w, r, err)
		return
	}
	if body.Name == nil {
		routes.writeError(w, r, domain.NewValidationError(domain.FieldIssue{
			Field: "name", Code: "required", Message: "must be provided",
		}))
		return
	}
	result, err := routes.creator.CreateProject(r.Context(), services.CreateProjectCommand{Name: *body.Name})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/projects/"+result.Project.ID().String())
	_ = response.WriteData(w, http.StatusCreated, projectDTO(result.Project))
}

func (routes *ProjectRoutes) listProjects(w http.ResponseWriter, r *http.Request) {
	query, parseErr := url.ParseQuery(r.URL.RawQuery)
	if parseErr != nil || len(query["limit"]) > 1 || len(query["cursor"]) > 1 {
		routes.writeError(w, r, &response.ClientError{
			Status: http.StatusBadRequest, Code: "invalid_query", Message: "query parameters must be well formed and occur at most once", Cause: parseErr,
		})
		return
	}
	var limit uint64
	var err error
	if raw := query.Get("limit"); raw != "" {
		limit, err = strconv.ParseUint(raw, 10, 32)
		if err != nil || limit == 0 || limit > 100 {
			routes.writeError(w, r, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "limit must be an integer between 1 and 100", Cause: err})
			return
		}
	}
	cursor := query.Get("cursor")
	if len(cursor) > 1024 {
		routes.writeError(w, r, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "cursor must be at most 1024 bytes"})
		return
	}
	result, err := routes.projects.ListProjects(r.Context(), services.ListProjectsQuery{Limit: uint32(limit), Cursor: cursor})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	items := make([]projectResponse, 0, len(result.Projects))
	for _, project := range result.Projects {
		items = append(items, projectDTO(project))
	}
	_ = response.WriteData(w, http.StatusOK, struct {
		Items      []projectResponse `json:"items"`
		NextCursor string            `json:"nextCursor,omitempty"`
	}{Items: items, NextCursor: result.NextCursor})
}

func (routes *ProjectRoutes) getProject(w http.ResponseWriter, r *http.Request) {
	id, ok := routes.projectID(w, r)
	if !ok {
		return
	}
	result, err := routes.projects.GetProject(r.Context(), services.GetProjectQuery{ProjectID: id})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	_ = response.WriteData(w, http.StatusOK, projectDTO(result.Project))
}

func (routes *ProjectRoutes) getWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := routes.projectID(w, r)
	if !ok {
		return
	}
	result, err := routes.workspaces.GetWorkspace(r.Context(), services.GetWorkspaceQuery{ProjectID: id})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("ETag", versionETag(result.Workspace.Version()))
	_ = response.WriteData(w, http.StatusOK, workspaceDTO(result.Workspace))
}

type saveWorkspaceRequest struct {
	Document *workspaceDocumentRequest `json:"document"`
}

type workspaceDocumentRequest struct {
	SchemaVersion *uint32                   `json:"schemaVersion"`
	Objects       *[]workspaceObjectRequest `json:"objects"`
}

type workspaceObjectRequest struct {
	ID     *string  `json:"id"`
	Kind   *string  `json:"kind"`
	Label  *string  `json:"label"`
	X      *float64 `json:"x"`
	Y      *float64 `json:"y"`
	Width  *float64 `json:"width"`
	Height *float64 `json:"height"`
}

func (routes *ProjectRoutes) saveWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := routes.projectID(w, r)
	if !ok {
		return
	}
	expectedVersion, err := parseIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		routes.writeError(w, r, &response.ClientError{Status: http.StatusPreconditionRequired, Code: "if_match_required", Message: "If-Match must contain the current quoted workspace version", Cause: err})
		return
	}
	var body saveWorkspaceRequest
	if err := response.DecodeJSON(w, r, &body, routes.maxBodyBytes); err != nil {
		routes.writeError(w, r, err)
		return
	}
	document, err := domainDocument(body.Document)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	result, err := routes.workspaces.SaveWorkspace(r.Context(), services.SaveWorkspaceCommand{
		ProjectID: id, ExpectedVersion: expectedVersion, Document: document,
	})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("ETag", versionETag(result.Workspace.Version()))
	_ = response.WriteData(w, http.StatusOK, workspaceDTO(result.Workspace))
}

func (routes *ProjectRoutes) projectID(w http.ResponseWriter, r *http.Request) (domain.ProjectID, bool) {
	id, err := domain.NewProjectID(r.PathValue("projectId"))
	if err != nil {
		routes.writeError(w, r, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_path_parameter", Message: "projectId must be a UUID", Cause: err})
		return domain.ProjectID{}, false
	}
	return id, true
}

func parseIfMatch(value string) (uint64, error) {
	if value == "" || strings.HasPrefix(value, "W/") {
		return 0, errors.New("missing or weak entity tag")
	}
	unquoted, err := strconv.Unquote(value)
	if err != nil || unquoted == "" {
		return 0, errors.New("malformed entity tag")
	}
	version, err := strconv.ParseUint(unquoted, 10, 64)
	if err != nil || version == 0 || version > domain.MaxWorkspaceVersion {
		return 0, errors.New("invalid workspace version")
	}
	return version, nil
}

func versionETag(version uint64) string { return fmt.Sprintf("\"%d\"", version) }

type projectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func projectDTO(project domain.Project) projectResponse {
	return projectResponse{ID: project.ID().String(), Name: project.Name(), CreatedAt: project.CreatedAt().Format("2006-01-02T15:04:05.000000000Z07:00"), UpdatedAt: project.UpdatedAt().Format("2006-01-02T15:04:05.000000000Z07:00")}
}

type workspaceResponse struct {
	ProjectID string                    `json:"projectId"`
	Document  workspaceDocumentResponse `json:"document"`
	Version   uint64                    `json:"version"`
	CreatedAt string                    `json:"createdAt"`
	UpdatedAt string                    `json:"updatedAt"`
}

type workspaceDocumentResponse struct {
	SchemaVersion uint32                    `json:"schemaVersion"`
	Objects       []workspaceObjectResponse `json:"objects"`
}

type workspaceObjectResponse struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	Label  string  `json:"label"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func workspaceDTO(workspace domain.Workspace) workspaceResponse {
	document := workspace.Document()
	objects := make([]workspaceObjectResponse, 0, len(document.Objects()))
	for _, object := range document.Objects() {
		objects = append(objects, workspaceObjectResponse{ID: object.ID(), Kind: object.Kind(), Label: object.Label(), X: object.X(), Y: object.Y(), Width: object.Width(), Height: object.Height()})
	}
	return workspaceResponse{ProjectID: workspace.ProjectID().String(), Document: workspaceDocumentResponse{SchemaVersion: document.SchemaVersion(), Objects: objects}, Version: workspace.Version(), CreatedAt: workspace.CreatedAt().Format("2006-01-02T15:04:05.000000000Z07:00"), UpdatedAt: workspace.UpdatedAt().Format("2006-01-02T15:04:05.000000000Z07:00")}
}

func domainDocument(document *workspaceDocumentRequest) (domain.WorkspaceDocument, error) {
	if document == nil {
		return domain.WorkspaceDocument{}, domain.NewValidationError(domain.FieldIssue{Field: "document", Code: "required", Message: "must be provided"})
	}
	var issues []domain.FieldIssue
	if document.SchemaVersion == nil {
		issues = append(issues, domain.FieldIssue{Field: "document.schemaVersion", Code: "required", Message: "must be provided"})
	}
	if document.Objects == nil {
		issues = append(issues, domain.FieldIssue{Field: "document.objects", Code: "required", Message: "must be provided"})
	}
	if err := domain.NewValidationError(issues...); err != nil {
		return domain.WorkspaceDocument{}, err
	}
	objects := make([]domain.WorkspaceObject, 0, len(*document.Objects))
	for index, item := range *document.Objects {
		path := fmt.Sprintf("document.objects[%d]", index)
		if item.ID == nil {
			issues = append(issues, domain.FieldIssue{Field: path + ".id", Code: "required", Message: "must be provided"})
		}
		if item.Kind == nil {
			issues = append(issues, domain.FieldIssue{Field: path + ".kind", Code: "required", Message: "must be provided"})
		}
		if item.Label == nil {
			issues = append(issues, domain.FieldIssue{Field: path + ".label", Code: "required", Message: "must be provided"})
		}
		if item.X == nil {
			issues = append(issues, domain.FieldIssue{Field: path + ".x", Code: "required", Message: "must be provided"})
		}
		if item.Y == nil {
			issues = append(issues, domain.FieldIssue{Field: path + ".y", Code: "required", Message: "must be provided"})
		}
		if item.Width == nil {
			issues = append(issues, domain.FieldIssue{Field: path + ".width", Code: "required", Message: "must be provided"})
		}
		if item.Height == nil {
			issues = append(issues, domain.FieldIssue{Field: path + ".height", Code: "required", Message: "must be provided"})
		}
	}
	if err := domain.NewValidationError(issues...); err != nil {
		return domain.WorkspaceDocument{}, err
	}
	for index, item := range *document.Objects {
		object, err := domain.NewWorkspaceObject(*item.ID, *item.Kind, *item.Label, *item.X, *item.Y, *item.Width, *item.Height)
		if err != nil {
			return domain.WorkspaceDocument{}, prefixValidationPath(err, fmt.Sprintf("document.objects[%d]", index))
		}
		objects = append(objects, object)
	}
	domainDocument, err := domain.NewWorkspaceDocument(*document.SchemaVersion, objects)
	if err != nil {
		return domain.WorkspaceDocument{}, prefixValidationPath(err, "document")
	}
	return domainDocument, nil
}

func prefixValidationPath(err error, prefix string) error {
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		return err
	}
	issues := validation.Issues()
	for index := range issues {
		field := strings.TrimPrefix(issues[index].Field, "objects[]")
		if !strings.HasPrefix(field, ".") {
			field = "." + field
		}
		issues[index].Field = prefix + field
	}
	return domain.NewValidationError(issues...)
}

func (routes *ProjectRoutes) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message, details := mapError(err)
	requestID := middleware.RequestIDFromContext(r.Context())
	if status >= 500 {
		routes.logger.ErrorContext(r.Context(), "HTTP request failed", "error", err, "request_id", requestID)
	}
	_ = response.WriteError(w, status, code, message, requestID, details)
}

func mapError(err error) (int, string, string, []response.FieldIssue) {
	var clientError *response.ClientError
	if errors.As(err, &clientError) {
		return clientError.Status, clientError.Code, clientError.Message, nil
	}
	var validation *domain.ValidationError
	if errors.As(err, &validation) {
		details := make([]response.FieldIssue, 0, len(validation.Issues()))
		for _, issue := range validation.Issues() {
			details = append(details, response.FieldIssue{Path: issue.Field, Code: issue.Code, Message: issue.Message})
		}
		return http.StatusUnprocessableEntity, "validation_failed", "request validation failed", details
	}
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		return http.StatusNotFound, "project_not_found", "project not found", nil
	case errors.Is(err, domain.ErrWorkspaceNotFound):
		return http.StatusNotFound, "workspace_not_found", "workspace not found", nil
	case errors.Is(err, domain.ErrProjectExists):
		return http.StatusConflict, "project_exists", "project already exists", nil
	case errors.Is(err, domain.ErrVersionConflict):
		return http.StatusConflict, "version_conflict", "workspace was changed by another client", nil
	default:
		return http.StatusInternalServerError, "internal_error", "an unexpected error occurred", nil
	}
}
