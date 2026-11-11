// Package state keeps Sovereign's small local journal of installation results
// and the lock that prevents two `apply` runs from overlapping.
//
// The journal is informational: package managers remain the source of truth and
// every run re-inspects them. It never stores secrets, raw package-manager output,
// host names or personal paths; failures are recorded as short error classes.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	SchemaVersion = 1
	journalFile   = "journal.json"
	lockFile      = "apply.lock"
	maxJournal    = 1 << 20
)

// Result values recorded per application.
const (
	ResultInstalled        = "installed"
	ResultAlreadyInstalled = "already_installed"
	ResultFailed           = "failed"
)

// Entry is the latest known outcome for one application on this device.
type Entry struct {
	Application string `json:"application"`
	Platform    string `json:"platform"`
	Provider    string `json:"provider"`
	PackageID   string `json:"package_id"`
	Scope       string `json:"scope,omitempty"`
	Result      string `json:"result"`
	Changed     bool   `json:"changed"`
	ErrorClass  string `json:"error_class,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

type Journal struct {
	SchemaVersion int     `json:"schema_version"`
	Entries       []Entry `json:"entries"`
}

// ErrLocked means another apply run holds the lock.
var ErrLocked = errors.New("another suitectl apply is running")

// Store reads and writes the journal under Dir.
type Store struct {
	Dir string
	Now func() time.Time
}

// DefaultDir returns the per-user state directory:
// %LOCALAPPDATA%\Sovereign\state on Windows, $XDG_STATE_HOME/sovereign or
// ~/.local/state/sovereign elsewhere.
func DefaultDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" || !filepath.IsAbs(base) {
			return "", errors.New("LOCALAPPDATA is not set to an absolute path")
		}
		return filepath.Join(base, "Sovereign", "state"), nil
	}
	if base := os.Getenv("XDG_STATE_HOME"); base != "" {
		if !filepath.IsAbs(base) {
			return "", errors.New("XDG_STATE_HOME must be an absolute path")
		}
		return filepath.Join(base, "sovereign"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "sovereign"), nil
}

func (s Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Load returns the journal. A missing journal is empty. A corrupt one is moved
// aside (never deleted) and an empty journal is returned with a non-nil warning.
func (s Store) Load() (Journal, error) {
	empty := Journal{SchemaVersion: SchemaVersion, Entries: []Entry{}}
	path := filepath.Join(s.Dir, journalFile)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return empty, err
	}
	var j Journal
	if info.Mode().IsRegular() && info.Size() <= maxJournal {
		if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &j) == nil && j.SchemaVersion == SchemaVersion {
			if j.Entries == nil {
				j.Entries = []Entry{}
			}
			return j, nil
		}
	}
	aside := fmt.Sprintf("%s.corrupt-%d", path, s.now().UTC().Unix())
	if err := os.Rename(path, aside); err != nil {
		return empty, fmt.Errorf("unreadable journal and cannot move it aside: %w", err)
	}
	return empty, fmt.Errorf("unreadable journal moved to %s; starting a new one", filepath.Base(aside))
}

// Record upserts entries by application and platform and writes atomically.
func (s Store) Record(entries ...Entry) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	journal, loadErr := s.Load()
	stamp := s.now().UTC().Format(time.RFC3339)
	index := map[string]int{}
	for i, e := range journal.Entries {
		index[e.Application+"/"+e.Platform] = i
	}
	for _, e := range entries {
		e.UpdatedAt = stamp
		key := e.Application + "/" + e.Platform
		if i, ok := index[key]; ok {
			journal.Entries[i] = e
		} else {
			index[key] = len(journal.Entries)
			journal.Entries = append(journal.Entries, e)
		}
	}
	sort.Slice(journal.Entries, func(i, j int) bool {
		a, b := journal.Entries[i], journal.Entries[j]
		if a.Application != b.Application {
			return a.Application < b.Application
		}
		return a.Platform < b.Platform
	})
	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(s.Dir, journalFile), append(data, '\n')); err != nil {
		return err
	}
	return loadErr
}

// writeAtomic writes to a temporary file in the same directory, syncs it and
// renames it over the target, so readers never see a partial journal.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// Lock takes the apply lock. A lock left by a process that no longer exists is
// replaced; a live holder yields ErrLocked with the lock path in the message.
func (s Store) Lock() (func(), error) {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(s.Dir, lockFile)
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, werr := fmt.Fprintf(f, "%d\n", os.Getpid())
			cerr := f.Close()
			if werr != nil || cerr != nil {
				os.Remove(path)
				return nil, errors.Join(werr, cerr)
			}
			return func() { os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if pid, ok := lockHolder(path); ok && processAlive(pid) {
			return nil, fmt.Errorf("%w (pid %d); if it is not, remove %s", ErrLocked, pid, path)
		}
		// Stale or unreadable lock from a run that ended abnormally.
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%w; remove %s if no other run is active", ErrLocked, path)
}

func lockHolder(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	return pid, err == nil && pid > 0
}
