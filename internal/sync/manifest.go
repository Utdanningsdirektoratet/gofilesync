package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// manifest records the destinations gofilesync itself wrote, so deletion
// handling only ever restores/removes managed files, never pre-existing ones.
type manifest struct {
	path  string
	paths map[string]bool
}

func loadManifest(path string) *manifest {
	m := &manifest{path: path, paths: make(map[string]bool)}

	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}

	var list []string
	if json.Unmarshal(data, &list) == nil {
		for _, p := range list {
			m.paths[p] = true
		}
	}
	return m
}

func (m *manifest) save() error {
	list := make([]string, 0, len(m.paths))
	for p := range m.paths {
		list = append(list, p)
	}
	sort.Strings(list)

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0644)
}
