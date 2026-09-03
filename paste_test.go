package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func postPaste(t *testing.T, body string, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	router := NewRouter()
	req := httptest.NewRequest(http.MethodPost, "/pastes", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func getPaste(t *testing.T, id string) *httptest.ResponseRecorder {
	t.Helper()
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/pastes/"+id, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestCreatePasteReturns201WithID(t *testing.T) {
	rec := postPaste(t, `{"content":"hello world"}`, "application/json")

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got error: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Fatalf("expected id key in body, got %v", body)
	}
}

func TestCreatePasteEmptyContentReturns400(t *testing.T) {
	rec := postPaste(t, `{"content":""}`, "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreatePasteMissingContentReturns400(t *testing.T) {
	rec := postPaste(t, `{"language":"go"}`, "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreatePasteInvalidJSONReturns400(t *testing.T) {
	rec := postPaste(t, `not-json`, "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreatePasteNonPositiveExpiryReturns400(t *testing.T) {
	for _, body := range []string{
		`{"content":"x","expires_in_seconds":0}`,
		`{"content":"x","expires_in_seconds":-5}`,
	} {
		rec := postPaste(t, body, "application/json")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for %q, got %d", body, rec.Code)
		}
	}
}

func TestCreatePasteWrongContentTypeReturns415(t *testing.T) {
	rec := postPaste(t, `{"content":"x"}`, "text/plain")
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status 415, got %d", rec.Code)
	}
}

func TestCreatePasteBodyTooLargeReturns413(t *testing.T) {
	body := `{"content":"` + strings.Repeat("a", maxPasteBodyBytes) + `"}`
	rec := postPaste(t, body, "application/json")
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", rec.Code)
	}
}

func TestGetPasteUnknownIDReturns404(t *testing.T) {
	rec := getPaste(t, "abcdef")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestGetPasteInvalidIDReturns400(t *testing.T) {
	for _, id := range []string{"xyz!", "not-hex", "abc-123"} {
		rec := getPaste(t, id)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for id %q, got %d", id, rec.Code)
		}
	}
}

func TestGetPasteResponseContentTypeJSON(t *testing.T) {
	rec := getPaste(t, "abcdef")
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
}

func TestCreatePasteResponseContentTypeJSON(t *testing.T) {
	rec := postPaste(t, `{"content":"x"}`, "application/json")
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
}

func TestGetPasteFoundReturns200WithContent(t *testing.T) {
	store = NewStore()
	store.pastes["deadbeef"] = Paste{
		ID:        "deadbeef",
		Content:   "hello world",
		Language:  "go",
		CreatedAt: time.Now().UTC(),
	}

	rec := getPaste(t, "deadbeef")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body Paste
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got error: %v", err)
	}
	if body.ID != "deadbeef" {
		t.Fatalf("expected id %q, got %q", "deadbeef", body.ID)
	}
	if body.Content != "hello world" {
		t.Fatalf("expected content %q, got %q", "hello world", body.Content)
	}
	if body.Language != "go" {
		t.Fatalf("expected language %q, got %q", "go", body.Language)
	}
}

func TestCreatePasteThenGetReturnsContent(t *testing.T) {
	store = NewStore()

	rec := postPaste(t, `{"content":"roundtrip content","language":"go"}`, "application/json")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var created map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("expected JSON body, got error: %v", err)
	}
	id := created["id"]
	if id == "" {
		t.Fatalf("expected non-empty id in body, got %v", created)
	}

	getRec := getPaste(t, id)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getRec.Code)
	}

	var body Paste
	if err := json.Unmarshal(getRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got error: %v", err)
	}
	if body.ID != id {
		t.Fatalf("expected id %q, got %q", id, body.ID)
	}
	if body.Content != "roundtrip content" {
		t.Fatalf("expected content %q, got %q", "roundtrip content", body.Content)
	}
}
