package main

import (
	"math"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type bubble struct {
	x, y   float64
	speedY float64
	wobble float64
	phase  float64
	char   string
	color  lipgloss.Style
}

type bubbleField struct {
	bubbles     []bubble
	width       int
	height      int
	targetCount int
	tick        int
}

var bubbleChars = []string{"·", "•", "○", "◦", "°", "◯"}

var bubbleColors = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#818cf8")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#7c3aed")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#3b82f6")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#8b5cf6")),
}

func newBubbleField(width, height int) bubbleField {
	bf := bubbleField{
		width:       width,
		height:      height,
		targetCount: 20,
	}
	for i := 0; i < bf.targetCount; i++ {
		bf.bubbles = append(bf.bubbles, bf.spawnBubble(true))
	}
	return bf
}

func (bf *bubbleField) spawnBubble(randomY bool) bubble {
	y := float64(bf.height)
	if randomY {
		y = rand.Float64() * float64(bf.height)
	}
	return bubble{
		x:      rand.Float64() * float64(bf.width),
		y:      y,
		speedY: 0.3 + rand.Float64()*1.2,
		wobble: 0.5 + rand.Float64()*2.0,
		phase:  rand.Float64() * math.Pi * 2,
		char:   bubbleChars[rand.Intn(len(bubbleChars))],
		color:  bubbleColors[rand.Intn(len(bubbleColors))],
	}
}

func (bf *bubbleField) update() {
	bf.tick++
	alive := bf.bubbles[:0]
	for i := range bf.bubbles {
		b := &bf.bubbles[i]
		b.y -= b.speedY
		b.x += math.Sin(b.phase+float64(bf.tick)*0.05) * b.wobble * 0.15
		if b.y > -1 {
			alive = append(alive, *b)
		}
	}
	bf.bubbles = alive

	for len(bf.bubbles) < bf.targetCount {
		bf.bubbles = append(bf.bubbles, bf.spawnBubble(false))
	}
}

func (bf *bubbleField) resize(width, height int) {
	bf.width = width
	bf.height = height
}

func (bf *bubbleField) view() string {
	if bf.width == 0 || bf.height == 0 {
		return ""
	}

	grid := make([][]string, bf.height)
	for i := range grid {
		grid[i] = make([]string, bf.width)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}

	for _, b := range bf.bubbles {
		ix := int(math.Round(b.x))
		iy := int(math.Round(b.y))
		if ix >= 0 && ix < bf.width && iy >= 0 && iy < bf.height {
			grid[iy][ix] = b.color.Render(b.char)
		}
	}

	var sb strings.Builder
	for i, row := range grid {
		for _, cell := range row {
			sb.WriteString(cell)
		}
		if i < len(grid)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
