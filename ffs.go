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
}

type Fs interface {
	Read(path string) (File, error)
	Stat(path string) (os.FileInfo, error)
}