package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixture(t *testing.T) (string, string, string) {
	t.Helper()
	h := t.TempDir()
	data := filepath.Join(h, ".local", "share")
	app := "org.example.App"
	folder := filepath.Join(data, "flatpak/exports/share/applications")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(h, "Desktop"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, app+".desktop"), []byte("[Desktop Entry]\nType=Application\nName=Example\nExec=/usr/bin/flatpak run org.example.App\nIcon=org.example.App\nX-Flatpak=org.example.App\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return h, data, app
}
func TestSyncCreatesMenuAndDesktopIdempotently(t *testing.T) {
	h, data, id := fixture(t)
	count, err := Sync(h, data, id)
	if err != nil || count != 2 {
		t.Fatalf("created=%d error=%v", count, err)
	}
	for _, p := range []string{filepath.Join(data, "applications", id+".desktop"), filepath.Join(h, "Desktop", id+".desktop")} {
		b, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(b), marker) || !strings.Contains(string(b), "Icon=org.example.App") {
			t.Fatalf("invalid launcher %s %s %v", p, b, err)
		}
		st, _ := os.Stat(p)
		if runtime.GOOS != "windows" && st.Mode().Perm() != 0755 {
			t.Fatalf("not executable: %v", st.Mode())
		}
	}
	count, err = Sync(h, data, id)
	if err != nil || count != 0 {
		t.Fatalf("repeat created=%d error=%v", count, err)
	}
}
func TestNeverOverwriteUnmanagedShortcut(t *testing.T) {
	h, data, id := fixture(t)
	path := filepath.Join(h, "Desktop", id+".desktop")
	if err := os.WriteFile(path, []byte("personal"), 0600); err != nil {
		t.Fatal(err)
	}
	n, err := Sync(h, data, id)
	if err != nil || n != 1 {
		t.Fatalf("changed=%d err=%v", n, err)
	}
	content, _ := os.ReadFile(path)
	if string(content) != "personal" {
		t.Fatal("overwrote personal shortcut")
	}
}
func TestInvalidAndUnsafeExportRejected(t *testing.T) {
	h, data, id := fixture(t)
	if _, err := Sync(h, data, "../other"); err == nil {
		t.Fatal("path traversal allowed")
	}
	p := filepath.Join(data, "flatpak/exports/share/applications", id+".desktop")
	if err := os.WriteFile(p, []byte("[Desktop Entry]\nType=Application\nName=bad\nExec=sh -c something\nX-Flatpak=org.other.App\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(h, data, id); err == nil {
		t.Fatal("unverified desktop entry accepted")
	}
}
func TestDesktopNotPresentStillCreatesMenu(t *testing.T) {
	h, data, id := fixture(t)
	if err := os.Remove(filepath.Join(h, "Desktop")); err != nil {
		t.Fatal(err)
	}
	n, err := Sync(h, data, id)
	if err != nil || n != 1 {
		t.Fatalf("got %d err %v", n, err)
	}
}
func TestCustomDesktopDirectoryUnderHome(t *testing.T) {
	h, data, id := fixture(t)
	if err := os.MkdirAll(filepath.Join(h, "Escritorio"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(h, ".config"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h, ".config", "user-dirs.dirs"), []byte("XDG_DESKTOP_DIR=\"$HOME/Escritorio\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	n, err := Sync(h, data, id)
	if err != nil || n != 2 {
		t.Fatalf("got %d %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(h, "Escritorio", id+".desktop")); err != nil {
		t.Fatal(err)
	}
}

func TestExportedIconIsVisibleWithoutXDGExportPath(t *testing.T) {
	h, data, id := fixture(t)
	iconDir := filepath.Join(data, "flatpak", "exports", "share", "icons", "hicolor", "scalable", "apps")
	if err := os.MkdirAll(iconDir, 0o700); err != nil {
		t.Fatal(err)
	}
	icon := filepath.Join(iconDir, id+".svg")
	if err := os.WriteFile(icon, []byte("<svg/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n, err := Sync(h, data, id); err != nil || n != 2 {
		t.Fatalf("sync: n=%d err=%v", n, err)
	}
	launcher, err := os.ReadFile(filepath.Join(data, "applications", id+".desktop"))
	if err != nil || !strings.Contains(string(launcher), "Icon="+icon) {
		t.Fatalf("icon not resolved: %s %v", launcher, err)
	}
}

func TestFlatpakManagedExportSymlink(t *testing.T) {
	h, data, id := fixture(t)
	exported := filepath.Join(data, "flatpak", "exports", "share", "applications", id+".desktop")
	targetDir := filepath.Join(data, "flatpak", "app", id, "x86_64", "stable", "active", "export", "share", "applications")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, id+".desktop")
	b, err := os.ReadFile(exported)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(exported); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, exported); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("Windows symlink creation may require privileges")
		}
		t.Fatal(err)
	}
	if n, err := Sync(h, data, id); err != nil || n != 2 {
		t.Fatalf("symlink export n=%d err=%v", n, err)
	}
}

func TestRejectExportSymlinkOutsideFlatpakApp(t *testing.T) {
	h, data, id := fixture(t)
	exported := filepath.Join(data, "flatpak", "exports", "share", "applications", id+".desktop")
	target := filepath.Join(h, "untrusted.desktop")
	b, _ := os.ReadFile(exported)
	if err := os.WriteFile(target, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(exported); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, exported); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("Windows symlink creation may require privileges")
		}
		t.Fatal(err)
	}
	if _, err := Sync(h, data, id); err == nil {
		t.Fatal("untrusted export symlink accepted")
	}
}
