package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://rhysfuller.com"

type APIClient struct {
	secret   string
	username string
	http     *http.Client
}

func newAPIClient(secret, username string) *APIClient {
	return &APIClient{
		secret:   secret,
		username: username,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Response types

type GuessEntry struct {
	Word    string `json:"word"`
	Pattern string `json:"pattern"`
}

type GuessResponse struct {
	Pattern   string       `json:"pattern"`
	Guesses   []GuessEntry `json:"guesses"`
	Completed bool         `json:"completed"`
	Won       bool         `json:"won"`
	Word      string       `json:"word"`
	Error     string       `json:"error"`
}

type GameState struct {
	Username  string       `json:"username"`
	Date      string       `json:"date"`
	Guesses   []GuessEntry `json:"guesses"`
	Completed bool         `json:"completed"`
	Won       bool         `json:"won"`
}

type LeaderboardEntry struct {
	Username      string  `json:"username"`
	TotalPoints   int     `json:"total_points"`
	GamesPlayed   int     `json:"games_played"`
	GamesWon      int     `json:"games_won"`
	AvgTries      float64 `json:"avg_tries"`
	CurrentStreak int     `json:"current_streak"`
	BestStreak    int     `json:"best_streak"`
}

type LeaderboardResponse struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
}

type UserStatsResponse struct {
	Username      string  `json:"username"`
	TotalPoints   int     `json:"total_points"`
	GamesPlayed   int     `json:"games_played"`
	GamesWon      int     `json:"games_won"`
	AvgTries      float64 `json:"avg_tries"`
	CurrentStreak int     `json:"current_streak"`
	BestStreak    int     `json:"best_streak"`
}

type ResultEntry struct {
	Username   string   `json:"username"`
	Guesses    []string `json:"guesses"`
	Patterns   []string `json:"patterns"`
	Solved     bool     `json:"solved"`
	Completed  bool     `json:"completed"`
	NumGuesses int      `json:"num_guesses"`
}

type ResultsResponse struct {
	Date    string        `json:"date"`
	Word    string        `json:"word"`
	Results []ResultEntry `json:"results"`
}

// API methods

func (c *APIClient) getJSON(path string, params url.Values, result any) error {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Wordle-Secret", c.secret)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, result)
}

func (c *APIClient) SubmitGuess(guess string) (*GuessResponse, error) {
	payload := map[string]string{
		"secret_key": c.secret,
		"username":   c.username,
		"guess":      guess,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+"/api/wordle/guess/", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result GuessResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *APIClient) GetGameState(username, date string) (*GameState, error) {
	var state GameState
	params := url.Values{"username": {username}, "date": {date}}
	err := c.getJSON("/api/wordle/state/", params, &state)
	return &state, err
}

func (c *APIClient) GetLeaderboard() (*LeaderboardResponse, error) {
	var lb LeaderboardResponse
	err := c.getJSON("/api/wordle/leaderboard/", nil, &lb)
	return &lb, err
}

func (c *APIClient) GetUserStats(username string) (*UserStatsResponse, error) {
	var stats UserStatsResponse
	params := url.Values{"username": {username}}
	err := c.getJSON("/api/wordle/user-stats/", params, &stats)
	return &stats, err
}

func (c *APIClient) GetResults(date string) (*ResultsResponse, error) {
	var results ResultsResponse
	params := url.Values{"date": {date}}
	err := c.getJSON("/api/wordle/results/", params, &results)
	return &results, err
}
