package scan

import (
	"github.com/ppxb/miyabi/internal/nfo"
	"github.com/ppxb/miyabi/internal/pan"
)

// Directory describes a directory queued during BFS traversal.
type Directory struct {
	ID   string
	Path string
}

// Video wraps a 115 file with an assigned catalogue code.
type Video struct {
	pan.File
	Code   string
	Manual bool
	// Document is read from an existing sidecar, never from a metadata provider.
	Document *nfo.Movie
	// DirectoryCode records a provisional part group for transactional validation.
	DirectoryCode string
}
