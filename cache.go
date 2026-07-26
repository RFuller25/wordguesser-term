package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time // zero value means never expires
}

type memCache[T any] struct {
	mu   sync.Mutex
	data map[string]cacheEntry[T]
}

func newMemCache[T any]() *memCache[T] {
	return &memCache[T]{data: make(map[string]cacheEntry[T])}
}

func (c *memCache[T]) get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok {
		var zero T
		return zero, false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		delete(c.data, key)
		var zero T
		return zero, false
	}
	return e.value, true
}

func (c *memCache[T]) set(key string, v T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl != 0 {
		exp = time.Now().Add(ttl)
	}
	c.data[key] = cacheEntry[T]{value: v, expiresAt: exp}
}

func isFinalDate(date, today string) bool {
	return date < today
}

type resultsStore struct {
	fetch func(string) (*ResultsResponse, error)
	mem   *memCache[*ResultsResponse]
	dir   string // "" disables disk persistence
}

func newResultsStore(fetch func(string) (*ResultsResponse, error)) *resultsStore {
	dir := ""
	if base, err := os.UserCacheDir(); err == nil {
		dir = filepath.Join(base, "wordguesser", "results")
	}
	return &resultsStore{fetch: fetch, mem: newMemCache[*ResultsResponse](), dir: dir}
}

func (s *resultsStore) diskPath(date string) string {
	if s.dir == "" {
		return ""
	}
	return filepath.Join(s.dir, date+".json")
}

// Get fetches the ResultsResponse for date ("2006-01-02"). final indicates the
// date is strictly before today (Chicago tz) and therefore immutable: such
// results are cached forever, in memory and on disk. Non-final (today's) results
// are cached in memory only, for 60 seconds, and never written to disk.
func (s *resultsStore) Get(date string, final bool) (*ResultsResponse, error) {
	if v, ok := s.mem.get(date); ok {
		return v, nil
	}

	if final {
		if p := s.diskPath(date); p != "" {
			if data, err := os.ReadFile(p); err == nil {
				var r ResultsResponse
				if json.Unmarshal(data, &r) == nil {
					s.mem.set(date, &r, 0)
					return &r, nil
				}
			}
		}
	}

	r, err := s.fetch(date)
	if err != nil {
		return nil, err
	}

	if final {
		s.mem.set(date, r, 0)
		if p := s.diskPath(date); p != "" {
			if err := os.MkdirAll(s.dir, 0o755); err == nil {
				if data, err := json.MarshalIndent(r, "", "  "); err == nil {
					_ = os.WriteFile(p, data, 0o644)
				}
			}
		}
	} else {
		s.mem.set(date, r, 60*time.Second)
	}

	return r, nil
}

type statsStore struct {
	fetch func(string) (*UserStatsResponse, error)
	mem   *memCache[*UserStatsResponse]
}

func newStatsStore(fetch func(string) (*UserStatsResponse, error)) *statsStore {
	return &statsStore{fetch: fetch, mem: newMemCache[*UserStatsResponse]()}
}

func (s *statsStore) Get(username string) (*UserStatsResponse, error) {
	if v, ok := s.mem.get(username); ok {
		return v, nil
	}
	r, err := s.fetch(username)
	if err != nil {
		return nil, err
	}
	s.mem.set(username, r, 5*time.Minute)
	return r, nil
}
