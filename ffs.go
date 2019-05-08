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

type Writer interface {
	File
	io.Writer
	io.WriterAt
	Truncate(size int64) error
}

type Dir interface {
	Readdir(n int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
}

type Fs interface {
	Open(path string, mode int) (interface{}, error)
	ReadDir(path string) (Dir, error)
	Stat(path string) (os.FileInfo, error)
}
