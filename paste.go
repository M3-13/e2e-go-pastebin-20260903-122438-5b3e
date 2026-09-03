package main

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"
)

const maxPasteBodyBytes = 1 << 20

type createPasteRequest struct {
	Content          string `json:"content"`
	Language         string `json:"language"`
	ExpiresInSeconds *int64 `json:"expires_in_seconds"`
}

func CreatePasteHandler(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			writeError(w, http.StatusUnsupportedMediaType, "unsupported content type")
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPasteBodyBytes)

	var req createPasteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.ExpiresInSeconds != nil && *req.ExpiresInSeconds <= 0 {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	var expiresIn time.Duration
	if req.ExpiresInSeconds != nil {
		expiresIn = time.Duration(*req.ExpiresInSeconds) * time.Second
	}

	paste, err := store.Create(req.Content, req.Language, expiresIn)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": paste.ID})
}

func GetPasteHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidHex(id) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	paste, ok := store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "paste not found")
		return
	}

	writeJSON(w, http.StatusOK, paste)
}

func isValidHex(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
