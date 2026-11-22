// Package desktop makes verified Flatpak launchers visible in desktop menus and
// optionally places shortcuts on the current user's desktop. It never modifies
// system directories or invents an Exec command for an unverified application.
package desktop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const marker = "X-Sovereign-Managed=true"

// Sync copies an existing Flatpak-exported .desktop file (including its icon
// and Exec metadata). Copies are user-local. Existing user entries are never
// replaced unless they were created by Sovereign.
// Desktop shortcuts are created only if the configured desktop dir exists.
func Sync(home, dataHome, appID string) (int, error) {
	if home == "" || dataHome == "" {
		return 0, errors.New("home or XDG data directory unavailable")
	}
	if !safeID(appID) {
		return 0, fmt.Errorf("invalid Flatpak application ID %q", appID)
	}
	exported, err := findExport(home, appID)
	if err != nil {
		return 0, err
	}
	content, err := os.ReadFile(exported)
	if err != nil {
		return 0, fmt.Errorf("read exported launcher: %w", err)
	}
	if !validLauncher(content, appID) {
		return 0, fmt.Errorf("Flatpak launcher for %s has unexpected metadata; refusing to copy", appID)
	}
	// If the session does not include Flatpak exports in XDG_DATA_DIRS,
	// its icon theme may also be invisible. Prefer a real exported icon path.
	content = visibleIcon(content, exported)
	name := appID + ".desktop"
	changed := 0
	menu := filepath.Join(dataHome, "applications", name)
	did, err := writeManaged(menu, content, true)
	if err != nil {
		return changed, fmt.Errorf("menu launcher: %w", err)
	}
	if did {
		changed++
	}
	desk, ok := desktopDir(home)
	if ok {
		did, err = writeManaged(filepath.Join(desk, name), content, false)
		if err != nil {
			return changed, fmt.Errorf("desktop shortcut: %w", err)
		}
		if did {
			changed++
		}
	}
	return changed, nil
}

func safeID(id string) bool {
	if len(id) < 3 || len(id) > 180 || strings.HasPrefix(id, ".") || strings.HasSuffix(id, ".") {
		return false
	}
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return strings.Contains(id, ".")
}

func findExport(home, id string) (string, error) {
	locations := []string{
		filepath.Join(home, ".local/share/flatpak/exports/share/applications", id+".desktop"),
		filepath.Join("/var/lib/flatpak/exports/share/applications", id+".desktop"),
	}
	for _, name := range locations {
		info, err := os.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("exported desktop entry is not a regular file: %s", name)
		}
		return name, nil
	}
	return "", fmt.Errorf("Flatpak has not exported a launcher for %s; try logging out and back in", id)
}

// Validate the provenance marker supplied by Flatpak itself. Copies of
// untrusted arbitrary desktop files are not permitted.
func validLauncher(data []byte, id string) bool {
	s := string(data)
	if !strings.Contains(s, "[Desktop Entry]") || !strings.Contains(s, "\nType=Application") {
		return false
	}
	if !(strings.Contains(s, "\nX-Flatpak="+id+"\n") || strings.HasSuffix(s, "\nX-Flatpak="+id)) {
		return false
	}
	return strings.Contains(s, "\nExec=") && strings.Contains(s, "\nName=")
}

// visibleIcon preserves the exported icon, but uses an absolute reference when
// its theme directory is not available through XDG_DATA_DIRS. Never execute
// metadata or search outside the Flatpak export tree.
func visibleIcon(entry []byte, exported string) []byte {
	lines := strings.Split(string(entry), "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "Icon=") {
			continue
		}
		icon := strings.TrimPrefix(line, "Icon=")
		if icon == "" || strings.ContainsAny(icon, "/\\") {
			return entry
		}
		for _, r := range icon {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-') {
				return entry
			}
		}
		share := filepath.Dir(filepath.Dir(exported))
		for _, size := range []string{"scalable", "256x256", "128x128", "64x64", "48x48"} {
			for _, ext := range []string{".svg", ".png"} {
				path := filepath.Join(share, "icons", "hicolor", size, "apps", icon+ext)
				if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
					lines[i] = "Icon=" + path
					return []byte(strings.Join(lines, "\n"))
				}
			}
		}
		return entry
	}
	return entry
}

func writeManaged(path string, original []byte, menu bool) (bool, error) {
	dir := filepath.Dir(path)
	if menu {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return false, err
		}
	} else {
		stat, err := os.Lstat(dir)
		if err != nil {
			return false, err
		}
		if !stat.IsDir() {
			return false, errors.New("desktop directory is not a directory")
		}
	}
	if st, err := os.Lstat(path); err == nil {
		if !st.Mode().IsRegular() {
			return false, fmt.Errorf("refusing to replace non-regular launcher %s", path)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return false, err
		}
		if !strings.Contains(string(b), "\n"+marker+"\n") && !strings.HasSuffix(string(b), "\n"+marker) {
			return false, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	// Write atomically so interrupted setup cannot leave a truncated launcher.
	data := append([]byte(nil), original...)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, []byte(marker+"\n")...)
	if old, err := os.ReadFile(path); err == nil && string(old) == string(data) {
		return false, nil
	}
	tmp, err := os.CreateTemp(dir, ".sovereign-launcher-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return false, err
	}
	if err = tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return false, err
	}
	if err = tmp.Close(); err != nil {
		return false, err
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}

// XDG user-dirs.desktop uses a shell-like quoted HOME-prefixed path.
// Parse only absolute paths under HOME to avoid writing arbitrary locations.
func desktopDir(home string) (string, bool) {
	if home == "" {
		return "", false
	}
	candidate := filepath.Join(home, "Desktop")
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	data, err := os.ReadFile(filepath.Join(configHome, "user-dirs.dirs"))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "XDG_DESKTOP_DIR=") {
				continue
			}
			value := strings.Trim(strings.TrimPrefix(line, "XDG_DESKTOP_DIR="), "\"")
			value = strings.ReplaceAll(value, "$HOME", home)
			if filepath.IsAbs(value) && withinHome(home, value) {
				candidate = value
			}
		}
	}
	info, err := os.Lstat(candidate)
	return candidate, err == nil && info.IsDir() && info.Mode()&fs.ModeSymlink == 0
}

func withinHome(home, path string) bool {
	h, err := filepath.Abs(home)
	if err != nil {
		return false
	}
	p, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(h, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// EnvironmentSync resolves XDG directories without requiring xdg-user-dir,
// desktop-file-install or another external helper.
func EnvironmentSync(appID string) (int, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	return Sync(home, dataHome, appID)
}
