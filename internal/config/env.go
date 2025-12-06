package config

type watcherType string

const (
	WatcherFSNotify watcherType = "fsnotify"
	WatcherTimer    watcherType = "timer"
)

func New() (Config, error) {
	cfg := Load()

	return cfg, nil
}
