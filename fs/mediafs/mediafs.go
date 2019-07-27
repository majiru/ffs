package mediafs

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
	"strings"

	anidb "github.com/majiru/anidb2json"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/chanfile"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Mediafs struct {
	*sync.RWMutex
	Root       *fsutil.Dir
	DB         *anidb.TitleDB
	dbfile     *chanfile.File
	searchfile *chanfile.File
	homepage   *fsutil.File
	Tags       *fsutil.Dir
	Staff      *fsutil.Dir
}

func NewMediafs(db io.ReadWriter) (fs *Mediafs, err error) {
	fs = &Mediafs{
		&sync.RWMutex{},
		nil,
		&anidb.TitleDB{},
		chanfile.CreateFile([]byte(""), 0644, "db"),
		chanfile.CreateFile([]byte(""), 0644, "search"),
		fsutil.CreateFile([]byte(""), 0644, "index.html"),
		nil,
		nil,
	}
	if db != nil {
		_, err = io.Copy(fs.dbfile.Content, db)
		if err != nil {
			return
		}
		fs.dbfile.Content.Seek(0, io.SeekStart)
		err = fs.update()
	}
	fs.updateTree()
	go fs.dbproc()
	go fs.searchproc()
	return
}

func (fs *Mediafs) updateTree() {
	shows := fsutil.CreateDir("shows")
	fs.Tags = fsutil.CreateDir("tags")
	fs.Staff = fsutil.CreateDir("staff")
	fs.Root = fsutil.CreateDir("/", shows.Stats, fs.Tags.Stats, fs.Staff.Stats)
	for _, s := range fs.DB.Anime {
		subdir := fsutil.CreateDir(s.Name)
		for _, p := range s.Path {
			subdir.Append(fsutil.CreateFile([]byte(p), 0644, path.Base(p)).Stats)
		}
		shows.Append(subdir.Stats)
		for _, t := range s.Tags {
			switch _, err := fs.Tags.WalkForDir(t.Name); err {
			case os.ErrNotExist:
				fs.Tags.Append(fsutil.CreateDir(t.Name, subdir.Stats).Stats)
			case nil:
				fi, err := fs.Tags.Walk(t.Name)
				if err != nil {
					log.Fatal("Could not find", t.Name, err)
				}
				if d, ok := fi.Sys().(*fsutil.Dir); ok {
					d.Append(subdir.Stats)
				}
			}
		}
		for _, c := range s.Creators {
			switch _, err := fs.Staff.WalkForDir(c.Name); err {
			case os.ErrNotExist:
				fs.Staff.Append(fsutil.CreateDir(c.Name, subdir.Stats).Stats)
			case nil:
				fi, err := fs.Staff.Walk(c.Name)
				if err != nil {
					log.Fatal("Could not find", c.Name, err)
				}
				if d, ok := fi.Sys().(*fsutil.Dir); ok {
					d.Append(subdir.Stats)
				}
			}
		}
	}
}

func (fs *Mediafs) updateDB() error {
	fs.dbfile.Seek(0, io.SeekStart)
	b, err := ioutil.ReadAll(fs.dbfile.Content)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, fs.DB)
}

func (fs *Mediafs) update() (err error) {
	fs.Lock()
	if err = fs.updateDB(); err != nil {
		fs.Unlock()
		return
	}
	fs.updateTree()
	err = fs.genwindow(fs.homepage, fs.DB.Anime, 0)
	fs.Unlock()
	return
}

func (fs *Mediafs) Check() (error) {
	if fs.dbfile.Content.Stats.ModTime().After(fs.homepage.Stats.ModTime()) {
		return fs.update()
	}
	return nil
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

func (fs *Mediafs) search(name string) []*anidb.Anime {
	var result []*anidb.Anime
	name = strings.ToLower(name)
	fs.RLock()
	for _, series := range fs.DB.Anime {
		if strings.Contains(strings.ToLower(series.Name), name) {
			result = append(result, series)
		}
	}
	fs.RUnlock()
	return result
}

func (fs *Mediafs) searchproc() {
	for {
		m := <-fs.searchfile.Req
		switch m.Type {
		case chanfile.Read:
			fs.searchfile.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Write:
			parts := strings.Split(string(m.Content), "=")
			if len(parts) != 2 {
				fs.searchfile.Recv <- chanfile.RecvMsg{chanfile.Discard, nil}
				continue
			}
			err := fs.genpage(fs.searchfile.Content, fs.search(parts[1]))
			fs.searchfile.Content.Seek(0, io.SeekStart)
			fs.searchfile.Recv <- chanfile.RecvMsg{chanfile.Discard, err}
		case chanfile.Trunc:
			fs.searchfile.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Close:
			fs.searchfile.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		}
	}
}

func (fs *Mediafs) dir2slice(f *fsutil.Dir) []*anidb.Anime {
	var result []*anidb.Anime
	lookup := make(map[string]*anidb.Anime)
	for _, show := range fs.DB.Anime {
		lookup[show.Name] = show
	}
	if show, ok := lookup[f.Stats.Name()]; ok {
		return []*anidb.Anime{show}
	}
	for _, fi := range f.Copy() {
		if show, ok := lookup[fi.Name()]; ok {
			result = append(result, show)
		}
	}
	return result
}

func (fs *Mediafs) Stat(file string) (os.FileInfo, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch {
	//Return stubs for html only sections of the site
	case strings.HasPrefix(file, "/page"), strings.HasPrefix(file, "/bookmark"):
		_, reqfile := path.Split(file)
		return fsutil.CreateFile([]byte(""), 0644, reqfile).Stat()
	case file == "/index.html":
		return fs.homepage.Stat()
	case file == "/db":
		return fs.dbfile.Stat()
	case file == "/search":
		return fs.searchfile.Stat()
	case file == "/":
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

func (fs *Mediafs) createPageFromDir(dir string) (*fsutil.File, error) {
	d, err := fs.Root.WalkForDir(dir)
	if err != nil {
		return nil, err
	}
	f := fsutil.CreateFile([]byte(""), 0644, "index.html")
	err = fs.genpage(f, fs.dir2slice(d))
	f.Seek(0, io.SeekStart)
	return f, nil
}

func (fs *Mediafs) Open(file string, mode int) (ffs.File, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch {
	case strings.HasPrefix(file, "/page"):
		return fs.handlePagination(file, fs.DB.Anime)
	case strings.HasPrefix(file, "/bookmark"):
		return fs.createPageFromDir(strings.Replace(file, "/bookmark", "/shows", 1))
	case strings.HasPrefix(file, "/tags"), strings.HasPrefix(file, "/staff"):
		return fs.createPageFromDir(file)
	case file == "/index.html":
		return fs.homepage.Dup(), nil
	case file == "/db":
		return fs.dbfile.Dup(), nil
	case file == "/search":
		return fs.searchfile.Dup(), nil
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
