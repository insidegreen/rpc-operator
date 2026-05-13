package api

import (
	"encoding/json"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type errorResponse struct {
	Error   string            `json:"error"`
	Details string            `json:"details,omitempty"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logf.Log.Error(err, "writeJSON encode failed")
	}
}

func writeJSONError(w http.ResponseWriter, code int, msg, details string) {
	writeJSON(w, code, errorResponse{Error: msg, Details: details})
}

func writeValidationErrors(w http.ResponseWriter, errs []ValidationError) {
	writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
		Error:  "validation failed",
		Errors: errs,
	})
}

func writeJSONWithWarning(w http.ResponseWriter, code int, body any, warning string) {
	type withWarning struct {
		Data     any      `json:"data"`
		Warnings []string `json:"warnings,omitempty"`
	}
	writeJSON(w, code, withWarning{Data: body, Warnings: []string{warning}})
}

func writeK8sError(w http.ResponseWriter, err error) {
	switch {
	case apierrors.IsNotFound(err):
		writeJSONError(w, http.StatusNotFound, "not found", err.Error())
	case apierrors.IsAlreadyExists(err):
		writeJSONError(w, http.StatusConflict, "already exists", err.Error())
	case apierrors.IsConflict(err):
		writeJSONError(w, http.StatusConflict, "conflict", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal error", err.Error())
	}
}
