package ramfs

import (
	"errors"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Ramfs struct {
	sync.RWMutex
	Root *fsutil.Dir
}

var DirExists = errors.New("File exists already as dir")
var FileExists = errors.New("Dir exists already as file")

func (r *Ramfs) FindOrCreate(path string, isDir bool) (ffs.File, ffs.Dir, error) {
	r.Lock()
	defer r.Unlock()
	dir := r.Root
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		for i, p := range parts[1 : len(parts)-1] {
			if p == "" {
				continue
			}
			if d, err := dir.Find(p); err == nil {
				if !d.IsDir() && i < len(parts)-1 {
					return nil, nil, DirExists
				}
				dir = d.Sys().(*fsutil.Dir)
			} else {
				dir.Append(fsutil.CreateDir(p).Stats)
			}
		}
	}
	fi, err := dir.Find(parts[len(parts)-1])
	if err == nil {
		if isDir {
			if !fi.IsDir() {
				return nil, nil, FileExists
			}
			return nil, fi.Sys().(*fsutil.Dir), nil
		}
		if !isDir {
			if fi.IsDir() {
				return nil, nil, DirExists
			}
			return fi.Sys().(ffs.File), nil, nil
		}
	}
	switch isDir {
	case true:
		d := fsutil.CreateDir(parts[len(parts)-1])
		dir.Append(d.Stats)
		return nil, d, nil
	default:
		f := fsutil.CreateFile([]byte{}, 0644, parts[len(parts)-1])
		dir.Append(f.Stats)
		return f, nil, nil
	}
}

func (r *Ramfs) Open(path string, mode int) (ffs.File, error) {
	f, _, err := r.FindOrCreate(path, false)
	return f, err
}

func (r *Ramfs) ReadDir(path string) (ffs.Dir, error) {
	_, d, err := r.FindOrCreate(path, true)
	return d, err
}

func (r *Ramfs) Stat(file string) (os.FileInfo, error) {
	//If we are stating something that doesn't exist we assume a file
	fi, err := r.Root.Walk(file)
	if err == nil {
		return fi, nil
	}
	return fsutil.CreateFile([]byte{}, 0644, path.Base(file)).Stats, nil
}
