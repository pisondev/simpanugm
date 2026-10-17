package migrations

import (
	"embed"
	"io/fs"
	"strconv"
	"strings"
)

//go:embed *.sql
var FS embed.FS

func Latest() int64 {
	entries, _ := fs.ReadDir(FS, ".")
	var latest int64
	for _, e := range entries {
		prefix, _, ok := strings.Cut(e.Name(), "_")
		if !ok {
			continue
		}
		if n, err := strconv.ParseInt(prefix, 10, 64); err == nil && n > latest {
			latest = n
		}
	}
	return latest
}
