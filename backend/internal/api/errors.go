package api

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *APIError       `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	CodeValidation   = "MC_VALIDATION"
	CodeNotFound     = "MC_NOT_FOUND"
	CodeConflict     = "MC_CONFLICT"
	CodeUnauthorized = "MC_UNAUTHORIZED"
	CodeForbidden    = "MC_FORBIDDEN"
	CodeDevice       = "MC_DEVICE"
	CodeExecBudget   = "MC_EXEC_BUDGET"
	CodeExecStopped  = "MC_EXEC_STOPPED"
	CodeImport       = "MC_IMPORT"
	CodeInternal     = "MC_INTERNAL"
)

func writeOK(w http.ResponseWriter, data any) {
	b, _ := json.Marshal(data)
	if b == nil {
		b = []byte("{}")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(Envelope{OK: true, Data: b})
}

func writeErr(w http.ResponseWriter, status, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode(status))
	_ = json.NewEncoder(w).Encode(Envelope{OK: false, Error: &APIError{Code: code, Message: msg}})
}

func statusCode(s string) int {
	switch s {
	case "bad":
		return http.StatusBadRequest
	case "unauth":
		return http.StatusUnauthorized
	case "forbid":
		return http.StatusForbidden
	case "notfound":
		return http.StatusNotFound
	case "conflict":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
