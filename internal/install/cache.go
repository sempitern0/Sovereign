package install

import (
	"context"
	"sync"
)

// Cache memoizes read-only provider output for the duration of one command, so
// a profile with several Flatpak applications lists installed applications and
// remotes once instead of once per application. It is safe for concurrent use;
// concurrent callers of the same query wait for a single provider call.
//
// Only successful results are cached: a failed inspection is retried by the next
// caller and never turns into "missing". Any install invalidates the installed
// application list, so the verification after installing always re-inspects.
type Cache struct {
	mu          sync.Mutex
	flatpakApps *string
	flathubOK   bool
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

// invalidate forgets installed-package state after a host change.
func (c *Cache) invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.flatpakApps = nil
	c.mu.Unlock()
}
