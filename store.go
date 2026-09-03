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
	return Paste{}, nil
}

func (s *Store) Get(id string) (Paste, bool) {
	return Paste{}, false
}

func (s *Store) List() []PasteMeta {
	return nil
}

func (s *Store) Delete(id string) bool {
	return false
}
