package main

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
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
	return tea.Batch(
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) }),
	)
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

func (m model) View() string { return "" }
