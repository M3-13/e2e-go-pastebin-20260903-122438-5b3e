package main

import (
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCreateAndGet(t *testing.T) {
	s := NewStore()
	p, err := s.Create("hello", "text", 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if p.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if p.Content != "hello" {
		t.Fatalf("content = %q, want %q", p.Content, "hello")
	}
	if p.Language != "text" {
		t.Fatalf("language = %q, want %q", p.Language, "text")
	}
	if p.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if p.ExpiresAt != nil {
		t.Fatal("expected ExpiresAt to be nil for expiresIn <= 0")
	}

	got, ok := s.Get(p.ID)
	if !ok {
		t.Fatal("expected Get to find the paste")
	}
	if got.Content != "hello" {
		t.Fatalf("Get content = %q, want %q", got.Content, "hello")
	}
}

func TestCreateWithExpiry(t *testing.T) {
	s := NewStore()
	p, err := s.Create("secret", "text", time.Hour)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if p.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	if !p.ExpiresAt.After(p.CreatedAt) {
		t.Fatal("expected ExpiresAt to be after CreatedAt")
	}
	if p.ExpiresAt.Sub(p.CreatedAt) != time.Hour {
		t.Fatalf("expiry duration = %v, want %v", p.ExpiresAt.Sub(p.CreatedAt), time.Hour)
	}
}

func TestGetUnknown(t *testing.T) {
	s := NewStore()
	if _, ok := s.Get("nonexistent"); ok {
		t.Fatal("expected Get to return false for unknown ID")
	}
}

func TestList(t *testing.T) {
	s := NewStore()
	p1, _ := s.Create("a", "go", 0)
	p2, _ := s.Create("b", "py", 0)

	metas := s.List()
	if len(metas) != 2 {
		t.Fatalf("List returned %d entries, want 2", len(metas))
	}
	seen := map[string]bool{}
	for _, m := range metas {
		seen[m.ID] = true
		if m.ID != p1.ID && m.ID != p2.ID {
			t.Fatalf("unexpected ID %q in list", m.ID)
		}
		if m.CreatedAt.IsZero() {
			t.Fatal("expected CreatedAt to be set in list")
		}
	}
	if !seen[p1.ID] || !seen[p2.ID] {
		t.Fatal("expected both created pastes to appear in List")
	}
}

func TestDelete(t *testing.T) {
	s := NewStore()
	p, _ := s.Create("x", "go", 0)

	if !s.Delete(p.ID) {
		t.Fatal("expected Delete to return true for existing paste")
	}
	if _, ok := s.Get(p.ID); ok {
		t.Fatal("expected paste to be gone after Delete")
	}
	if s.Delete(p.ID) {
		t.Fatal("expected Delete to return false for already-deleted paste")
	}
}

func TestExpiryLazyRemoval(t *testing.T) {
	s := NewStore()
	p, _ := s.Create("soon", "go", 50*time.Millisecond)

	if _, ok := s.Get(p.ID); !ok {
		t.Fatal("expected paste to be available before expiry")
	}

	time.Sleep(80 * time.Millisecond)

	if _, ok := s.Get(p.ID); ok {
		t.Fatal("expected expired paste to be unavailable from Get")
	}
	if metas := s.List(); len(metas) != 0 {
		t.Fatalf("expected expired paste to be absent from List, got %d entries", len(metas))
	}
	if s.Delete(p.ID) {
		t.Fatal("expected Delete on expired paste to return false")
	}
}

func TestIDFormatAndUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := NewID()
		if len(id) != 32 {
			t.Fatalf("ID length = %d, want 32", len(id))
		}
		if _, err := hex.DecodeString(id); err != nil {
			t.Fatalf("ID %q is not valid hex: %v", id, err)
		}
		if strings.ToLower(id) != id {
			t.Fatalf("ID %q is not lowercase hex", id)
		}
		if ids[id] {
			t.Fatalf("duplicate ID %q generated", id)
		}
		ids[id] = true
	}
}

func TestConcurrentCreateGetListDelete(t *testing.T) {
	s := NewStore()

	var wg sync.WaitGroup
	const workers = 50
	const perWorker = 20

	created := make([]string, 0, workers*perWorker)
	var createdMu sync.Mutex

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				p, err := s.Create("c", "go", 0)
				if err != nil {
					t.Errorf("Create error: %v", err)
					return
				}
				createdMu.Lock()
				created = append(created, p.ID)
				createdMu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	if len(created) != workers*perWorker {
		t.Fatalf("created %d pastes, want %d", len(created), workers*perWorker)
	}

	// Verify every created paste is retrievable and unique.
	seen := map[string]bool{}
	for _, id := range created {
		if seen[id] {
			t.Fatalf("duplicate ID %q", id)
		}
		seen[id] = true
		if _, ok := s.Get(id); !ok {
			t.Fatalf("paste %q not retrievable after concurrent create", id)
		}
	}

	metas := s.List()
	if len(metas) != workers*perWorker {
		t.Fatalf("List returned %d entries, want %d", len(metas), workers*perWorker)
	}

	// Concurrently delete.
	for _, id := range created {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if !s.Delete(id) {
				t.Errorf("Delete(%q) returned false", id)
			}
		}(id)
	}
	wg.Wait()

	if metas := s.List(); len(metas) != 0 {
		t.Fatalf("expected empty store after deleting all, got %d entries", len(metas))
	}
}
