package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/domain"
)

type envelope struct {
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}
type errorBody struct {
	Error     errorDetail `json:"error"`
	RequestID string      `json:"request_id"`
}
type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Details any    `json:"details,omitempty"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return domain.NewError(domain.CodeInvalid, "请求体超过 1 MiB 限制")
		}
		if errors.Is(err, io.EOF) {
			return domain.NewError(domain.CodeInvalid, "请求体不能为空")
		}
		return domain.NewError(domain.CodeInvalid, "JSON 请求体无效: "+err.Error())
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.NewError(domain.CodeInvalid, "请求体只能包含一个 JSON 对象")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func success(w http.ResponseWriter, r *http.Request, status int, value any) {
	writeJSON(w, status, envelope{Data: value, RequestID: requestID(r.Context())})
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	detail := errorDetail{Code: string(domain.CodeInternal), Message: "服务内部错误"}
	var de *domain.Error
	if errors.As(err, &de) {
		detail = errorDetail{Code: string(de.Code), Message: de.Message, Field: de.Field, Details: de.Details}
		switch de.Code {
		case domain.CodeInvalid:
			status = http.StatusBadRequest
		case domain.CodeNotFound:
			status = http.StatusNotFound
		case domain.CodeForbidden:
			status = http.StatusForbidden
		case domain.CodeConflict, domain.CodeIdempotency:
			status = http.StatusConflict
		case domain.CodeState, domain.CodeChain, domain.CodeClosed:
			status = http.StatusUnprocessableEntity
		}
	}
	writeJSON(w, status, errorBody{Error: detail, RequestID: requestID(r.Context())})
}

func actor(r *http.Request) (application.Actor, error) {
	value := application.Actor{Name: domain.NormalizeText(r.Header.Get("X-Actor")), Role: domain.NormalizeText(r.Header.Get("X-Role"))}
	if value.Name == "" {
		return value, domain.FieldError("X-Actor", "必须提供操作人头")
	}
	if value.Role == "" {
		return value, domain.FieldError("X-Role", "必须提供角色头")
	}
	return value, nil
}

func revision(r *http.Request, bodyRevision int64) (int64, error) {
	value := strings.Trim(strings.TrimSpace(r.Header.Get("If-Match")), `"`)
	if value == "" {
		if bodyRevision <= 0 {
			return 0, domain.FieldError("revision", "必须通过 If-Match 或请求字段提供修订号")
		}
		return bodyRevision, nil
	}
	value = strings.TrimPrefix(value, "W/\"")
	value = strings.TrimSuffix(value, "\"")
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, domain.FieldError("If-Match", "If-Match 必须是正整数修订号")
	}
	return parsed, nil
}

func setRevision(w http.ResponseWriter, value int64) {
	w.Header().Set("ETag", fmt.Sprintf(`"%d"`, value))
}
