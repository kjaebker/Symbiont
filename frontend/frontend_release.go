//go:build release

package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Assets returns the embedded frontend with the "dist/" prefix stripped.
// Only available in release builds (-tags release).
func Assets() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic("embedded frontend: " + err.Error())
	}
	return sub
}
