package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	rollingWindow = 10
	distBarWidth  = 30
)

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

// renderDistribution draws a horizontal bar per outcome from
// [fails, won on 1st, won on 2nd, won on 3rd], ordered 1, 2, 3, Fail.
func renderDistribution(dist []int) string {
	if len(dist) != 4 {
		return ""
	}
	rows := []struct {
		label string
		count int
		style lipgloss.Style
	}{
		{"1   ", dist[1], greenStyle},
		{"2   ", dist[2], accentStyle},
		{"3   ", dist[3], accentStyle},
		{"Fail", dist[0], redStyle},
	}
	max := 0
	total := 0
	for _, r := range rows {
		total += r.count
		if r.count > max {
			max = r.count
		}
	}
	var sb strings.Builder
	for _, r := range rows {
		width := 0
		if max > 0 {
			width = r.count * distBarWidth / max
		}
		if r.count > 0 && width == 0 {
			width = 1
		}
		pct := 0.0
		if total > 0 {
			pct = float64(r.count) / float64(total) * 100
		}
		sb.WriteString(fmt.Sprintf("  %s %s %d (%.0f%%)\n", r.label, r.style.Render(strings.Repeat("█", width)), r.count, pct))
	}
	return sb.String()
}

// rollingWinAverages returns, for each win in history order, the mean number
// of attempts across that win and the previous window-1 wins. Losses are
// skipped, matching the website's "Rolling Avg Attempts (wins)" chart.
func rollingWinAverages(history []HistoryGame, window int) []float64 {
	var wins []int
	for _, g := range history {
		if g.Won {
			wins = append(wins, g.Attempts)
		}
	}
	out := make([]float64, len(wins))
	for i := range wins {
		from := 0
		if window > 0 && i-window+1 > 0 {
			from = i - window + 1
		}
		sum := 0
		for _, a := range wins[from : i+1] {
			sum += a
		}
		out[i] = float64(sum) / float64(i+1-from)
	}
	return out
}

// renderRollingSparkline draws the rolling average on the fixed 1–3 attempt
// scale as one block per win (most recent 40 shown), plus the latest value.
func renderRollingSparkline(history []HistoryGame, window int) string {
	avgs := rollingWinAverages(history, window)
	if len(avgs) == 0 {
		return ""
	}
	const maxPoints = 40
	shown := avgs
	if len(shown) > maxPoints {
		shown = shown[len(shown)-maxPoints:]
	}
	var sb strings.Builder
	for _, v := range shown {
		idx := int((v - 1) / 2 * float64(len(sparkRunes)-1))
		if idx < 0 {
			idx = 0
		}
		if idx > len(sparkRunes)-1 {
			idx = len(sparkRunes) - 1
		}
		sb.WriteRune(sparkRunes[idx])
	}
	return fmt.Sprintf("  %s  %.2f", accentStyle.Render(sb.String()), avgs[len(avgs)-1])
}
