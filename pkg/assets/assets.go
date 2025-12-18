package assets

import (
	"net/http"
	"os"
	"path/filepath"
)

// chooseDir attempts to find the attached_assets directory from a few
// likely relative paths. This avoids using go:embed so we can serve the
// existing files on disk without moving them.
func chooseDir() (string, error) {
	candidates := []string{
		"./attached_assets",
		"../attached_assets",
		"../../attached_assets",
	}
	for _, c := range candidates {
		p := filepath.Clean(c)
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}

// Handler returns an http.Handler that serves the attached_assets files
// from disk. Caller should check the error and use the handler (e.g.
// mount it at "/assets/").
func Handler() (http.Handler, error) {
	dir, err := chooseDir()
	if err != nil {
		return nil, err
	}
	return http.StripPrefix("/assets/", http.FileServer(http.Dir(dir))), nil
}
