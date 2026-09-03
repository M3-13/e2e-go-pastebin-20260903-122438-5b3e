package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListPastesReturns200JSON(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
}

func TestListPastesReturnsJSONArray(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var metas []PasteMeta
	if err := json.Unmarshal(rec.Body.Bytes(), &metas); err != nil {
		t.Fatalf("expected JSON array, got error: %v (body=%q)", err, rec.Body.String())
	}
	if metas == nil {
		t.Fatalf("expected a JSON array [], got null (body=%q)", rec.Body.String())
	}
}

func TestListPastesResponseHasNoContentField(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var raw []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("expected JSON array, got error: %v (body=%q)", err, rec.Body.String())
	}
	for _, item := range raw {
		if _, ok := item["content"]; ok {
			t.Fatalf("list response must not contain a content field, got %q", rec.Body.String())
		}
	}
}

func TestPasteMetaSerializationOmitsContent(t *testing.T) {
	now := time.Now().UTC()
	exp := now.Add(time.Hour)
	m := PasteMeta{ID: "abc123", Language: "go", CreatedAt: now, ExpiresAt: &exp}

	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal PasteMeta: %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatalf("unmarshal PasteMeta: %v", err)
	}

	if _, ok := fields["content"]; ok {
		t.Fatalf("PasteMeta must not serialize a content field, got %s", b)
	}
	for _, k := range []string{"id", "language", "created_at", "expires_at"} {
		if _, ok := fields[k]; !ok {
			t.Fatalf("PasteMeta must include field %q, got %s", k, b)
		}
	}
}
