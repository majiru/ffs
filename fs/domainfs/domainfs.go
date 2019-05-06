package domainfs

import (
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
	"strings"
	"os"
)

func map2dir(m map[string]ffs.Fs) ffs.Dir {
	root := fsutil.CreateDir("/")

	for k, _ := range m {
		fi, _ := fsutil.CreateDir(k).Stat()
		root.Append(fi)
	}

	return root
}

type Domainfs struct {
	Domains map[string]ffs.Fs
}

func (fs *Domainfs) path2fs(path string) (ffs.Fs, string, error) {
	paths := strings.Split(path, "/")
	if(len(paths) < 2){
		return nil, "", os.ErrNotExist
	}

	child := fs.Domains[paths[1]]
	if child == nil {
		return nil, "", os.ErrNotExist
	}

	file := "/" + strings.Join(paths[2:], "/")
	return child, file, nil
}

func (fs Domainfs) Stat(path string) (os.FileInfo, error) {
	if(path == "/") {
		return map2dir(fs.Domains).Stat()
	}
	child, file, err := fs.path2fs(path)
	if err != nil {
		return nil, err
	}
	return child.Stat(file)
}

func (fs Domainfs) ReadDir(path string) (ffs.Dir, error) {
	if(path == "/") {
		return map2dir(fs.Domains), nil
	}
	child, file, err := fs.path2fs(path)
	if err != nil {
		return nil, err
	}
	return child.ReadDir(file)
}

func (fs Domainfs) Read(path string) (ffs.File, error) {
	child, file, err := fs.path2fs(path)
	if err != nil {
		return nil, err
	}
	return child.Read(file)
}