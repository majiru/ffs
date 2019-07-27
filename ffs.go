//Package ffs provides an interface for implementing in memory file systems.
package ffs

import (
	"io"
	"os"
)

//File represenets a read only file.
//os.File satisfies this interface.
type File interface {
	io.Reader
	io.Seeker
	io.Closer
	io.ReaderAt
	Stat() (os.FileInfo, error)
}

//Writer represents a read/write file.
//Truncate is required for satisfying os.O_TRUNC open mode and 9p Rtruncate.
//os.File satisfies this interface.
type Writer interface {
	File
	io.Writer
	io.WriterAt
	Truncate(size int64) error
}

//Dir represents a directory containing zero or more files.
//os.File satisfies this interface.
type Dir interface {
	Readdir(n int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
}

//Fs represents a filesystem.
//It is safe to assume ReadDir will only be called on directory paths.
//Stat is used for both directories and files.
type Fs interface {
	Open(path string, mode int) (File, error)
	ReadDir(path string) (Dir, error)
	Stat(path string) (os.FileInfo, error)
}
