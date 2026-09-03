package main

import (
	"sync"
	"time"
)

type Paste struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	Language  string     `json:"language"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type PasteMeta struct {
	ID        string     `json:"id"`
	Language  string     `json:"language"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type Store struct {
	mu     sync.RWMutex
	pastes map[string]Paste
}

func NewStore() *Store {
	return &Store{
		pastes: make(map[string]Paste),
	}
}

func (s *Store) Create(content, language string, expiresIn time.Duration) (Paste, error) {
	now := time.Now().UTC()
	p := Paste{
		ID:        NewID(),
		Content:   content,
		Language:  language,
		CreatedAt: now,
	}
	if expiresIn > 0 {
		exp := now.Add(expiresIn)
		p.ExpiresAt = &exp
	}

	s.mu.Lock()
	s.pastes[p.ID] = p
	s.mu.Unlock()

	return p, nil
}

func (s *Store) Get(id string) (Paste, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pastes[id]
	if !ok {
		return Paste{}, false
	}
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		delete(s.pastes, id)
		return Paste{}, false
	}
	return p, true
}

func (s *Store) List() []PasteMeta {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	metas := make([]PasteMeta, 0, len(s.pastes))
	for id, p := range s.pastes {
		if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
			delete(s.pastes, id)
			continue
		}
		metas = append(metas, PasteMeta{
			ID:        p.ID,
			Language:  p.Language,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
		})
	}
	return metas
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pastes[id]
	if !ok {
		return false
	}
	delete(s.pastes, id)
	return p.ExpiresAt == nil || time.Now().Before(*p.ExpiresAt)
}
