// Package storage provides a tiny localStorage-style key/value store backed
// by a single JSON file in the platform user-config directory
// (os.UserConfigDir()/goed/storage.json). Values are process-lifetime cached;
// every mutation rewrites the file atomically (tmp file + rename) so a crash
// mid-write cannot corrupt the store.
package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store is a persistent key/value map. It is safe for concurrent use.
type Store struct {
	mu    sync.Mutex
	path  string
	data  map[string]any
	loaded bool
}

var (
	defaultStore *Store
	defaultOnce  sync.Once
	defaultErr   error
)

// Default returns the process-wide store rooted at the platform user-config
// directory. The file is created lazily on first write.
func Default() (*Store, error) {
	defaultOnce.Do(func() {
		dir, err := os.UserConfigDir()
		if err != nil {
			defaultErr = err
			return
		}
		defaultStore, defaultErr = Open(filepath.Join(dir, "goed", "storage.json"))
	})
	return defaultStore, defaultErr
}

// Open loads (or initializes) a store backed by the given JSON file.
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loaded = true
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = map[string]any{}
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		s.data = map[string]any{}
		return nil
	}
	return json.Unmarshal(raw, &s.data)
}

// Get returns the value stored under key and whether it exists.
func (s *Store) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loaded {
		_ = s.load()
	}
	v, ok := s.data[key]
	return v, ok
}

// GetString returns a string value (JSON default when absent or mistyped).
func (s *Store) GetString(key, def string) string {
	if v, ok := s.Get(key); ok {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return def
}

// GetInt returns an int value. JSON numbers decode as float64, so both int
// and float64 representations are accepted.
func (s *Store) GetInt(key string, def int) int {
	if v, ok := s.Get(key); ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

// GetBool returns a bool value (def when absent or mistyped).
func (s *Store) GetBool(key string, def bool) bool {
	if v, ok := s.Get(key); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

// Set stores a value and persists the file immediately. The error from the
// underlying write is returned so callers (e.g. the storage hook) can decide
// whether to surface it.
func (s *Store) Set(key string, value any) error {
	s.mu.Lock()
	s.data[key] = value
	err := s.writeLocked()
	s.mu.Unlock()
	return err
}

// writeLocked persists the map atomically; caller must hold s.mu.
func (s *Store) writeLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Path reports the backing file location (mainly for diagnostics and tests).
func (s *Store) Path() string {
	return s.path
}
