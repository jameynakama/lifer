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

// Stub Init/Update/View so the package compiles; replaced in Task 3 and Task 7.

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m model) View() string { return "" }
