package migrations

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"testing"
)

var activeMigrationFilename = regexp.MustCompile(`^(\d{6})_([a-z0-9][a-z0-9_-]*)\.(up|down)\.sql$`)

type migrationFile struct {
	version   int64
	name      string
	direction string
	filename  string
}

// TestActiveMigrationRegistry is intentionally directory-wide. It must inspect
// the same directory passed to golang-migrate, rather than a hand-maintained
// list that could omit a conflicting migration.
func TestActiveMigrationRegistry(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	files := make([]migrationFile, 0)
	byVersion := make(map[int64]string)
	pairs := make(map[string]map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if len(filename) < len(".sql") || filename[len(filename)-len(".sql"):] != ".sql" {
			continue
		}
		matches := activeMigrationFilename.FindStringSubmatch(filename)
		if matches == nil {
			t.Fatalf("active migration SQL filename is not deterministically parseable: %s", filename)
		}
		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			t.Fatalf("parse migration version %q: %v", matches[1], err)
		}
		name := matches[2]
		direction := matches[3]
		if existing, ok := byVersion[version]; ok && existing != name {
			t.Fatalf("migration version %06d is shared by %q and %q", version, existing, name)
		}
		byVersion[version] = name
		key := fmt.Sprintf("%06d_%s", version, name)
		if pairs[key] == nil {
			pairs[key] = make(map[string]string)
		}
		if existing, ok := pairs[key][direction]; ok {
			t.Fatalf("duplicate %s migration file for %s: %q and %q", direction, key, existing, filename)
		}
		pairs[key][direction] = filename
		files = append(files, migrationFile{version: version, name: name, direction: direction, filename: filename})
	}

	if len(files) == 0 {
		t.Fatal("no active migration SQL files found")
	}
	for key, directions := range pairs {
		if directions["up"] == "" || directions["down"] == "" {
			t.Fatalf("migration %s must have exactly one matching .up.sql and .down.sql", key)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].version != files[j].version {
			return files[i].version < files[j].version
		}
		if files[i].name != files[j].name {
			return files[i].name < files[j].name
		}
		return files[i].direction < files[j].direction
	})
	for i := 1; i < len(files); i++ {
		if files[i].version == files[i-1].version && files[i].name != files[i-1].name {
			t.Fatalf("migration ordering is ambiguous at version %06d: %q and %q", files[i].version, files[i-1].name, files[i].name)
		}
	}
}
