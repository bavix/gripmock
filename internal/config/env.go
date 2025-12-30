package config

type watcherType string

const (
	WatcherFSNotify watcherType = "fsnotify"
	WatcherTimer    watcherType = "timer"
)
