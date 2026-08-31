package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi/response"
)

type AdministrationActions interface {
	ListOrders(context.Context, domain.Session, services.OrderPageRequest) (services.OrderPage, error)
	GetOrder(context.Context, domain.Session, domain.Identifier) (domain.Order, []domain.Export, error)
	UpdateIntake(context.Context, services.AdminIntakeCommand) (domain.Order, []domain.Export, error)
	CreateExport(context.Context, services.CreateExportCommand) (domain.Export, error)
	GetExport(context.Context, domain.Session, domain.Identifier, domain.Identifier) (domain.Export, error)
	OpenFile(context.Context, domain.Session, domain.Identifier, domain.Identifier) (services.ObjectReader, domain.OrderFile, error)
	OpenExport(context.Context, domain.Session, domain.Identifier) (services.ObjectReader, domain.Export, error)
}

func (routes *ClaimRoutes) listAdminOrders(w http.ResponseWriter, r *http.Request) {
	session, err := routes.readSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	request, err := parseOrderPage(r.URL)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	page, err := routes.administration.ListOrders(r.Context(), session, request)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	items := make([]any, 0, len(page.Orders))
	for _, order := range page.Orders {
		items = append(items, orderDTO(order))
	}
	_ = response.WriteData(w, http.StatusOK, struct {
		Items      []any  `json:"items"`
		NextCursor string `json:"nextCursor,omitempty"`
	}{items, page.NextCursor})
}

func parseOrderPage(location *url.URL) (services.OrderPageRequest, error) {
	query, err := url.ParseQuery(location.RawQuery)
	if err != nil {
		return services.OrderPageRequest{}, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "query parameters are invalid", Cause: err}
	}
	for _, name := range []string{"status", "createdAfter", "createdBefore", "publicReference", "limit", "cursor"} {
		if len(query[name]) > 1 {
			return services.OrderPageRequest{}, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "query parameters may occur at most once"}
		}
	}
	request := services.OrderPageRequest{Status: query.Get("status"), PublicReference: query.Get("publicReference"), Cursor: query.Get("cursor"), Limit: 20}
	if request.Status != "" && !domain.ValidOrderStatus(request.Status) {
		return services.OrderPageRequest{}, domain.NewValidationError(domain.FieldIssue{Field: "status", Code: "invalid", Message: "must be an order status"})
	}
	if request.PublicReference != "" && !domain.ValidPublicReference(request.PublicReference) {
		return services.OrderPageRequest{}, domain.NewValidationError(domain.FieldIssue{Field: "publicReference", Code: "invalid", Message: "must be a ClaimBounty reference"})
	}
	if raw := query.Get("limit"); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || value < 1 || value > 100 {
			return services.OrderPageRequest{}, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "limit must be between 1 and 100", Cause: err}
		}
		request.Limit = uint32(value)
	}
	if len(request.Cursor) > 1024 {
		return services.OrderPageRequest{}, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "cursor is too long"}
	}
	if raw := query.Get("createdAfter"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return services.OrderPageRequest{}, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "createdAfter must be RFC 3339", Cause: err}
		}
		request.CreatedAfter = &value
	}
	if raw := query.Get("createdBefore"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return services.OrderPageRequest{}, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_query", Message: "createdBefore must be RFC 3339", Cause: err}
		}
		request.CreatedBefore = &value
	}
	return request, nil
}

func (routes *ClaimRoutes) getAdminOrder(w http.ResponseWriter, r *http.Request) {
	session, err := routes.readSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	orderID, err := claimID(r.PathValue("orderId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	order, exports, err := routes.administration.GetOrder(r.Context(), session, orderID)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("ETag", versionETag(order.Version))
	_ = response.WriteData(w, http.StatusOK, adminOrderDTO(order, exports))
}

type intakeRequest struct {
	AuditRequest     json.RawMessage `json:"auditRequest"`
	ScientificPolicy json.RawMessage `json:"scientificPolicy"`
	ExecutionPolicy  json.RawMessage `json:"executionPolicy"`
	RoutineContract  *struct {
		RoutineID  *string `json:"routineId"`
		Revision   *string `json:"revision"`
		Validation *struct {
			Status         *string    `json:"status"`
			ValidatedAt    *time.Time `json:"validatedAt"`
			EvidenceSHA256 *string    `json:"evidenceSha256"`
		} `json:"validation"`
	} `json:"routineContract"`
}

func (routes *ClaimRoutes) updateAdminIntake(w http.ResponseWriter, r *http.Request) {
	session, err := routes.unsafeSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	orderID, err := claimID(r.PathValue("orderId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	version, err := parseOrderIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	var body intakeRequest
	if err := response.DecodeJSON(w, r, &body, routes.maxJSONBytes); err != nil {
		routes.writeError(w, r, err)
		return
	}
	if body.RoutineContract == nil || body.RoutineContract.RoutineID == nil || body.RoutineContract.Revision == nil || body.RoutineContract.Validation == nil || body.RoutineContract.Validation.Status == nil || body.RoutineContract.Validation.ValidatedAt == nil || body.RoutineContract.Validation.EvidenceSHA256 == nil || *body.RoutineContract.RoutineID != "claim-bounty-operations/run-claimbounty-scientific-audit" || *body.RoutineContract.Validation.Status != "validated" {
		routes.writeError(w, r, domain.NewValidationError(domain.FieldIssue{Field: "routineContract", Code: "invalid", Message: "must contain the validated ClaimBounty routine contract"}))
		return
	}
	order, exports, err := routes.administration.UpdateIntake(r.Context(), services.AdminIntakeCommand{Session: session, OrderID: orderID, ExpectedVersion: version, AuditRequest: body.AuditRequest, ScientificPolicy: body.ScientificPolicy, ExecutionPolicy: body.ExecutionPolicy, RoutineRevision: *body.RoutineContract.Revision, RoutineValidatedAt: *body.RoutineContract.Validation.ValidatedAt, RoutineEvidenceSHA: *body.RoutineContract.Validation.EvidenceSHA256})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("ETag", versionETag(order.Version))
	_ = response.WriteData(w, http.StatusOK, adminOrderDTO(order, exports))
}

func (routes *ClaimRoutes) downloadAdminFile(w http.ResponseWriter, r *http.Request) {
	session, err := routes.readSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	orderID, err := claimID(r.PathValue("orderId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	fileID, err := claimID(r.PathValue("fileId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	reader, file, err := routes.administration.OpenFile(r.Context(), session, orderID, fileID)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	defer reader.Close()
	contentDigest, err := contentDigestHeader(file.SHA256)
	if err != nil {
		routes.writeError(w, r, domain.ErrFileNotClean)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.OriginalDisplayName}))
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	w.Header().Set("Content-Digest", contentDigest)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.CopyN(w, reader, file.SizeBytes); err != nil {
		routes.logger.WarnContext(r.Context(), "admin file download interrupted", "error", err)
	}
}

type createExportRequest struct {
	RetentionPolicyVersion *string `json:"retentionPolicyVersion"`
	PreserveRunOutputs     *bool   `json:"preserveRunOutputs"`
}

func (routes *ClaimRoutes) createExport(w http.ResponseWriter, r *http.Request) {
	session, err := routes.unsafeSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	orderID, err := claimID(r.PathValue("orderId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	version, err := parseOrderIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	key, err := idempotencyKey(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	var body createExportRequest
	if err := response.DecodeJSON(w, r, &body, routes.maxJSONBytes); err != nil {
		routes.writeError(w, r, err)
		return
	}
	if body.RetentionPolicyVersion == nil || body.PreserveRunOutputs == nil {
		routes.writeError(w, r, domain.NewValidationError(domain.FieldIssue{Field: "body", Code: "required", Message: "export policy fields are required"}))
		return
	}
	export, err := routes.administration.CreateExport(r.Context(), services.CreateExportCommand{Session: session, OrderID: orderID, ExpectedVersion: version, RetentionPolicyVersion: *body.RetentionPolicyVersion, PreserveRunOutputs: *body.PreserveRunOutputs, IdempotencyKey: key, RequestHash: requestHash(body)})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	order, _, err := routes.administration.GetOrder(r.Context(), session, orderID)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/admin/orders/"+orderID.String()+"/exports/"+export.ID.String())
	_ = response.WriteData(w, http.StatusAccepted, exportDTO(export, order.Files))
}

func (routes *ClaimRoutes) getExport(w http.ResponseWriter, r *http.Request) {
	session, err := routes.readSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	orderID, err := claimID(r.PathValue("orderId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	exportID, err := claimID(r.PathValue("exportId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	export, err := routes.administration.GetExport(r.Context(), session, orderID, exportID)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	order, _, err := routes.administration.GetOrder(r.Context(), session, orderID)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	_ = response.WriteData(w, http.StatusOK, exportDTO(export, order.Files))
}

func (routes *ClaimRoutes) downloadExport(w http.ResponseWriter, r *http.Request) {
	session, err := routes.readSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	exportID, err := claimID(r.PathValue("exportId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	reader, export, err := routes.administration.OpenExport(r.Context(), session, exportID)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	defer reader.Close()
	contentDigest, err := contentDigestHeader(export.SHA256)
	if err != nil {
		routes.writeError(w, r, domain.ErrExportNotReady)
		return
	}
	filename := "claimbounty-" + export.ID.String() + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", strconv.FormatInt(export.SizeBytes, 10))
	w.Header().Set("Content-Digest", contentDigest)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.CopyN(w, reader, export.SizeBytes); err != nil {
		routes.logger.WarnContext(r.Context(), "export download interrupted", "error", err)
	}
}

func contentDigestHeader(sha256Hex string) (string, error) {
	digestBytes, err := hex.DecodeString(sha256Hex)
	if err != nil || len(digestBytes) != 32 || hex.EncodeToString(digestBytes) != sha256Hex {
		return "", domain.ErrExportNotReady
	}
	return "sha-256=:" + base64.StdEncoding.EncodeToString(digestBytes) + ":", nil
}
