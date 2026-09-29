package service

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// FileChangePayload is the event data pushed to the frontend when a watched
// file or directory changes.
type FileChangePayload struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
	Op    string `json:"op"` // "created" | "changed" | "deleted" | "renamed"
}

// WatchService uses fsnotify to watch one or more directories for filesystem
// changes and broadcasts them to the frontend via Wails events.
type WatchService struct {
	watcher   *fsnotify.Watcher
	mu        sync.Mutex
	watched   map[string]bool
	closeOnce sync.Once
	closed    chan struct{}
}

// NewWatchService creates a new WatchService. The underlying fsnotify watcher
// is created lazily on the first Watch call.
func NewWatchService() *WatchService {
	return &WatchService{
		watched: make(map[string]bool),
		closed:  make(chan struct{}),
	}
}

// Watch starts watching dir and all of its subdirectories recursively. If the
// same directory is already watched this is a no-op.
func (s *WatchService) Watch(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.watcher == nil {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		s.watcher = w
		go s.eventLoop()
	}

	if s.watched[abs] {
		return nil
	}

	// Watch the root.
	if err := s.watcher.Add(abs); err != nil {
		return err
	}
	s.watched[abs] = true

	// Walk and watch every subdirectory (fsnotify v1 is non-recursive).
	filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || !info.IsDir() {
			return nil
		}
		if s.watched[path] {
			return nil
		}
		if err := s.watcher.Add(path); err != nil {
			return nil // best-effort; skip unreadable dirs
		}
		s.watched[path] = true
		return nil
	})

	return nil
}

// Unwatch removes a directory and all its watched subdirectories from the
// watcher.
func (s *WatchService) Unwatch(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.watcher == nil {
		return nil
	}

	for p := range s.watched {
		if p == abs || hasPathPrefix(p, abs) {
			_ = s.watcher.Remove(p)
			delete(s.watched, p)
		}
	}
	return nil
}

// Close shuts down the watcher.
func (s *WatchService) Close() {
	s.closeOnce.Do(func() {
		close(s.closed)
		s.mu.Lock()
		if s.watcher != nil {
			_ = s.watcher.Close()
			s.watcher = nil
		}
		s.mu.Unlock()
	})
}

func (s *WatchService) eventLoop() {
	app := application.Get()
	if app == nil {
		return
	}
	for {
		select {
		case <-s.closed:
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			s.handleEvent(app, event)
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watch error: %v", err)
		}
	}
}

func (s *WatchService) handleEvent(app *application.App, event fsnotify.Event) {
	payload := FileChangePayload{
		Path: event.Name,
		Name: filepath.Base(event.Name),
		Op:   opName(event.Op),
	}

	// Try to determine if it's a directory. For removed/renamed entries the
	// stat may fail, so fall back to the basename heuristic.
	if info, err := os.Stat(event.Name); err == nil {
		payload.IsDir = info.IsDir()
	}

	// Newly created directories must be watched so their children are tracked.
	if event.Op&fsnotify.Create != 0 && payload.IsDir {
		s.mu.Lock()
		if !s.watched[event.Name] {
			if err := s.watcher.Add(event.Name); err == nil {
				s.watched[event.Name] = true
			}
		}
		s.mu.Unlock()
	}

	switch {
	case event.Op&fsnotify.Create != 0:
		app.Event.Emit("file:created", payload)
	case event.Op&fsnotify.Write != 0:
		app.Event.Emit("file:changed", payload)
	case event.Op&fsnotify.Remove != 0:
		s.mu.Lock()
		delete(s.watched, event.Name)
		s.mu.Unlock()
		app.Event.Emit("file:deleted", payload)
	case event.Op&fsnotify.Rename != 0:
		app.Event.Emit("file:renamed", payload)
	}
}

func opName(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create != 0:
		return "created"
	case op&fsnotify.Write != 0:
		return "changed"
	case op&fsnotify.Remove != 0:
		return "deleted"
	case op&fsnotify.Rename != 0:
		return "renamed"
	default:
		return "changed"
	}
}

// hasPathPrefix reports whether child is under parent.
func hasPathPrefix(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel)
}
