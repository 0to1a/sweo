package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateSessionID checks that a session ID is safe for filesystem use.
func ValidateSessionID(id string) error {
	if !sessionIDPattern.MatchString(id) {
		return fmt.Errorf("invalid session ID %q: must match [a-zA-Z0-9_-]+", id)
	}
	return nil
}

// ReadMetadata reads the key=value metadata file for a session.
// Returns nil, nil if the file does not exist.
func ReadMetadata(sessionsDir, sessionID string) (map[string]string, error) {
	if err := ValidateSessionID(sessionID); err != nil {
		return nil, err
	}

	path := filepath.Join(sessionsDir, sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read metadata %s: %w", path, err)
	}

	return ParseKeyValue(string(data)), nil
}

// WriteMetadata atomically writes key=value metadata for a session.
func WriteMetadata(sessionsDir, sessionID string, data map[string]string) error {
	if err := ValidateSessionID(sessionID); err != nil {
		return err
	}

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	target := filepath.Join(sessionsDir, sessionID)
	content := SerializeKeyValue(data)

	// Atomic write: temp file + rename
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return fmt.Errorf("write temp metadata: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename metadata: %w", err)
	}

	return nil
}

// UpdateMetadata merges updates into existing metadata. Empty string values delete keys.
func UpdateMetadata(sessionsDir, sessionID string, updates map[string]string) error {
	existing, err := ReadMetadata(sessionsDir, sessionID)
	if err != nil {
		return err
	}
	if existing == nil {
		existing = make(map[string]string)
	}

	for k, v := range updates {
		if v == "" {
			delete(existing, k)
		} else {
			existing[k] = v
		}
	}

	return WriteMetadata(sessionsDir, sessionID, existing)
}

// ListSessions returns all session IDs in the sessions directory.
func ListSessions(sessionsDir string) ([]string, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		if sessionIDPattern.MatchString(name) {
			ids = append(ids, name)
		}
	}

	sort.Strings(ids)
	return ids, nil
}

// ReserveSessionID atomically creates an empty metadata file, failing if it already exists.
func ReserveSessionID(sessionsDir, sessionID string) error {
	if err := ValidateSessionID(sessionID); err != nil {
		return err
	}

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	path := filepath.Join(sessionsDir, sessionID)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("session ID %q already exists", sessionID)
		}
		return fmt.Errorf("reserve session ID: %w", err)
	}
	f.Close()
	return nil
}

// DeleteMetadata removes the metadata file for a session.
func DeleteMetadata(sessionsDir, sessionID string) error {
	if err := ValidateSessionID(sessionID); err != nil {
		return err
	}
	path := filepath.Join(sessionsDir, sessionID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete metadata: %w", err)
	}
	return nil
}

// ParseKeyValue parses key=value lines into a map.
func ParseKeyValue(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := line[:idx]
		value := line[idx+1:]
		result[key] = value
	}
	return result
}

// SerializeKeyValue serializes a map to key=value lines, sorted by key.
func SerializeKeyValue(data map[string]string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, data[k])
	}
	return b.String()
}
