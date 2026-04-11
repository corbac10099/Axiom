package jsruntime

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher surveille les fichiers .js des modules et déclenche le rechargement.
type Watcher struct {
	mu       sync.RWMutex
	watcher  *fsnotify.Watcher
	handlers map[string]func(string) // path → callback(newCode)
	logger   *slog.Logger
}

// NewWatcher crée un Watcher prêt à surveiller des fichiers JS.
func NewWatcher(logger *slog.Logger) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher:  fw,
		handlers: make(map[string]func(string)),
		logger:   logger,
	}, nil
}

// Watch ajoute un fichier à surveiller.
// Quand le fichier change, callback(newCode) est appelé avec le nouveau contenu.
func (w *Watcher) Watch(jsPath string, callback func(newCode string)) error {
	abs, err := filepath.Abs(jsPath)
	if err != nil {
		return err
	}
	if err := w.watcher.Add(abs); err != nil {
		return err
	}
	w.mu.Lock()
	w.handlers[abs] = callback
	w.mu.Unlock()
	w.logger.Debug("watcher: watching", slog.String("path", abs))
	return nil
}

// Start lance la boucle de surveillance en arrière-plan.
func (w *Watcher) Start() {
	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				// On ne réagit qu'aux écritures et renames (save dans les éditeurs)
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
					abs := event.Name
					w.mu.RLock()
					cb, exists := w.handlers[abs]
					w.mu.RUnlock()
					if !exists {
						continue
					}
					code, err := os.ReadFile(abs)
					if err != nil {
						w.logger.Warn("watcher: cannot read file",
							slog.String("path", abs),
							slog.String("error", err.Error()),
						)
						continue
					}
					w.logger.Info("watcher: hot-reloading module",
						slog.String("path", abs),
					)
					cb(string(code))
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				w.logger.Warn("watcher: error", slog.String("error", err.Error()))
			}
		}
	}()
}

// Stop arrête le watcher.
func (w *Watcher) Stop() error {
	return w.watcher.Close()
}