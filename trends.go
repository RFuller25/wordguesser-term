package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dayStat struct {
	date        string
	avgScore    float64
	playerCount int
}

type playerPoint struct {
	date  string
	value float64
}

type guessBucket struct {
	label string
	count int
}

func scoreOf(r ResultEntry) int {
	if r.Solved {
		return 7 - r.NumGuesses
	}
	return 0
}

func addDays(date string, n int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

func buildDayStats(dates []string, byDate map[string]*ResultsResponse) []dayStat {
	var out []dayStat
	for _, d := range dates {
		resp := byDate[d]
		if resp == nil {
			continue
		}
		total, completed := 0, 0
		for _, r := range resp.Results {
			if r.Completed {
				total += scoreOf(r)
				completed++
			}
		}
		avg := 0.0
		if completed > 0 {
			avg = float64(total) / float64(completed)
		}
		out = append(out, dayStat{date: d, avgScore: avg, playerCount: len(resp.Results)})
	}
	return out
}

type scoredDay struct {
	date  string
	score int
}

// rollingPlayerAverages returns, for each username, a chronologically-ordered
// slice of (date, rollingAvg) points, one per day that player completed a
// game. rollingAvg is the mean score over that player's completions within
// the trailing 7 calendar days ending on that date — days they didn't play
// are simply absent, not zero-filled.
func rollingPlayerAverages(dates []string, byDate map[string]*ResultsResponse) map[string][]playerPoint {
	raw := map[string][]scoredDay{}
	for _, d := range dates {
		resp := byDate[d]
		if resp == nil {
			continue
		}
		for _, r := range resp.Results {
			if r.Completed {
				username := strings.ToLower(r.Username)
				raw[username] = append(raw[username], scoredDay{date: d, score: scoreOf(r)})
			}
		}
	}

	out := map[string][]playerPoint{}
	for username, entries := range raw {
		for i, e := range entries {
			cutoff := addDays(e.date, -6)
			sum, count := 0, 0
			for j := i; j >= 0 && entries[j].date >= cutoff; j-- {
				sum += entries[j].score
				count++
			}
			out[username] = append(out[username], playerPoint{date: e.date, value: float64(sum) / float64(count)})
		}
	}
	return out
}

func buildDistribution(dates []string, byDate map[string]*ResultsResponse) []guessBucket {
	order := []string{"1", "2", "3", "4", "5", "6", "Fail"}
	counts := map[string]int{}
	for _, d := range dates {
		resp := byDate[d]
		if resp == nil {
			continue
		}
		for _, r := range resp.Results {
			if !r.Completed {
				continue
			}
			if r.Solved {
				counts[fmt.Sprintf("%d", r.NumGuesses)]++
			} else {
				counts["Fail"]++
			}
		}
	}
	out := make([]guessBucket, 0, len(order))
	for _, label := range order {
		out = append(out, guessBucket{label: label, count: counts[label]})
	}
	return out
}

type wordStat struct {
	solved int
	total  int
}

func hardestWord(dates []string, byDate map[string]*ResultsResponse) (word string, solveRate float64, ok bool) {
	stats := map[string]*wordStat{}
	for _, d := range dates {
		resp := byDate[d]
		if resp == nil || resp.Word == "" {
			continue
		}
		w := strings.ToUpper(resp.Word)
		st := stats[w]
		if st == nil {
			st = &wordStat{}
			stats[w] = st
		}
		for _, r := range resp.Results {
			if !r.Completed {
				continue
			}
			st.total++
			if r.Solved {
				st.solved++
			}
		}
	}

	best := ""
	bestRate := 2.0 // sentinel above any real rate (max 1.0)
	for w, st := range stats {
		if st.total == 0 {
			continue
		}
		rate := float64(st.solved) / float64(st.total)
		if rate < bestRate || (rate == bestRate && w < best) {
			bestRate = rate
			best = w
		}
	}
	if best == "" {
		return "", 0, false
	}
	return best, bestRate, true
}

func bestDay(days []dayStat) (date string, avgScore float64, ok bool) {
	best := -1.0
	bestDate := ""
	for _, d := range days {
		if d.avgScore > best {
			best = d.avgScore
			bestDate = d.date
		}
	}
	if bestDate == "" {
		return "", 0, false
	}
	return bestDate, best, true
}

const trendsWindowDays = 30

type trendsModel struct {
	resultsStore *resultsStore

	windowEnd time.Time

	days         []dayStat
	perPlayer    map[string][]playerPoint
	playerNames  []string
	distribution []guessBucket

	hardestWordText string
	hardestRate     float64
	haveHardest     bool
	bestDayDate     string
	bestDayScore    float64
	haveBestDay     bool

	loading bool
	spinner spinner.Model
	errMsg  string

	panel        int // 0 = Overview, 1 = Player, 2 = Distribution
	playerMode   int // 0 = single, 1 = all-overlay
	playerCursor int

	width, height int
}

type trendsMsg struct {
	days         []dayStat
	perPlayer    map[string][]playerPoint
	playerNames  []string
	distribution []guessBucket
	hardestWord  string
	hardestRate  float64
	haveHardest  bool
	bestDayDate  string
	bestDayScore float64
	haveBestDay  bool
}

func newTrendsModel(store *resultsStore) trendsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle
	return trendsModel{
		resultsStore: store,
		windowEnd:    time.Now().In(chicagoTZ()),
		spinner:      s,
	}
}

func (m trendsModel) Init() tea.Cmd { return nil }

func windowDates(end time.Time) []string {
	dates := make([]string, trendsWindowDays)
	for i := 0; i < trendsWindowDays; i++ {
		d := end.AddDate(0, 0, -(trendsWindowDays-1-i))
		dates[i] = d.Format("2006-01-02")
	}
	return dates
}

func fetchResultsWindow(store *resultsStore, dates []string, today string) map[string]*ResultsResponse {
	type item struct {
		date string
		resp *ResultsResponse
	}
	sem := make(chan struct{}, 5)
	out := make(chan item, len(dates))
	var wg sync.WaitGroup
	for _, d := range dates {
		wg.Add(1)
		go func(date string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			resp, err := store.Get(date, isFinalDate(date, today))
			if err != nil {
				resp = nil
			}
			out <- item{date: date, resp: resp}
		}(d)
	}
	go func() {
		wg.Wait()
		close(out)
	}()

	byDate := make(map[string]*ResultsResponse, len(dates))
	for it := range out {
		byDate[it.date] = it.resp
	}
	return byDate
}

func (m *trendsModel) Fetch() tea.Cmd {
	m.loading = true
	dates := windowDates(m.windowEnd)
	today := time.Now().In(chicagoTZ()).Format("2006-01-02")
	store := m.resultsStore
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		byDate := fetchResultsWindow(store, dates, today)
		days := buildDayStats(dates, byDate)
		perPlayer := rollingPlayerAverages(dates, byDate)
		names := make([]string, 0, len(perPlayer))
		for n := range perPlayer {
			names = append(names, n)
		}
		sort.Strings(names)
		dist := buildDistribution(dates, byDate)
		word, rate, haveHardest := hardestWord(dates, byDate)
		bDate, bScore, haveBest := bestDay(days)
		return trendsMsg{
			days: days, perPlayer: perPlayer, playerNames: names,
			distribution: dist,
			hardestWord:  word, hardestRate: rate, haveHardest: haveHardest,
			bestDayDate: bDate, bestDayScore: bScore, haveBestDay: haveBest,
		}
	})
}

func (m trendsModel) Update(msg tea.Msg) (trendsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case trendsMsg:
		m.loading = false
		m.days = msg.days
		m.perPlayer = msg.perPlayer
		m.playerNames = msg.playerNames
		m.distribution = msg.distribution
		m.hardestWordText = msg.hardestWord
		m.hardestRate = msg.hardestRate
		m.haveHardest = msg.haveHardest
		m.bestDayDate = msg.bestDayDate
		m.bestDayScore = msg.bestDayScore
		m.haveBestDay = msg.haveBestDay
		if m.playerCursor >= len(m.playerNames) {
			m.playerCursor = 0
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyLeft:
			m.windowEnd = m.windowEnd.AddDate(0, 0, -1)
			return m, m.Fetch()
		case tea.KeyRight:
			tomorrow := m.windowEnd.AddDate(0, 0, 1)
			today := time.Now().In(chicagoTZ())
			if !tomorrow.After(today) {
				m.windowEnd = tomorrow
				return m, m.Fetch()
			}
		case tea.KeyEnter:
			if m.panel == 1 {
				m.playerMode = 1 - m.playerMode
			}
		case tea.KeyUp:
			if m.panel == 1 && m.playerMode == 0 && m.playerCursor > 0 {
				m.playerCursor--
			}
		case tea.KeyDown:
			if m.panel == 1 && m.playerMode == 0 && m.playerCursor < len(m.playerNames)-1 {
				m.playerCursor++
			}
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "[":
				m.panel = (m.panel + 2) % 3
			case "]":
				m.panel = (m.panel + 1) % 3
			}
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *trendsModel) Resize(w, h int) {
	m.width = w
	m.height = h
}

func (m trendsModel) chartSize() (int, int) {
	w := m.width - 8
	if w > 70 {
		w = 70
	}
	if w < 20 {
		w = 20
	}
	return w, 8
}

func toTimePoints(points []playerPoint) []timeserieslinechart.TimePoint {
	out := make([]timeserieslinechart.TimePoint, 0, len(points))
	for _, p := range points {
		t, err := time.Parse("2006-01-02", p.date)
		if err != nil {
			continue
		}
		out = append(out, timeserieslinechart.TimePoint{Time: t, Value: p.value})
	}
	return out
}

func dayStatsToTimePoints(days []dayStat, score bool) []timeserieslinechart.TimePoint {
	out := make([]timeserieslinechart.TimePoint, 0, len(days))
	for _, d := range days {
		t, err := time.Parse("2006-01-02", d.date)
		if err != nil {
			continue
		}
		v := d.avgScore
		if !score {
			v = float64(d.playerCount)
		}
		out = append(out, timeserieslinechart.TimePoint{Time: t, Value: v})
	}
	return out
}

var trendsPalette = []lipgloss.Color{
	lipgloss.Color("#22c55e"), // green
	lipgloss.Color("#eab308"), // yellow
	lipgloss.Color("#6366f1"), // indigo
	lipgloss.Color("#ec4899"), // pink
	lipgloss.Color("#06b6d4"), // cyan
	lipgloss.Color("#f97316"), // orange
}

func (m trendsModel) windowStart() time.Time {
	return m.windowEnd.AddDate(0, 0, -(trendsWindowDays - 1))
}

func (m trendsModel) renderOverview() string {
	w, h := m.chartSize()
	var sb strings.Builder

	sb.WriteString(dimStyle.Render("  Avg score / day") + "\n")
	scoreChart := timeserieslinechart.New(w, h,
		timeserieslinechart.WithTimeRange(m.windowStart(), m.windowEnd),
		timeserieslinechart.WithTimeSeries(dayStatsToTimePoints(m.days, true)),
	)
	scoreChart.SetDataSetStyle(timeserieslinechart.DefaultDataSetName, lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")))
	scoreChart.DrawBraille()
	sb.WriteString(scoreChart.View() + "\n\n")

	sb.WriteString(dimStyle.Render("  Players / day") + "\n")
	playersChart := timeserieslinechart.New(w, h,
		timeserieslinechart.WithTimeRange(m.windowStart(), m.windowEnd),
		timeserieslinechart.WithTimeSeries(dayStatsToTimePoints(m.days, false)),
	)
	playersChart.SetDataSetStyle(timeserieslinechart.DefaultDataSetName, lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1")))
	playersChart.DrawBraille()
	sb.WriteString(playersChart.View() + "\n")

	return sb.String()
}

func (m trendsModel) renderPlayerPanel() string {
	w, h := m.chartSize()
	var sb strings.Builder

	if len(m.playerNames) == 0 {
		sb.WriteString(dimStyle.Render("  No player data in this window.") + "\n")
		return sb.String()
	}

	if m.playerMode == 0 {
		cursor := m.playerCursor
		if cursor >= len(m.playerNames) {
			cursor = 0
		}
		username := m.playerNames[cursor]
		sb.WriteString(fmt.Sprintf("  %s  ", accentStyle.Bold(true).Render(username)))
		sb.WriteString(dimStyle.Render("(↑/↓ change player)") + "\n")

		chart := timeserieslinechart.New(w, h,
			timeserieslinechart.WithTimeRange(m.windowStart(), m.windowEnd),
			timeserieslinechart.WithTimeSeries(toTimePoints(m.perPlayer[username])),
		)
		chart.SetDataSetStyle(timeserieslinechart.DefaultDataSetName, lipgloss.NewStyle().Foreground(trendsPalette[cursor%len(trendsPalette)]))
		chart.DrawBraille()
		sb.WriteString(chart.View() + "\n")
		return sb.String()
	}

	sb.WriteString(dimStyle.Render("  All players (Enter: single player view)") + "\n")
	opts := []timeserieslinechart.Option{
		timeserieslinechart.WithTimeRange(m.windowStart(), m.windowEnd),
	}
	for _, name := range m.playerNames {
		opts = append(opts, timeserieslinechart.WithDataSetTimeSeries(name, toTimePoints(m.perPlayer[name])))
	}
	chart := timeserieslinechart.New(w, h, opts...)
	var legend strings.Builder
	for i, name := range m.playerNames {
		style := lipgloss.NewStyle().Foreground(trendsPalette[i%len(trendsPalette)])
		chart.SetDataSetStyle(name, style)
		legend.WriteString(style.Render("■ "+name) + "  ")
	}
	chart.DrawBrailleAll()
	sb.WriteString(chart.View() + "\n")
	sb.WriteString(legend.String() + "\n")
	return sb.String()
}

func (m trendsModel) renderDistribution() string {
	w, h := m.chartSize()
	var sb strings.Builder

	data := make([]barchart.BarData, 0, len(m.distribution))
	for _, b := range m.distribution {
		style := greenStyle
		if b.label == "Fail" {
			style = grayStyle
		}
		data = append(data, barchart.BarData{
			Label:  b.label,
			Values: []barchart.BarValue{{Name: b.label, Value: float64(b.count), Style: style}},
		})
	}
	chart := barchart.New(w, h)
	chart.PushAll(data)
	chart.Draw()
	sb.WriteString(chart.View() + "\n\n")

	if m.haveHardest {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  Hardest word: %s (%.0f%% solve rate)", m.hardestWordText, m.hardestRate*100)) + "\n")
	}
	if m.haveBestDay {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  Best day: %s (avg score %.1f)", m.bestDayDate, m.bestDayScore)) + "\n")
	}
	return sb.String()
}

func (m trendsModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1")).MarginBottom(1)
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Trends") + "\n\n")

	dateStr := m.windowEnd.Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("  ◀  window ending %s  ▶\n", accentStyle.Bold(true).Render(dateStr)))
	sb.WriteString(dimStyle.Render("  ← / → shift window | [ / ] switch panel") + "\n\n")

	if m.loading {
		sb.WriteString(m.spinner.View() + " Loading trends...\n")
		return sb.String()
	}

	panelNames := []string{"Overview", "Player Trend", "Distribution"}
	sb.WriteString(accentStyle.Bold(true).Render("  "+panelNames[m.panel]) + "\n\n")

	switch m.panel {
	case 0:
		sb.WriteString(m.renderOverview())
	case 1:
		sb.WriteString(m.renderPlayerPanel())
	case 2:
		sb.WriteString(m.renderDistribution())
	}

	return sb.String()
}
