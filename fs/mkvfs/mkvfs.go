package mkvfs

import (
	"os"
	"strings"
	"sync"

	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/chanfile"
	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/remko/go-mkvparse"
)

type MKVfs struct {
	*sync.RWMutex
	root *fsutil.Dir
	path *chanfile.File
	p *TreeParser
	mkv *os.File
}

func NewMKVfs() *MKVfs {
	m := &MKVfs{
		&sync.RWMutex{},
		nil,
		chanfile.CreateFile([]byte(""), 0644, "mkv"),
		nil,
		nil,
	}
	contents := fsutil.CreateDir("contents")
	m.root = fsutil.CreateDir("/", m.path.Content.Stats, contents.Stats)
	m.p = NewTreeParser(contents)
	go m.pathproc()
	return m
}

func (fs *MKVfs) decode(fpath string) (err error) {
	fs.Lock()
	defer fs.Unlock()
	if fs.mkv, err = os.Open(strings.TrimSuffix(fpath, "\n")); err != nil {
		return
	}
	err = mkvparse.Parse(fs.mkv, fs.p)
	fs.mkv.Close()
	return
}

func (fs *MKVfs) pathproc() {
	for {
		m := <- fs.path.Req
		switch(m.Type) {
		case chanfile.Read:
			fs.path.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Write:
			fs.path.Recv <- chanfile.RecvMsg{chanfile.Commit, fs.decode(string(m.Content))}
		case chanfile.Trunc:
			fs.path.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Close:
			fs.path.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		}
	}
}

func (fs *MKVfs) Stat(fpath string) (os.FileInfo, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch fpath {
	case "/":
		return fs.root.Stat()
	default:
		return fs.root.Walk(fpath)
	}
}

func (fs *MKVfs) ReadDir(fpath string) (ffs.Dir, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch fpath {
	case "/":
		return fs.root.Dup(), nil
	default:
		return fs.root.WalkForDir(fpath)
	}
}

func (fs *MKVfs) Open(fpath string, mode int) (interface{}, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch fpath {
	case "/mkv":
		return fs.path.Dup(), nil
	default:
		return fs.root.WalkForFile(fpath)
	}
}