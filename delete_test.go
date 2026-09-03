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

func TestDeletePaste(t *testing.T) {
	router := NewRouter()

	p, err := store.Create("hello world", "text", 0)
	if err != nil {
		t.Fatalf("create paste: %v", err)
	}

	// First DELETE removes the paste with 204 and an empty body.
	req := httptest.NewRequest(http.MethodDelete, "/pastes/"+p.ID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("first DELETE: expected 204, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("first DELETE: expected empty body, got %q", rec.Body.String())
	}

	// Second DELETE returns 404 in the JSON error format.
	req2 := httptest.NewRequest(http.MethodDelete, "/pastes/"+p.ID, nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotFound {
		t.Fatalf("second DELETE: expected 404, got %d", rec2.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec2.Body.Bytes(), &body); err != nil {
		t.Fatalf("second DELETE: expected JSON error body, got error: %v", err)
	}
	if body["error"] != "paste not found" {
		t.Fatalf("second DELETE: expected error %q, got %q", "paste not found", body["error"])
	}

	// The paste is no longer retrievable.
	if _, ok := store.Get(p.ID); ok {
		t.Fatal("expected paste to be gone after delete")
	}
}

func TestDeletePasteEmptyID(t *testing.T) {
	// The empty-id branch of the handler is defensive: ServeMux's `{id}`
	// wildcard never matches an empty segment, so it is exercised directly.
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
