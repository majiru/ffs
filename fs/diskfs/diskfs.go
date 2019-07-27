package diskfs

import (
	"os"
	"path"

	"github.com/majiru/ffs"
)

type Diskfs struct {
	Root string
}

func (fs *Diskfs) Stat(name string) (os.FileInfo, error) {
	return os.Stat(path.Join(fs.Root, name))
}

func (fs *Diskfs) ReadDir(name string) (ffs.Dir, error) {
	return os.Open(path.Join(fs.Root, name))
}

func (fs *Diskfs) Open(name string, mode int) (ffs.File, error) {
	return os.Open(path.Join(fs.Root, name))
}
