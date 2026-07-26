package main

import "testing"

func TestRenderGuessGrid_RowCount(t *testing.T) {
	words := []string{"crane", "board"}
	patterns := []string{"xxxxx", "ggggg"}
	out := renderGuessGrid(words, patterns, 6)

	lines := 1
	for _, r := range out {
		if r == '\n' {
			lines++
		}
	}
	// Each row is a rounded-border cell 3 lines tall (top border, content, bottom
	// border), so 6 rows = 18 total lines.
	if lines != 18 {
		t.Errorf("expected 18 total lines for 6 rows, got %d", lines)
	}
}

func TestRenderGuessGrid_EmptyRowsForMissingGuesses(t *testing.T) {
	words := []string{"crane"}
	patterns := []string{"ggggg"}
	out := renderGuessGrid(words, patterns, 6)

	if out == "" {
		t.Fatal("expected non-empty output")
	}
	// Sanity: with only 1 real guess, the grid should still produce 6 rows
	// total (18 lines, 3 lines per bordered row).
	lineCount := 1
	for _, r := range out {
		if r == '\n' {
			lineCount++
		}
	}
	if lineCount != 18 {
		t.Errorf("expected 18 total lines, got %d", lineCount)
	}
}
