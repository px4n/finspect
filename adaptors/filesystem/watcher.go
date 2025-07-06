package filesystem

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/px4n/finspect/pkg/vfs"
)

// watcher wraps fsnotify.Watcher for the filesystem adaptor
type watcher struct {
	watcher   *fsnotify.Watcher
	events    chan<- vfs.Event
	stop      chan struct{}
	done      sync.WaitGroup
	path      string
	recursive bool
	adaptor   *Adaptor
}

// Watch watches for changes in the specified path.
func (a *Adaptor) Watch(path string, recursive bool, events chan<- vfs.Event) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	// Check if already watching
	if _, exists := a.watches[path]; exists {
		return nil // Already watching
	}

	realPath := a.resolvePath(path)

	// Create fsnotify watcher
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	w := &watcher{
		watcher:   fsWatcher,
		events:    events,
		stop:      make(chan struct{}),
		path:      path,
		recursive: recursive,
		adaptor:   a,
	}

	// Add the path to watch
	if err := w.addPath(realPath); err != nil {
		_ = fsWatcher.Close()
		return err
	}

	// If recursive, add all subdirectories
	if recursive {
		if err := w.addRecursive(realPath); err != nil {
			_ = fsWatcher.Close()
			return err
		}
	}

	// Start the watcher goroutine
	w.done.Add(1)
	go w.run()

	a.watches[path] = w
	return nil
}

// Unwatch stops watching the specified path.
func (a *Adaptor) Unwatch(path string) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	w, exists := a.watches[path]
	if !exists {
		return nil // Not watching
	}

	w.Stop()
	delete(a.watches, path)
	return nil
}

// Stop stops the watcher.
func (w *watcher) Stop() {
	close(w.stop)
	w.done.Wait()
	_ = w.watcher.Close()
}

// run is the main watcher loop.
func (w *watcher) run() {
	defer w.done.Done()

	for {
		select {
		case <-w.stop:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Convert fsnotify event to VFS event
			vfsEvent := w.convertEvent(event)

			// Send the event
			select {
			case w.events <- vfsEvent:
			case <-w.stop:
				return
			}

			// Handle recursive watching for new directories
			if w.recursive && event.Op&fsnotify.Create != 0 {
				if info, err := w.adaptor.Stat(vfsEvent.Path); err == nil && info.IsDir() {
					realPath := w.adaptor.resolvePath(vfsEvent.Path)
					_ = w.addRecursive(realPath) // Best effort recursive watch
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}

			// Send error event
			select {
			case w.events <- vfs.Event{
				Type:   vfs.EventError,
				Path:   w.path,
				Time:   time.Now(),
				Source: w.adaptor.Name(),
				Error:  err,
			}:
			case <-w.stop:
				return
			}
		}
	}
}

// convertEvent converts an fsnotify event to a VFS event.
func (w *watcher) convertEvent(event fsnotify.Event) vfs.Event {
	// Convert real path back to VFS path
	relPath, _ := filepath.Rel(w.adaptor.root, event.Name)
	vfsPath := "/" + filepath.ToSlash(relPath)

	vfsEvent := vfs.Event{
		Path:   vfsPath,
		Time:   time.Now(),
		Source: w.adaptor.Name(),
	}

	// Map fsnotify operations to VFS event types
	switch {
	case event.Op&fsnotify.Create != 0:
		vfsEvent.Type = vfs.EventCreate
	case event.Op&fsnotify.Write != 0:
		vfsEvent.Type = vfs.EventWrite
	case event.Op&fsnotify.Remove != 0:
		vfsEvent.Type = vfs.EventRemove
	case event.Op&fsnotify.Rename != 0:
		vfsEvent.Type = vfs.EventRename
	case event.Op&fsnotify.Chmod != 0:
		vfsEvent.Type = vfs.EventChmod
	default:
		vfsEvent.Type = vfs.EventUnknown
	}

	return vfsEvent
}

// addPath adds a path to the watcher.
func (w *watcher) addPath(path string) error {
	return w.watcher.Add(path)
}

// addRecursive recursively adds all subdirectories.
func (w *watcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			if err := w.addPath(path); err != nil {
				return nil // Skip errors for individual paths
			}
		}

		return nil
	})
}
