// Package storage provides small, dependency-free file-persistence helpers
// used by Snapshot/Replay, project Auto-Save and Config loading. It is
// intentionally simple (JSON-on-disk) so the whole platform runs with zero
// external services; swap in a real object store by implementing the same
// two functions in a build tag if needed later.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SaveJSON serializes v to path (creating parent directories as needed) with
// pretty-printed, human-diffable JSON — useful for Snapshot/Replay files
// that users may want to inspect or check into version control.
func SaveJSON(path string, v interface{}) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadJSON reads and decodes a JSON file written by SaveJSON.
func LoadJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// SnapshotMeta describes one saved Universe snapshot on disk.
type SnapshotMeta struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
	SizeBytes int64     `json:"sizeBytes"`
}

// ListSnapshots scans dir for *.json snapshot files and returns their meta.
func ListSnapshots(dir string) ([]SnapshotMeta, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var metas []SnapshotMeta
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		id := e.Name()[:len(e.Name())-len(".json")]
		metas = append(metas, SnapshotMeta{
			ID:        id,
			Label:     id,
			Path:      filepath.Join(dir, e.Name()),
			CreatedAt: info.ModTime(),
			SizeBytes: info.Size(),
		})
	}
	return metas, nil
}

// AutoSaver periodically calls save() to disk in the background, implementing
// the platform's Auto Save / Crash Recovery requirement.
type AutoSaver struct {
	stop chan struct{}
}

func StartAutoSaver(interval time.Duration, save func() error) *AutoSaver {
	as := &AutoSaver{stop: make(chan struct{})}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = save()
			case <-as.stop:
				return
			}
		}
	}()
	return as
}

func (as *AutoSaver) Stop() { close(as.stop) }

// EnsureDir is a tiny convenience wrapper used across cmd/ entry points.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("storage: creating %s: %w", path, err)
	}
	return nil
}
