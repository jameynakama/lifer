package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages sent from worker goroutines to the TUI via p.Send.

type speciesStartedMsg struct {
	workerID int
	code     string
	name     string
}

type fetchStartedMsg struct {
	workerID int
	key      string
}

type uploadStartedMsg struct {
	workerID int
	key      string
}

type uploadDoneMsg struct {
	workerID int
	key      string
	err      error
}

type speciesDoneMsg struct {
	workerID   int
	code       string
	name       string
	recordings int
	images     int
	failures   []string
}

type allDoneMsg struct{}

type tickMsg time.Time

// Model state

type uploadStatus int

const (
	statusWaiting uploadStatus = iota
	statusFetching
	statusUploading
	statusDone
	statusFailed
)

type uploadItem struct {
	key    string
	status uploadStatus
	err    error
}

type workerState struct {
	name    string
	uploads []uploadItem
}

type completedItem struct {
	name       string
	recordings int
	images     int
	failed     bool
}

type model struct {
	workers   []workerState
	completed []completedItem
	done      int
	total     int
	start     time.Time
	width     int
	height    int
	progress  progress.Model
}

var (
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleFailed  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
	styleSubItem = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleYellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleDivider = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleElapsed = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleCounter = lipgloss.NewStyle().Bold(true)
)

func newModel(total, numWorkers int) model {
	return model{
		workers:  make([]workerState, numWorkers),
		total:    total,
		start:    time.Now(),
		progress: progress.New(progress.WithDefaultGradient()),
	}
}

// appendOrUpdateUpload finds the upload item with key and sets its status,
// or appends a new item if not found.
func appendOrUpdateUpload(items []uploadItem, key string, status uploadStatus, err error) []uploadItem {
	for i := range items {
		if items[i].key == key {
			items[i].status = status
			items[i].err = err
			return items
		}
	}
	return append(items, uploadItem{key: key, status: status, err: err})
}

func (m model) Init() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case speciesStartedMsg:
		m.workers[msg.workerID].name = msg.name
		m.workers[msg.workerID].uploads = nil
		return m, nil

	case fetchStartedMsg:
		m.workers[msg.workerID].uploads = appendOrUpdateUpload(
			m.workers[msg.workerID].uploads, msg.key, statusFetching, nil)
		return m, nil

	case uploadStartedMsg:
		m.workers[msg.workerID].uploads = appendOrUpdateUpload(
			m.workers[msg.workerID].uploads, msg.key, statusUploading, nil)
		return m, nil

	case uploadDoneMsg:
		status := statusDone
		if msg.err != nil {
			status = statusFailed
		}
		m.workers[msg.workerID].uploads = appendOrUpdateUpload(
			m.workers[msg.workerID].uploads, msg.key, status, msg.err)
		return m, nil

	case speciesDoneMsg:
		m.done++
		m.workers[msg.workerID] = workerState{}
		failed := len(msg.failures) > 0 || msg.recordings == 0 || msg.images == 0
		m.completed = append(m.completed, completedItem{
			name:       msg.name,
			recordings: msg.recordings,
			images:     msg.images,
			failed:     failed,
		})
		cmd := m.progress.SetPercent(float64(m.done) / float64(m.total))
		return m, cmd

	case allDoneMsg:
		return m, tea.Quit

	case tickMsg:
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })

	case progress.FrameMsg:
		pm, cmd := m.progress.Update(msg)
		if newPM, ok := pm.(progress.Model); ok {
			m.progress = newPM
		}
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 20
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	if m.total == 0 {
		return ""
	}
	var b strings.Builder

	// Header
	elapsed := time.Since(m.start).Round(time.Second)
	h := int(elapsed.Hours())
	min := int(elapsed.Minutes()) % 60
	sec := int(elapsed.Seconds()) % 60
	elapsedStr := fmt.Sprintf("%d:%02d:%02d", h, min, sec)
	counter := styleCounter.Render(fmt.Sprintf("%d/%d", m.done, m.total))
	bar := m.progress.View()
	b.WriteString(fmt.Sprintf("  %s  %s  %s\n\n", counter, bar, styleElapsed.Render(elapsedStr)))

	// Completed species -- show last N that fit above the divider + workers
	workerH := 0
	for _, w := range m.workers {
		if w.name != "" {
			workerH += 1 + len(w.uploads)
		}
	}
	headerLines := 2
	dividerLine := 1
	bottomPad := 1
	availableForCompleted := m.height - headerLines - dividerLine - workerH - bottomPad
	if availableForCompleted < 0 {
		availableForCompleted = 0
	}
	start := 0
	if len(m.completed) > availableForCompleted {
		start = len(m.completed) - availableForCompleted
	}
	for _, c := range m.completed[start:] {
		icon := styleSuccess.Render("✓")
		nameStr := c.name
		detail := fmt.Sprintf("%d rec  %d img", c.recordings, c.images)
		if c.failed {
			icon = styleFailed.Render("✗")
			nameStr = styleFailed.Render(c.name)
			detail = styleFailed.Render(detail)
		}
		b.WriteString(fmt.Sprintf("  %s %-30s  %s\n", icon, nameStr, detail))
	}

	// Divider
	divider := styleDivider.Render(strings.Repeat("─", max(m.width-2, 40)))
	b.WriteString(fmt.Sprintf("  %s\n", divider))

	// Active workers
	for i, w := range m.workers {
		if w.name == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, styleBold.Render(w.name)))
		for _, u := range w.uploads {
			symbol, style := uploadSymbol(u.status)
			b.WriteString(fmt.Sprintf("      %s %-45s  %s\n",
				styleSubItem.Render("·"),
				styleSubItem.Render(shortKey(u.key)),
				style.Render(symbol),
			))
		}
	}

	return b.String()
}

func uploadSymbol(s uploadStatus) (string, lipgloss.Style) {
	switch s {
	case statusFetching:
		return "↓ fetching", styleSubItem
	case statusUploading:
		return "↑ uploading", styleYellow
	case statusDone:
		return "✓", styleSuccess
	case statusFailed:
		return "✗", styleFailed
	default:
		return "· waiting", styleSubItem
	}
}

// shortKey abbreviates "recordings/code/file.mp3" → "rec/code/file.mp3" for display.
func shortKey(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) == 3 {
		return parts[0][:3] + "/" + parts[1] + "/" + parts[2]
	}
	return key
}
