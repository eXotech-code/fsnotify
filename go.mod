module github.com/eXotech-code/fsnotify

go 1.17

require (
	github.com/eXotech-code/fsevents v0.0.0-20231002091326-c324a3face61
	github.com/fsnotify/fsnotify v1.6.0
	golang.org/x/sys v0.4.0
)

retract (
	v1.5.3 // Published an incorrect branch accidentally https://github.com/fsnotify/fsnotify/issues/445
	v1.5.0 // Contains symlink regression https://github.com/fsnotify/fsnotify/pull/394
)
