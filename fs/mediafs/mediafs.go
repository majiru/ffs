package mediafs

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"

	anidb "github.com/majiru/anidb2json"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/chanfile"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Mediafs struct {
	*sync.RWMutex
	Root     *fsutil.Dir
	DB       *anidb.TitleDB
	dbfile   *chanfile.File
	homepage *fsutil.File
}

func NewMediafs(db io.ReadWriter) (fs *Mediafs, err error) {
	fs = &Mediafs{
		&sync.RWMutex{},
		fsutil.CreateDir("/"),
		&anidb.TitleDB{},
		chanfile.CreateFile([]byte(""), 0644, "db"),
		fsutil.CreateFile([]byte(""), 0644, "index.html"),
	}
	if db != nil {
		_, err = io.Copy(fs.dbfile.Content, db)
		if err != nil {
			return
		}
		fs.dbfile.Content.Seek(0, io.SeekStart)
		err = fs.update()
	}
	go fs.dbproc()
	return
}

func (fs *Mediafs) updateTree() {
	fs.Root = fsutil.CreateDir("/")
	for _, s := range fs.DB.Anime {
		subdir := fsutil.CreateDir(s.Name)
		for _, p := range s.Path {
			subdir.Append(fsutil.CreateFile([]byte(p), 0644, path.Base(p)).Stats)
		}
		fs.Root.Append(subdir.Stats)
	}
}

func (fs *Mediafs) updateHomepage() (err error) {
	t := template.New("homepage")
	t.Funcs(template.FuncMap{
		"files": func(name string) []os.FileInfo {
			if dir, err := fs.Root.WalkForDir(name); err == nil {
				return dir.Copy()
			}
			return nil
		},
	})
	t, err = t.Parse(homepagetemplate)
	if err != nil {
		return
	}
	fs.homepage.Truncate(1)
	fs.homepage.Seek(0, io.SeekStart)
	err = t.ExecuteTemplate(fs.homepage, "homepage", fs)
	return
}

func (fs *Mediafs) updateDB() (err error) {
	b, err := ioutil.ReadAll(fs.dbfile.Content)
	if err != nil {
		return
	}
	fs.dbfile.Content.Seek(0, io.SeekStart)
	err = json.Unmarshal(b, fs.DB)
	return
}

func (fs *Mediafs) update() (err error) {
	fs.Lock()
	if err = fs.updateDB(); err != nil {
		fs.Unlock()
		return
	}
	fs.updateTree()
	err = fs.updateHomepage()
	fs.Unlock()
	return
}

func (fs *Mediafs) Check() (err error) {
	if fs.dbfile.Content.Stats.ModTime().After(fs.homepage.Stats.ModTime()) {
		return fs.update()
	}
	return
}

func (fs *Mediafs) dbproc() {
	for {
		m := <-fs.dbfile.Req
		switch m.Type {
		case chanfile.Read:
			fs.dbfile.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Write:
			fs.dbfile.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Trunc:
			fs.dbfile.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Close:
			fs.dbfile.Recv <- chanfile.RecvMsg{chanfile.Commit, fs.Check()}
		}
	}
}

func (fs *Mediafs) Stat(file string) (os.FileInfo, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch file {
	case "/index.html":
		return fs.homepage.Stat()
	case "/db":
		return fs.dbfile.Stat()
	case "/":
		return fs.Root.Stat()
	default:
		return fs.Root.Walk(file)
	}
}

func (fs *Mediafs) ReadDir(path string) (ffs.Dir, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch path {
	case "/":
		return fs.Root.Dup(), nil
	default:
		return fs.Root.WalkForDir(path)
	}
}

func (fs *Mediafs) Open(file string, mode int) (interface{}, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch file {
	case "/index.html":
		return fs.homepage.Dup(), nil
	case "/db":
		return fs.dbfile.Dup(), nil
	default:
		if f, err := fs.Root.WalkForFile(file); err != nil {
			return nil, err
		} else {
			//These files store the absolute path, not the file contents
			f.Seek(0, io.SeekStart)
			if b, err := ioutil.ReadAll(f); err != nil {
				return nil, err
			} else {
				return os.OpenFile(string(b), os.O_RDONLY, 0555)
			}
		}
	}
}
