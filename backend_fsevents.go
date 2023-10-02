//go:build darwin
// +build darwin

package fsnotify

import (
	"github.com/eXotech-code/fsevents"
	"sync"
	"syscall"
	"time"
)

type Watcher struct {
	// Events sends the filesystem change events.
	//
	// fsnotify can send the following events; a "path" here can refer to a
	// file, directory, symbolic link, or special file like a FIFO.
	//
	//   fsnotify.Create    A new path was created; this may be followed by one
	//                      or more Write events if data also gets written to a
	//                      file.
	//
	//   fsnotify.Remove    A path was removed.
	//
	//   fsnotify.Rename    A path was renamed. A rename is always sent with the
	//                      old path as Event.Name, and a Create event will be
	//                      sent with the new name. Renames are only sent for
	//                      paths that are currently watched; e.g. moving an
	//                      unmonitored file into a monitored directory will
	//                      show up as just a Create. Similarly, renaming a file
	//                      to outside a monitored directory will show up as
	//                      only a Rename.
	//
	//   fsnotify.Write     A file or named pipe was written to. A Truncate will
	//                      also trigger a Write. A single "write action"
	//                      initiated by the user may show up as one or multiple
	//                      writes, depending on when the system syncs things to
	//                      disk. For example when compiling a large Go program
	//                      you may get hundreds of Write events, so you
	//                      probably want to wait until you've stopped receiving
	//                      them (see the dedup example in cmd/fsnotify).
	//                      Some systems may send Write event for directories
	//                      when the directory content changes.
	//
	//   fsnotify.Chmod     Attributes were changed. On Linux this is also sent
	//                      when a file is removed (or more accurately, when a
	//                      link to an inode is removed). On kqueue it's sent
	//                      and on kqueue when a file is truncated. On Windows
	//                      it's never sent.
	Events chan Event

	// Errors sends any errors.
	//
	// [ErrEventOverflow] is used to indicate there are too many events:
	//
	//  - inotify: there are too many queued events (fs.inotify.max_queued_events sysctl)
	//  - windows: The buffer size is too small; [WithBufferSize] can be used to increase it.
	//  - kqueue, fen: not used.
	Errors chan error

	done               chan struct{}
	watches            map[string]int // Watched file descriptors (key: path).
	eventStream        *fsevents.EventStream
	eventStreamStarted bool
	isClosed           bool
	mu                 sync.Mutex
}

// Returns true if the event was sent, or false if watcher is closed.
func (w *Watcher) sendEvent(event Event) bool {
	select {
	case w.Events <- event:
		return true
	case <-w.done:
		return false
	}
}

// Converts an fsevents.Event value to a fsnotify.Event value
// in order to get a portable event value that has the same
// meaing accross platforms.
func getPortableEvent(e fsevents.Event) (converted Event) {
	converted.Name = e.Path
	f := e.Flags

	if f&fsevents.ItemCreated == fsevents.ItemCreated {
		converted.Op |= Create
	}
	if f&fsevents.ItemRemoved == fsevents.ItemRemoved {
		converted.Op |= Remove
	}
	if f&fsevents.ItemModified == fsevents.ItemModified {
		converted.Op |= Write
	}
	if f&fsevents.ItemRenamed == fsevents.ItemRenamed {
		converted.Op |= Rename
	}
	if f&fsevents.ItemInodeMetaMod == fsevents.ItemInodeMetaMod || f&fsevents.ItemXattrMod == fsevents.ItemXattrMod {
		converted.Op |= Chmod
	}

	return
}

func (w *Watcher) readEvents() {
	defer func() {
		close(w.Events)
		close(w.Errors)
	}()

	ec := w.eventStream.Events
	for eventArr := range ec {
		for _, e := range eventArr {
			w.sendEvent(getPortableEvent(e))
		}
	}
}

func getDeviceIdForPath(path string) (int32, error) {
	stat := syscall.Stat_t{}
	if err := syscall.Lstat(path, &stat); err != nil {
		return -1, err
	}
	return stat.Dev, nil
}

func (w *Watcher) Add(name string) (err error) {
	dev, err := getDeviceIdForPath(name)
	if err != nil {
		return err
	}

	w.eventStream.Paths = append(w.eventStream.Paths, name)
	if !w.eventStreamStarted {
		w.eventStream.Device = dev
		w.eventStream.Start()
	} else {
		w.eventStream.Restart()
	}

	return
}

func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isClosed {
		return nil
	}
	w.isClosed = true

	w.eventStream.Stop()
	close(w.done)

	return nil
}

func NewWatcher() (*Watcher, error) {
	es := &fsevents.EventStream{
		Paths:   make([]string, 0),
		Latency: 500 * time.Millisecond,
		// "Device" will get populated later with the real device ID
		// for the first watched path in "Watcher.Add()".
		Device: -1,
		Flags:  fsevents.FileEvents | fsevents.WatchRoot,
	}

	w := &Watcher{
		Events:      make(chan Event),
		Errors:      make(chan error),
		done:        make(chan struct{}),
		watches:     make(map[string]int),
		eventStream: es,
	}

	go w.readEvents()
	return w, nil
}
