package server

import (
	"errors"
	"os"

	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

// NotFoundFs allows for tests when Open and ReadDir return os.ErrNotExist
// but Stats does not
type NotFoundFs struct{}

func (fs *NotFoundFs) Open(path string, mode int) (ffs.File, error) {
	return nil, os.ErrNotExist
}

func (fs *NotFoundFs) ReadDir(path string) (ffs.Dir, error) {
	return nil, os.ErrNotExist
}

func (fs *NotFoundFs) Stat(path string) (os.FileInfo, error) {
	return fsutil.CreateFile([]byte{}, 0644, "test").Stats, nil
}

// ErrFs alows for tests on non os.ErrNotExist errors returned by the fs
type ErrFs struct{}

var ErrBogus = errors.New("bogus test error")

func (fs *ErrFs) Open(path string, mode int) (ffs.File, error) {
	return nil, ErrBogus
}

func (fs *ErrFs) ReadDir(path string) (ffs.Dir, error) {
	return nil, ErrBogus
}

func (fs *ErrFs) Stat(path string) (os.FileInfo, error) {
	return fsutil.CreateFile([]byte{}, 0644, "test").Stats, nil
}
