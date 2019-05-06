package ffs

import (
	"io"
	"os"
)

type File interface {
	io.Reader
	io.Seeker
	io.Closer
	io.ReaderAt
	Stat() (os.FileInfo, error)
}

type Dir interface {
	Readdir(n int) ([]os.FileInfo, error)
	Reset() error
	Stat() (os.FileInfo, error)
}

type Fs interface {
	Read(path string) (File, error)
	ReadDir(path string) (Dir, error)
	Stat(path string) (os.FileInfo, error)
}