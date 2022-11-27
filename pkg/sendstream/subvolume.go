package sendstream

import (
	"path/filepath"

	"github.com/google/uuid"
)

type ReceivingSubvolume struct {
	// The path of the subvolume
	Path string
	// The UUID of the subvolume
	UUID uuid.UUID
	// The ctransid of the subvolume
	Ctransid uint64
}

func (r *ReceivingSubvolume) ResolvePath(path string) string {
	return filepath.Join(r.Path, path)
}
