package response

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Data  any    `json:"data,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
	Error *Error `json:"error,omitempty"`
}

type Meta struct {
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	Total int64 `json:"total"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Data: data})
}

func JSONList(w http.ResponseWriter, data any, page, size int, total int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Data: data,
		Meta: &Meta{Page: page, Size: size, Total: total},
	})
}

func Err(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Error: &Error{Code: code, Message: message},
	})
}

func BadRequest(w http.ResponseWriter, message string) {
	Err(w, http.StatusBadRequest, "bad_request", message)
}

func Unauthorized(w http.ResponseWriter, message string) {
	Err(w, http.StatusUnauthorized, "unauthorized", message)
}

func Forbidden(w http.ResponseWriter, message string) {
	Err(w, http.StatusForbidden, "forbidden", message)
}

func NotFound(w http.ResponseWriter, message string) {
	Err(w, http.StatusNotFound, "not_found", message)
}

func InternalError(w http.ResponseWriter, message string) {
	Err(w, http.StatusInternalServerError, "internal_error", message)
}
