module github.com/eXotech-code/fsnotify

go 1.21.1

require (
	github.com/eXotech-code/fsevents v0.0.0-20231004111012-8c853885a279
	github.com/fsnotify/fsnotify v1.6.0
	golang.org/x/sys v0.12.0
)

retract (
	v1.5.3 // Published an incorrect branch accidentally https://github.com/fsnotify/fsnotify/issues/445
	v1.5.0 // Contains symlink regression https://github.com/fsnotify/fsnotify/pull/394
)
