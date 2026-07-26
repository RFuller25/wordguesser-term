package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemCache_SetAndGet(t *testing.T) {
	c := newMemCache[string]()
	c.set("a", "hello", time.Minute)
	v, ok := c.get("a")
	if !ok || v != "hello" {
		t.Fatalf("expected (hello, true), got (%q, %v)", v, ok)
	}
}

func TestMemCache_MissingKey(t *testing.T) {
	c := newMemCache[string]()
	_, ok := c.get("missing")
	if ok {
		t.Fatal("expected miss for unset key")
	}
}

func TestMemCache_ExpiredEntryIsAMiss(t *testing.T) {
	c := newMemCache[string]()
	c.set("a", "hello", -time.Second) // already in the past
	_, ok := c.get("a")
	if ok {
		t.Fatal("expected expired entry to be a miss")
	}
}

func TestMemCache_ZeroTTLNeverExpires(t *testing.T) {
	c := newMemCache[int]()
	c.set("a", 42, 0)
	v, ok := c.get("a")
	if !ok || v != 42 {
		t.Fatalf("expected (42, true), got (%d, %v)", v, ok)
	}
}

func TestIsFinalDate(t *testing.T) {
	if !isFinalDate("2026-01-01", "2026-01-02") {
		t.Fatal("expected 2026-01-01 to be final relative to 2026-01-02")
	}
	if isFinalDate("2026-01-02", "2026-01-02") {
		t.Fatal("expected today to not be final")
	}
	if isFinalDate("2026-01-03", "2026-01-02") {
		t.Fatal("expected a future date to not be final")
	}
}

func TestResultsStore_FinalDateCachesInMemAndOnDisk(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	fetch := func(date string) (*ResultsResponse, error) {
		calls++
		return &ResultsResponse{Date: date, Word: "crane"}, nil
	}
	s := &resultsStore{fetch: fetch, mem: newMemCache[*ResultsResponse](), dir: dir}

	r1, err := s.Get("2026-01-01", true)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Word != "crane" {
		t.Fatalf("expected word crane, got %q", r1.Word)
	}
	if calls != 1 {
		t.Fatalf("expected 1 fetch, got %d", calls)
	}

	if _, err := s.Get("2026-01-01", true); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected mem cache hit (still 1 fetch), got %d", calls)
	}

	// A fresh store (simulating a new process) should hit the disk cache.
	s2 := &resultsStore{fetch: fetch, mem: newMemCache[*ResultsResponse](), dir: dir}
	r2, err := s2.Get("2026-01-01", true)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Word != "crane" {
		t.Fatalf("expected disk-cached word crane, got %q", r2.Word)
	}
	if calls != 1 {
		t.Fatalf("expected disk cache hit (still 1 fetch), got %d", calls)
	}

	if _, err := os.Stat(filepath.Join(dir, "2026-01-01.json")); err != nil {
		t.Fatalf("expected cache file to exist: %v", err)
	}
}

func TestResultsStore_NonFinalDateNeverTouchesDisk(t *testing.T) {
	dir := t.TempDir()
	fetch := func(date string) (*ResultsResponse, error) {
		return &ResultsResponse{Date: date}, nil
	}
	s := &resultsStore{fetch: fetch, mem: newMemCache[*ResultsResponse](), dir: dir}

	if _, err := s.Get("2026-01-02", false); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no disk files for a non-final date, found %d", len(entries))
	}
}

func TestResultsStore_NonFinalDateStillMemCachesBriefly(t *testing.T) {
	calls := 0
	fetch := func(date string) (*ResultsResponse, error) {
		calls++
		return &ResultsResponse{Date: date}, nil
	}
	s := &resultsStore{fetch: fetch, mem: newMemCache[*ResultsResponse]()}

	if _, err := s.Get("2026-01-02", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("2026-01-02", false); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected mem cache hit on second call, got %d fetches", calls)
	}
}

func TestResultsStore_FetchErrorPropagates(t *testing.T) {
	fetch := func(date string) (*ResultsResponse, error) {
		return nil, fmt.Errorf("boom")
	}
	s := &resultsStore{fetch: fetch, mem: newMemCache[*ResultsResponse]()}
	if _, err := s.Get("2026-01-01", true); err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestResultsStore_DiskCacheSurvivesAsValidJSON(t *testing.T) {
	dir := t.TempDir()
	fetch := func(date string) (*ResultsResponse, error) {
		return &ResultsResponse{Date: date, Word: "board", Results: []ResultEntry{{Username: "alice", NumGuesses: 3, Solved: true, Completed: true}}}, nil
	}
	s := &resultsStore{fetch: fetch, mem: newMemCache[*ResultsResponse](), dir: dir}
	if _, err := s.Get("2026-02-01", true); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "2026-02-01.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got ResultsResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("cache file is not valid ResultsResponse JSON: %v", err)
	}
	if got.Word != "board" || len(got.Results) != 1 || got.Results[0].Username != "alice" {
		t.Fatalf("unexpected disk-cached content: %+v", got)
	}
}

func TestStatsStore_CachesAcrossCalls(t *testing.T) {
	calls := 0
	fetch := func(username string) (*UserStatsResponse, error) {
		calls++
		return &UserStatsResponse{Username: username, CurrentStreak: 3}, nil
	}
	s := &statsStore{fetch: fetch, mem: newMemCache[*UserStatsResponse]()}

	r1, err := s.Get("alice")
	if err != nil {
		t.Fatal(err)
	}
	if r1.CurrentStreak != 3 {
		t.Fatalf("expected streak 3, got %d", r1.CurrentStreak)
	}
	if _, err := s.Get("alice"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected cache hit on second call, got %d fetches", calls)
	}
}

func TestStatsStore_DifferentUsersDontShareCache(t *testing.T) {
	calls := map[string]int{}
	fetch := func(username string) (*UserStatsResponse, error) {
		calls[username]++
		return &UserStatsResponse{Username: username}, nil
	}
	s := &statsStore{fetch: fetch, mem: newMemCache[*UserStatsResponse]()}

	if _, err := s.Get("alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("bob"); err != nil {
		t.Fatal(err)
	}
	if calls["alice"] != 1 || calls["bob"] != 1 {
		t.Fatalf("expected 1 fetch each, got %+v", calls)
	}
}

func TestStatsStore_FetchErrorPropagates(t *testing.T) {
	fetch := func(username string) (*UserStatsResponse, error) {
		return nil, fmt.Errorf("boom")
	}
	s := &statsStore{fetch: fetch, mem: newMemCache[*UserStatsResponse]()}
	if _, err := s.Get("alice"); err == nil {
		t.Fatal("expected error to propagate")
	}
}
