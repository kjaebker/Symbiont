//go:build !release

package frontend

import "io/fs"

// Assets returns nil in development builds.
// The API server falls back to SYMBIONT_FRONTEND_PATH in this case.
func Assets() fs.FS {
	return nil
}
