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

type TodayResponse struct {
	Date    string   `json:"date"`
	QuoteID int      `json:"quote_id"`
	Quote   string   `json:"quote"`
	Options []string `json:"options"`
}

type GuessResponse struct {
	Correct           bool   `json:"correct"`
	Completed         bool   `json:"completed"`
	Won               bool   `json:"won"`
	Attempts          []int  `json:"attempts"`
	AttemptsRemaining int    `json:"attempts_remaining"`
	Score             int    `json:"score"`
	CurrentStreak     int    `json:"current_streak"`
	BestStreak        int    `json:"best_streak"`
	CorrectOption     int    `json:"correct_option"`
	BrokenStreak      int    `json:"broken_streak"`
	Error             string `json:"error"`
}

type StateResponse struct {
	Username  string `json:"username"`
	Date      string `json:"date"`
	Attempts  []int  `json:"attempts"`
	Completed bool   `json:"completed"`
	Won       bool   `json:"won"`
	Score     int    `json:"score"`
}

type ResultEntry struct {
	Username      string `json:"username"`
	Attempts      []int  `json:"attempts"`
	Won           bool   `json:"won"`
	Score         int    `json:"score"`
	CurrentStreak int    `json:"current_streak"`
}

type ResultsResponse struct {
	Date    string        `json:"date"`
	Quote   string        `json:"quote"`
	Author  string        `json:"author"`
	Results []ResultEntry `json:"results"`
}

type LeaderboardEntry struct {
	Username      string  `json:"username"`
	GamesPlayed   int     `json:"games_played"`
	GamesWon      int     `json:"games_won"`
	TotalScore    int     `json:"total_score"`
	AvgAttempts   float64 `json:"avg_attempts"`
	CurrentStreak int     `json:"current_streak"`
	BestStreak    int     `json:"best_streak"`
}

type LeaderboardResponse struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
}

type UserStatsResponse struct {
	Username      string `json:"username"`
	Played        int    `json:"played"`
	Won           int    `json:"won"`
	CurrentStreak int    `json:"current_streak"`
	BestStreak    int    `json:"best_streak"`
}

func (c *APIClient) doGet(path string, params url.Values) (int, []byte, error) {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Wordle-Secret", c.secret)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("read error: %w", err)
	}
	return resp.StatusCode, body, nil
}

// getJSON treats any non-200 status, including 404, as an error.
func (c *APIClient) getJSON(path string, params url.Values, result any) error {
	status, body, err := c.doGet(path, params)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("API error %d: %s", status, string(body))
	}
	return json.Unmarshal(body, result)
}

// getJSONOptional treats a 404 as "not found" (returns found=false, err=nil)
// rather than an error, for endpoints where "no data yet" is expected.
func (c *APIClient) getJSONOptional(path string, params url.Values, result any) (bool, error) {
	status, body, err := c.doGet(path, params)
	if err != nil {
		return false, err
	}
	if status == 404 {
		return false, nil
	}
	if status != 200 {
		return false, fmt.Errorf("API error %d: %s", status, string(body))
	}
	if err := json.Unmarshal(body, result); err != nil {
		return false, err
	}
	return true, nil
}

func (c *APIClient) GetToday(date string) (*TodayResponse, error) {
	var today TodayResponse
	params := url.Values{}
	if date != "" {
		params.Set("date", date)
	}
	if err := c.getJSON("/api/quotes/today/", params, &today); err != nil {
		return nil, err
	}
	return &today, nil
}

func (c *APIClient) GetState(username, date string) (*StateResponse, error) {
	var state StateResponse
	params := url.Values{"username": {username}, "date": {date}}
	found, err := c.getJSONOptional("/api/quotes/state/", params, &state)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &state, nil
}

func (c *APIClient) GetResults(date string) (*ResultsResponse, error) {
	var results ResultsResponse
	params := url.Values{"date": {date}}
	if err := c.getJSON("/api/quotes/results/", params, &results); err != nil {
		return nil, err
	}
	return &results, nil
}

func (c *APIClient) GetLeaderboard() (*LeaderboardResponse, error) {
	var lb LeaderboardResponse
	if err := c.getJSON("/api/quotes/leaderboard/", nil, &lb); err != nil {
		return nil, err
	}
	return &lb, nil
}

func (c *APIClient) GetUserStats(username string) (*UserStatsResponse, error) {
	var stats UserStatsResponse
	params := url.Values{"username": {username}}
	found, err := c.getJSONOptional("/api/quotes/user-stats/", params, &stats)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &stats, nil
}

func (c *APIClient) SubmitGuess(guess int, date string) (*GuessResponse, error) {
	payload := map[string]any{
		"secret_key": c.secret,
		"username":   c.username,
		"guess":      guess,
	}
	if date != "" {
		payload["date"] = date
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+"/api/quotes/guess/", bytes.NewReader(data))
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
