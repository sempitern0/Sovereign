package install

import (
	"context"
	"sync"
)

// Cache memoizes read-only provider output for the duration of one command, so
// a profile with several Flatpak applications lists installed applications and
// remotes once instead of once per application, and a WinGet package inspected
// by the plan is not queried again before installing. It is safe for concurrent
// use: Flatpak callers wait for a single provider call, while WinGet queries for
// different packages run without holding the lock.
//
// Only successful results are cached: a failed inspection is retried by the next
// caller and never turns into "missing". Any install invalidates the installed
// application list, so the verification after installing always re-inspects.
type Cache struct {
	mu          sync.Mutex
	flatpakApps *string
	flathubOK   bool
	winget      map[string]State // by package ID
}

// NewCache returns an empty cache for one command run.
func NewCache() *Cache { return &Cache{} }

// flatpakList returns the cached `flatpak list` output or runs it once.
func (c *Cache) flatpakList(ctx context.Context, run func(context.Context) (string, error)) (string, error) {
	if c == nil {
		return run(ctx)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.flatpakApps != nil {
		return *c.flatpakApps, nil
	}
	out, err := run(ctx)
	if err != nil {
		return "", err
	}
	c.flatpakApps = &out
	return out, nil
}

// flathub runs the remote check until it succeeds once.
func (c *Cache) flathub(ctx context.Context, check func(context.Context) error) error {
	if c == nil {
		return check(ctx)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.flathubOK {
		return nil
	}
	if err := check(ctx); err != nil {
		return err
	}
	c.flathubOK = true
	return nil
}

// wingetStatus returns the cached state of one WinGet package or inspects it.
// The lock is not held while WinGet runs, so different packages are inspected
// concurrently.
func (c *Cache) wingetStatus(ctx context.Context, id string, inspect func(context.Context) (State, error)) (State, error) {
	if c == nil {
		return inspect(ctx)
	}
	c.mu.Lock()
	state, ok := c.winget[id]
	c.mu.Unlock()
	if ok {
		return state, nil
	}
	state, err := inspect(ctx)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	if c.winget == nil {
		c.winget = map[string]State{}
	}
	c.winget[id] = state
	c.mu.Unlock()
	return state, nil
}

// invalidate forgets all installed-package state after a host change (an
// install may also add declared dependencies).
func (c *Cache) invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.flatpakApps = nil
	c.winget = nil
	c.mu.Unlock()
}
