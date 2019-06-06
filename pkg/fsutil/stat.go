package fsutil

import (
	"os"
	"time"
)

//Stat implements os.FileInfo
type Stat struct {
	perm os.FileMode
	name string
	time time.Time
	size int64
	File interface{}
}

func (s Stat) Name() string     { return s.name }
func (s Stat) Sys() interface{} { return s.File }

func (s Stat) ModTime() time.Time { return s.time }

func (s Stat) Mode() os.FileMode { return s.perm }

func (s Stat) IsDir() bool { return s.perm.IsDir() }

func (s Stat) Size() int64 {
	if f, ok := s.File.(interface{ Size() int64 }); ok {
		return f.Size()
	}
	return s.size
}
