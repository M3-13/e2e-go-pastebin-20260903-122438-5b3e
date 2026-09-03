package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got error: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestUnknownPathReturns404JSON(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error body, got error: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatalf("expected error key in body, got %v", body)
	}
}

func TestStubRoutesAreWired(t *testing.T) {
	router := NewRouter()

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/pastes"},
		{http.MethodGet, "/pastes"},
		{http.MethodGet, "/pastes/not-valid"},
	}

	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Errorf("%s %s should be registered, got 404", c.method, c.path)
		}
	}
}

func TestMethodNotAllowedOnPastes(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodPut, "/pastes", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error body, got error: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatalf("expected error key in body, got %v", body)
	}
}
