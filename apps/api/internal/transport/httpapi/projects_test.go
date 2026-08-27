package httpapi

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
)

const testProjectID = "0190cafe-7a4f-7000-8000-000000000001"

type projectCreatorStub func(context.Context, services.CreateProjectCommand) (services.CreateProjectResult, error)

func (stub projectCreatorStub) CreateProject(ctx context.Context, command services.CreateProjectCommand) (services.CreateProjectResult, error) {
	return stub(ctx, command)
}

type projectReaderStub struct {
	get  func(context.Context, services.GetProjectQuery) (services.GetProjectResult, error)
	list func(context.Context, services.ListProjectsQuery) (services.ListProjectsResult, error)
}

func (stub projectReaderStub) GetProject(ctx context.Context, query services.GetProjectQuery) (services.GetProjectResult, error) {
	return stub.get(ctx, query)
}
func (stub projectReaderStub) ListProjects(ctx context.Context, query services.ListProjectsQuery) (services.ListProjectsResult, error) {
	return stub.list(ctx, query)
}

type workspaceStub struct {
	get  func(context.Context, services.GetWorkspaceQuery) (services.GetWorkspaceResult, error)
	save func(context.Context, services.SaveWorkspaceCommand) (services.SaveWorkspaceResult, error)
}

func (stub workspaceStub) GetWorkspace(ctx context.Context, query services.GetWorkspaceQuery) (services.GetWorkspaceResult, error) {
	return stub.get(ctx, query)
}
func (stub workspaceStub) SaveWorkspace(ctx context.Context, command services.SaveWorkspaceCommand) (services.SaveWorkspaceResult, error) {
	return stub.save(ctx, command)
}

func TestProjectRoutesCreateAndListContract(t *testing.T) {
	t.Parallel()
	project, workspace := testEntities(t)
	creator := projectCreatorStub(func(_ context.Context, command services.CreateProjectCommand) (services.CreateProjectResult, error) {
		if command.Name != "Hackathon" {
			t.Fatalf("name = %q", command.Name)
		}
		return services.CreateProjectResult{Project: project}, nil
	})
	reader := projectReaderStub{
		get: func(context.Context, services.GetProjectQuery) (services.GetProjectResult, error) {
			return services.GetProjectResult{Project: project}, nil
		},
		list: func(_ context.Context, query services.ListProjectsQuery) (services.ListProjectsResult, error) {
			if query.Limit != 2 {
				t.Fatalf("limit = %d", query.Limit)
			}
			return services.ListProjectsResult{Projects: []domain.Project{project}, NextCursor: "next"}, nil
		},
	}
	workspaces := successfulWorkspaceStub(workspace)
	handler := testProjectHandler(t, creator, reader, workspaces, workspaces)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"Hackathon"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Location") != "/api/v1/projects/"+testProjectID {
		t.Fatalf("Location = %q", recorder.Header().Get("Location"))
	}
	if !strings.Contains(recorder.Body.String(), `"data":{"id":"`+testProjectID+`"`) || strings.Contains(recorder.Body.String(), "workspace") {
		t.Fatalf("create body = %s", recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/projects?limit=2", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"data":{"items":[`) || !strings.Contains(recorder.Body.String(), `"nextCursor":"next"`) {
		t.Fatalf("list body = %s", recorder.Body.String())
	}
}

func TestWorkspaceRoutesUseStrongVersionETag(t *testing.T) {
	t.Parallel()
	project, workspace := testEntities(t)
	updated, err := workspace.ReplaceDocument(workspace.Document(), 1, workspace.UpdatedAt().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	stub := workspaceStub{
		get: func(context.Context, services.GetWorkspaceQuery) (services.GetWorkspaceResult, error) {
			return services.GetWorkspaceResult{Workspace: workspace}, nil
		},
		save: func(_ context.Context, command services.SaveWorkspaceCommand) (services.SaveWorkspaceResult, error) {
			if command.ExpectedVersion != 1 {
				t.Fatalf("expected version = %d", command.ExpectedVersion)
			}
			return services.SaveWorkspaceResult{Workspace: updated}, nil
		},
	}
	handler := testProjectHandler(t,
		projectCreatorStub(func(context.Context, services.CreateProjectCommand) (services.CreateProjectResult, error) {
			return services.CreateProjectResult{}, nil
		}),
		projectReaderStub{get: func(context.Context, services.GetProjectQuery) (services.GetProjectResult, error) {
			return services.GetProjectResult{Project: project}, nil
		}, list: func(context.Context, services.ListProjectsQuery) (services.ListProjectsResult, error) {
			return services.ListProjectsResult{}, nil
		}},
		stub,
		stub,
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/workspace", nil))
	if recorder.Code != http.StatusOK || recorder.Header().Get("ETag") != `"1"` {
		t.Fatalf("GET status/etag = %d/%q", recorder.Code, recorder.Header().Get("ETag"))
	}

	body := []byte(`{"document":{"schemaVersion":1,"objects":[]}}`)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+testProjectID+"/workspace", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"1"`)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("ETag") != `"2"` {
		t.Fatalf("PUT status/etag = %d/%q body=%s", recorder.Code, recorder.Header().Get("ETag"), recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+testProjectID+"/workspace", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusPreconditionRequired || !strings.Contains(recorder.Body.String(), `"code":"if_match_required"`) {
		t.Fatalf("missing If-Match response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestWorkspaceSaveRejectsOmittedZeroValueFields(t *testing.T) {
	t.Parallel()
	project, workspace := testEntities(t)
	saveCalled := false
	workspaces := workspaceStub{
		get: func(context.Context, services.GetWorkspaceQuery) (services.GetWorkspaceResult, error) {
			return services.GetWorkspaceResult{Workspace: workspace}, nil
		},
		save: func(context.Context, services.SaveWorkspaceCommand) (services.SaveWorkspaceResult, error) {
			saveCalled = true
			return services.SaveWorkspaceResult{}, nil
		},
	}
	handler := testProjectHandler(t,
		projectCreatorStub(func(context.Context, services.CreateProjectCommand) (services.CreateProjectResult, error) {
			return services.CreateProjectResult{}, nil
		}),
		projectReaderStub{get: func(context.Context, services.GetProjectQuery) (services.GetProjectResult, error) {
			return services.GetProjectResult{Project: project}, nil
		}, list: func(context.Context, services.ListProjectsQuery) (services.ListProjectsResult, error) {
			return services.ListProjectsResult{}, nil
		}},
		workspaces,
		workspaces,
	)

	// label, x, and y have valid Go zero values but are required by the wire
	// contract, so pointer request fields must distinguish omission.
	body := `{"document":{"schemaVersion":1,"objects":[{"id":"one","kind":"note","width":10,"height":10}]}}`
	request := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+testProjectID+"/workspace", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"1"`)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity || saveCalled {
		t.Fatalf("response = %d %s, saveCalled = %v", recorder.Code, recorder.Body.String(), saveCalled)
	}
	for _, path := range []string{"document.objects[0].label", "document.objects[0].x", "document.objects[0].y"} {
		if !strings.Contains(recorder.Body.String(), `"path":"`+path+`"`) {
			t.Errorf("response omitted field path %q: %s", path, recorder.Body.String())
		}
	}
}

func TestListProjectsRejectsAmbiguousOrOutOfContractQuery(t *testing.T) {
	t.Parallel()
	project, workspace := testEntities(t)
	listCalled := false
	handler := testProjectHandler(t,
		projectCreatorStub(func(context.Context, services.CreateProjectCommand) (services.CreateProjectResult, error) {
			return services.CreateProjectResult{}, nil
		}),
		projectReaderStub{get: func(context.Context, services.GetProjectQuery) (services.GetProjectResult, error) {
			return services.GetProjectResult{Project: project}, nil
		}, list: func(context.Context, services.ListProjectsQuery) (services.ListProjectsResult, error) {
			listCalled = true
			return services.ListProjectsResult{}, nil
		}},
		successfulWorkspaceStub(workspace),
		successfulWorkspaceStub(workspace),
	)

	for _, rawQuery := range []string{"limit=1&limit=2", "limit=101"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/projects?"+rawQuery, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"invalid_query"`) {
			t.Errorf("query %q = %d %s", rawQuery, recorder.Code, recorder.Body.String())
		}
	}
	if listCalled {
		t.Fatal("invalid query reached project service")
	}
}

func TestProjectRoutesMapValidationDetailsAndHideInternalErrors(t *testing.T) {
	t.Parallel()
	_, workspace := testEntities(t)
	validation := domain.NewValidationError(domain.FieldIssue{Field: "name", Code: "required", Message: "must not be empty"})
	creator := projectCreatorStub(func(context.Context, services.CreateProjectCommand) (services.CreateProjectResult, error) {
		return services.CreateProjectResult{}, validation
	})
	reader := projectReaderStub{get: func(context.Context, services.GetProjectQuery) (services.GetProjectResult, error) {
		return services.GetProjectResult{}, io.ErrUnexpectedEOF
	}, list: func(context.Context, services.ListProjectsQuery) (services.ListProjectsResult, error) {
		return services.ListProjectsResult{}, nil
	}}
	workspaces := successfulWorkspaceStub(workspace)
	handler := testProjectHandler(t, creator, reader, workspaces, workspaces)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":""}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), `"details":[{"path":"name","code":"required","message":"must not be empty"}]`) {
		t.Fatalf("validation = %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID, nil))
	if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "EOF") {
		t.Fatalf("internal = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestMapErrorCoversEveryPublicDomainCategory(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"project not found", domain.ErrProjectNotFound, http.StatusNotFound, "project_not_found"},
		{"workspace not found", domain.ErrWorkspaceNotFound, http.StatusNotFound, "workspace_not_found"},
		{"project exists", domain.ErrProjectExists, http.StatusConflict, "project_exists"},
		{"version conflict", domain.NewVersionConflictError(1, 2), http.StatusConflict, "version_conflict"},
		{"validation", domain.NewValidationError(domain.FieldIssue{Field: "name", Code: "required", Message: "required"}), http.StatusUnprocessableEntity, "validation_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			status, code, _, _ := mapError(test.err)
			if status != test.status || code != test.code {
				t.Fatalf("mapError() = %d/%q, want %d/%q", status, code, test.status, test.code)
			}
		})
	}
}

func testProjectHandler(
	t *testing.T,
	creator ProjectCreator,
	reader ProjectReader,
	workspaces WorkspaceReader,
	workspaceSaver WorkspaceSaver,
) http.Handler {
	t.Helper()
	routes, err := NewProjectRoutes(creator, reader, workspaces, workspaceSaver, slog.New(slog.DiscardHandler), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux)
	return mux
}

func testEntities(t *testing.T) (domain.Project, domain.Workspace) {
	t.Helper()
	id, err := domain.NewProjectID(testProjectID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject(id, "Hackathon", now)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := domain.NewWorkspace(id, domain.EmptyWorkspaceDocument(), now)
	if err != nil {
		t.Fatal(err)
	}
	return project, workspace
}

func successfulWorkspaceStub(workspace domain.Workspace) workspaceStub {
	return workspaceStub{get: func(context.Context, services.GetWorkspaceQuery) (services.GetWorkspaceResult, error) {
		return services.GetWorkspaceResult{Workspace: workspace}, nil
	}, save: func(context.Context, services.SaveWorkspaceCommand) (services.SaveWorkspaceResult, error) {
		return services.SaveWorkspaceResult{Workspace: workspace}, nil
	}}
}
