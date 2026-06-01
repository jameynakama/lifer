package main

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func baseModel(t *testing.T) model {
	t.Helper()
	return newModel(10, 3)
}

func TestUpdate_SpeciesStarted(t *testing.T) {
	m := baseModel(t)
	next, _ := m.Update(speciesStartedMsg{workerID: 1, code: "amro", name: "American Robin"})
	got := next.(model)
	if got.workers[1].name != "American Robin" {
		t.Errorf("Should set worker name: got %q", got.workers[1].name)
	}
	if len(got.workers[1].uploads) != 0 {
		t.Errorf("Should clear uploads on new species")
	}
}

func TestUpdate_FetchStarted(t *testing.T) {
	m := baseModel(t)
	m.workers[0].name = "Bushtit"
	next, _ := m.Update(fetchStartedMsg{workerID: 0, key: "recordings/busti/XC1.mp3"})
	got := next.(model)
	if len(got.workers[0].uploads) != 1 {
		t.Fatalf("Should add one upload item")
	}
	if got.workers[0].uploads[0].status != statusFetching {
		t.Errorf("Should set status to fetching")
	}
}

func TestUpdate_UploadStarted(t *testing.T) {
	m := baseModel(t)
	m.workers[0].uploads = []uploadItem{{key: "recordings/busti/XC1.mp3", status: statusFetching}}
	next, _ := m.Update(uploadStartedMsg{workerID: 0, key: "recordings/busti/XC1.mp3"})
	got := next.(model)
	if got.workers[0].uploads[0].status != statusUploading {
		t.Errorf("Should update status to uploading")
	}
}

func TestUpdate_UploadDone_Success(t *testing.T) {
	m := baseModel(t)
	m.workers[2].uploads = []uploadItem{{key: "images/busti/ML1.jpg", status: statusUploading}}
	next, _ := m.Update(uploadDoneMsg{workerID: 2, key: "images/busti/ML1.jpg", err: nil})
	got := next.(model)
	if got.workers[2].uploads[0].status != statusDone {
		t.Errorf("Should set status to done")
	}
	if got.workers[2].uploads[0].err != nil {
		t.Errorf("Should have nil err")
	}
}

func TestUpdate_UploadDone_Error(t *testing.T) {
	m := baseModel(t)
	uploadErr := errors.New("timeout")
	m.workers[1].uploads = []uploadItem{{key: "recordings/busti/XC1.mp3", status: statusUploading}}
	next, _ := m.Update(uploadDoneMsg{workerID: 1, key: "recordings/busti/XC1.mp3", err: uploadErr})
	got := next.(model)
	if got.workers[1].uploads[0].status != statusFailed {
		t.Errorf("Should set status to failed")
	}
	if !errors.Is(got.workers[1].uploads[0].err, uploadErr) {
		t.Errorf("Should store error")
	}
}

func TestUpdate_SpeciesDone_Success(t *testing.T) {
	m := baseModel(t)
	m.workers[0].name = "American Robin"
	m.workers[0].uploads = []uploadItem{{key: "recordings/amro/XC1.mp3", status: statusDone}}
	m.done = 2

	next, _ := m.Update(speciesDoneMsg{
		workerID: 0, code: "amro", name: "American Robin",
		recordings: 4, images: 3, failures: nil,
	})
	got := next.(model)

	if got.done != 3 {
		t.Errorf("Should increment done counter: got %d", got.done)
	}
	if got.workers[0].name != "" {
		t.Errorf("Should clear worker name")
	}
	if len(got.workers[0].uploads) != 0 {
		t.Errorf("Should clear worker uploads")
	}
	if len(got.completed) != 1 {
		t.Fatalf("Should add to completed list")
	}
	c := got.completed[0]
	if c.name != "American Robin" || c.recordings != 4 || c.images != 3 || c.failed {
		t.Errorf("Should store completed item correctly: %+v", c)
	}
}

func TestUpdate_SpeciesDone_WithFailures(t *testing.T) {
	m := baseModel(t)
	next, _ := m.Update(speciesDoneMsg{
		workerID: 0, code: "busti", name: "Bushtit",
		recordings: 0, images: 3, failures: []string{"upload error"},
	})
	got := next.(model)
	if !got.completed[0].failed {
		t.Errorf("Should mark as failed when there are upload failures")
	}
}

func TestUpdate_SpeciesDone_MissingMedia(t *testing.T) {
	m := baseModel(t)
	next, _ := m.Update(speciesDoneMsg{
		workerID: 0, code: "busti", name: "Bushtit",
		recordings: 0, images: 3, failures: nil,
	})
	got := next.(model)
	if !got.completed[0].failed {
		t.Errorf("Should mark as failed when recordings == 0")
	}
}

func TestUpdate_AllDone_ReturnsQuit(t *testing.T) {
	m := baseModel(t)
	_, cmd := m.Update(allDoneMsg{})
	if cmd == nil {
		t.Fatal("Should return a command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Should return tea.Quit command, got %T", msg)
	}
}

func TestUpdate_Tick_ReturnsTick(t *testing.T) {
	m := baseModel(t)
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("Should return a tick command for elapsed timer")
	}
}

func TestUpdate_WindowSize_SetsWidthHeight(t *testing.T) {
	m := baseModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(model)
	if got.width != 120 || got.height != 40 {
		t.Errorf("Should store terminal dimensions: got %dx%d", got.width, got.height)
	}
}
