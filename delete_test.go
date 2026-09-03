package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeletePasteInvalidID(t *testing.T) {
	router := NewRouter()

	for _, id := range []string{"xyz", "123xyz", "-", "G", "!@#"} {
		req := httptest.NewRequest(http.MethodDelete, "/pastes/"+id, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("DELETE /pastes/%q: expected 400, got %d", id, rec.Code)
		}

		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("DELETE /pastes/%q: expected JSON error body, got error: %v", id, err)
		}
		if body["error"] != "invalid request" {
			t.Errorf("DELETE /pastes/%q: expected error %q, got %q", id, "invalid request", body["error"])
		}
	}
}

func TestDeletePasteEmptyID(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/pastes/", nil)
	rec := httptest.NewRecorder()
	DeletePasteHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty id, got %d", rec.Code)
	}
}

func TestDeletePasteUnknownID(t *testing.T) {
	router := NewRouter()

	for _, id := range []string{"deadbeef", "abc123", "00"} {
		req := httptest.NewRequest(http.MethodDelete, "/pastes/"+id, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("DELETE /pastes/%q: expected 404, got %d", id, rec.Code)
		}

		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("DELETE /pastes/%q: expected JSON error body, got error: %v", id, err)
		}
		if body["error"] != "paste not found" {
			t.Errorf("DELETE /pastes/%q: expected error %q, got %q", id, "paste not found", body["error"])
		}
	}
}

func TestDeletePasteMethodNotAllowed(t *testing.T) {
	router := NewRouter()

	req := httptest.NewRequest(http.MethodPut, "/pastes/abc123", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for PUT, got %d", rec.Code)
	}
}
