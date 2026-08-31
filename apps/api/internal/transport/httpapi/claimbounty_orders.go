package httpapi

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi/response"
)

type IntakeActions interface {
	CreateOrder(context.Context, services.CreateOrderCommand) (domain.Order, error)
	GetOwnedOrder(context.Context, domain.Session, domain.Identifier) (domain.Order, error)
	UploadFile(context.Context, services.UploadFileCommand) (services.UploadedFile, error)
	DeleteFile(context.Context, services.DeleteFileCommand) error
	SubmitOrder(context.Context, services.SubmitOrderCommand) (domain.Order, error)
}

type createOrderRequest struct {
	Title       *string `json:"title"`
	Purpose     *string `json:"purpose"`
	TargetClaim *struct {
		Text           *string `json:"text"`
		SourceLocation *string `json:"sourceLocation"`
	} `json:"targetClaim"`
	Permissions *struct {
		ExecuteSuppliedCode *bool `json:"executeSuppliedCode"`
		ExternalSearch      *bool `json:"externalSearch"`
	} `json:"permissions"`
	Privacy *struct {
		ContainsParticipantLevelData *bool `json:"containsParticipantLevelData"`
		ContainsDirectIdentifiers    *bool `json:"containsDirectIdentifiers"`
	} `json:"privacy"`
}

func (routes *ClaimRoutes) createOrder(w http.ResponseWriter, r *http.Request) {
	session, err := routes.unsafeSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	key, err := idempotencyKey(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	var body createOrderRequest
	if err := response.DecodeJSON(w, r, &body, routes.maxJSONBytes); err != nil {
		routes.writeError(w, r, err)
		return
	}
	if body.Title == nil || body.Purpose == nil || body.TargetClaim == nil || body.TargetClaim.Text == nil || body.Permissions == nil || body.Permissions.ExecuteSuppliedCode == nil || body.Permissions.ExternalSearch == nil || body.Privacy == nil || body.Privacy.ContainsParticipantLevelData == nil || body.Privacy.ContainsDirectIdentifiers == nil {
		routes.writeError(w, r, domain.NewValidationError(domain.FieldIssue{Field: "body", Code: "required", Message: "all required order fields must be provided"}))
		return
	}
	source := ""
	if body.TargetClaim.SourceLocation != nil {
		source = *body.TargetClaim.SourceLocation
	}
	order, err := routes.intake.CreateOrder(r.Context(), services.CreateOrderCommand{Session: session, Title: *body.Title, Purpose: *body.Purpose, TargetClaim: domain.TargetClaim{Text: *body.TargetClaim.Text, SourceLocation: source}, Permissions: domain.Permissions{ExecuteSuppliedCode: *body.Permissions.ExecuteSuppliedCode, ExternalSearch: *body.Permissions.ExternalSearch}, Privacy: domain.Privacy{ContainsParticipantLevelData: *body.Privacy.ContainsParticipantLevelData, ContainsDirectIdentifiers: *body.Privacy.ContainsDirectIdentifiers}, IdempotencyKey: key, RequestHash: requestHash(body)})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/orders/"+order.ID.String())
	w.Header().Set("ETag", versionETag(order.Version))
	_ = response.WriteData(w, http.StatusCreated, orderDTO(order))
}

func (routes *ClaimRoutes) getOrder(w http.ResponseWriter, r *http.Request) {
	session, err := routes.readSession(r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	id, err := claimID(r.PathValue("orderId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	order, err := routes.intake.GetOwnedOrder(r.Context(), session, id)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("ETag", versionETag(order.Version))
	_ = response.WriteData(w, http.StatusOK, orderDTO(order))
}

func (routes *ClaimRoutes) uploadFile(w http.ResponseWriter, r *http.Request) {
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
	upload, filePart, err := routes.readMultipartUpload(w, r)
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	defer filePart.Close()
	result, err := routes.intake.UploadFile(r.Context(), services.UploadFileCommand{Session: session, OrderID: orderID, ExpectedVersion: version, Role: upload.role, OriginalDisplayName: upload.name, SizeBytes: upload.size, SHA256: upload.sha, DeclaredMediaType: upload.mediaType, IdempotencyKey: key, RequestHash: requestHash(struct {
		OrderID                    string
		Version                    uint64
		Role, Name, SHA, MediaType string
		Size                       int64
	}{orderID.String(), version, upload.role, upload.name, upload.sha, upload.mediaType, upload.size}), Body: filePart})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	if err := filePart.Finish(); err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/orders/"+orderID.String()+"/files/"+result.File.ID.String())
	w.Header().Set("ETag", versionETag(result.Order.Version))
	_ = response.WriteData(w, http.StatusCreated, fileDTO(result.File))
}

type multipartUpload struct {
	role, name, sha, mediaType string
	size                       int64
}

func (routes *ClaimRoutes) readMultipartUpload(w http.ResponseWriter, r *http.Request) (multipartUpload, *singleMultipartFile, error) {
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxFileBytes+2<<20)
	reader, err := r.MultipartReader()
	if err != nil {
		return multipartUpload{}, nil, &response.ClientError{Status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type", Message: "Content-Type must be multipart/form-data", Cause: err}
	}
	result := multipartUpload{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return multipartUpload{}, nil, domain.NewValidationError(domain.FieldIssue{Field: "file", Code: "required", Message: "must be provided"})
		}
		if err != nil {
			return multipartUpload{}, nil, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_multipart", Message: "multipart body is invalid", Cause: err}
		}
		name := part.FormName()
		if name == "file" {
			if result.role == "" || result.name == "" || result.size < 1 || result.sha == "" || result.mediaType == "" {
				_ = part.Close()
				return multipartUpload{}, nil, domain.NewValidationError(domain.FieldIssue{Field: "multipart", Code: "invalid", Message: "metadata fields must precede the file field"})
			}
			return result, &singleMultipartFile{part: part, reader: reader}, nil
		}
		value, readErr := readSmallPart(part, 2048)
		if readErr != nil {
			_ = part.Close()
			return multipartUpload{}, nil, readErr
		}
		switch name {
		case "role":
			result.role = value
		case "originalDisplayName":
			result.name = value
		case "sizeBytes":
			result.size, err = strconv.ParseInt(value, 10, 64)
		case "expectedSha256":
			result.sha = value
		case "declaredMediaType":
			result.mediaType = value
		default:
			err = errors.New("unexpected multipart field")
		}
		_ = part.Close()
		if err != nil {
			return multipartUpload{}, nil, &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_multipart", Message: "multipart fields are invalid", Cause: err}
		}
	}
}

type singleMultipartFile struct {
	part     *multipart.Part
	reader   *multipart.Reader
	checked  bool
	trailing bool
}

func (file *singleMultipartFile) Read(buffer []byte) (int, error) {
	n, err := file.part.Read(buffer)
	if n > 0 {
		return n, nil
	}
	if !errors.Is(err, io.EOF) {
		return n, err
	}
	if checkErr := file.checkTrailing(); checkErr != nil {
		if len(buffer) > 0 {
			buffer[0] = 'x'
			return 1, nil
		}
		return 0, checkErr
	}
	return 0, io.EOF
}
func (file *singleMultipartFile) Close() error { return file.part.Close() }
func (file *singleMultipartFile) Finish() error {
	if !file.checked {
		if _, err := io.Copy(io.Discard, file.part); err != nil {
			return err
		}
		return file.checkTrailing()
	}
	if file.trailing {
		return domain.NewValidationError(domain.FieldIssue{Field: "file", Code: "too_many", Message: "exactly one file is allowed"})
	}
	return nil
}
func (file *singleMultipartFile) checkTrailing() error {
	if file.checked {
		if file.trailing {
			return domain.NewValidationError(domain.FieldIssue{Field: "file", Code: "too_many", Message: "exactly one file is allowed"})
		}
		return nil
	}
	file.checked = true
	next, err := file.reader.NextPart()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_multipart", Message: "multipart body is invalid", Cause: err}
	}
	file.trailing = true
	_ = next.Close()
	return domain.NewValidationError(domain.FieldIssue{Field: "file", Code: "too_many", Message: "exactly one file is allowed"})
}

func readSmallPart(part *multipart.Part, limit int64) (string, error) {
	var builder strings.Builder
	written, err := io.CopyN(&builder, part, limit+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if written > limit {
		return "", &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_multipart", Message: "multipart text field is too large"}
	}
	return builder.String(), nil
}

func (routes *ClaimRoutes) deleteFile(w http.ResponseWriter, r *http.Request) {
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
	fileID, err := claimID(r.PathValue("fileId"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	version, err := parseOrderIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	if err := routes.intake.DeleteFile(r.Context(), services.DeleteFileCommand{Session: session, OrderID: orderID, FileID: fileID, ExpectedVersion: version}); err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type submitOrderRequest struct {
	TermsAccepted                    *bool   `json:"termsAccepted"`
	TermsVersion                     *string `json:"termsVersion"`
	UploadsAuthorized                *bool   `json:"uploadsAuthorized"`
	AnalysisUseAuthorized            *bool   `json:"analysisUseAuthorized"`
	ExternalRedistributionAuthorized *bool   `json:"externalRedistributionAuthorized"`
}

func (routes *ClaimRoutes) submitOrder(w http.ResponseWriter, r *http.Request) {
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
	var body submitOrderRequest
	if err := response.DecodeJSON(w, r, &body, routes.maxJSONBytes); err != nil {
		routes.writeError(w, r, err)
		return
	}
	if body.TermsAccepted == nil || body.TermsVersion == nil || body.UploadsAuthorized == nil || body.AnalysisUseAuthorized == nil || body.ExternalRedistributionAuthorized == nil {
		routes.writeError(w, r, domain.NewValidationError(domain.FieldIssue{Field: "customerConsent", Code: "required", Message: "terms, upload, analysis, and redistribution assertions are required"}))
		return
	}
	if !*body.TermsAccepted || !*body.UploadsAuthorized || !*body.AnalysisUseAuthorized || *body.ExternalRedistributionAuthorized {
		routes.writeError(w, r, domain.NewValidationError(domain.FieldIssue{Field: "customerConsent", Code: "invalid", Message: "P0 requires explicit upload and internal analysis consent and prohibits external redistribution"}))
		return
	}
	order, err := routes.intake.SubmitOrder(r.Context(), services.SubmitOrderCommand{Session: session, OrderID: orderID, ExpectedVersion: version, TermsAccepted: *body.TermsAccepted, TermsVersion: *body.TermsVersion, UploadsAuthorized: *body.UploadsAuthorized, AnalysisUseAuthorized: *body.AnalysisUseAuthorized, ExternalRedistributionAuthorized: *body.ExternalRedistributionAuthorized, IdempotencyKey: key, RequestHash: requestHash(body)})
	if err != nil {
		routes.writeError(w, r, err)
		return
	}
	w.Header().Set("ETag", versionETag(order.Version))
	_ = response.WriteData(w, http.StatusAccepted, orderDTO(order))
}

func claimID(raw string) (domain.Identifier, error) {
	id, err := domain.NewIdentifier(raw)
	if err != nil {
		return "", &response.ClientError{Status: http.StatusBadRequest, Code: "invalid_path_parameter", Message: "path identifier must be a UUID", Cause: err}
	}
	return id, nil
}
func parseOrderIfMatch(raw string) (uint64, error) {
	version, err := parseIfMatch(raw)
	if err != nil || version > domain.MaxOrderVersion {
		return 0, &response.ClientError{Status: http.StatusPreconditionRequired, Code: "if_match_required", Message: "If-Match must contain the current quoted order version", Cause: err}
	}
	return version, nil
}
func idempotencyKey(r *http.Request) (string, error) {
	value := r.Header.Get("Idempotency-Key")
	if len(value) < 16 || len(value) > 128 || strings.ContainsAny(value, " \t\r\n") {
		return "", domain.NewValidationError(domain.FieldIssue{Field: "Idempotency-Key", Code: "invalid", Message: "must be 16 to 128 non-whitespace characters"})
	}
	return value, nil
}
