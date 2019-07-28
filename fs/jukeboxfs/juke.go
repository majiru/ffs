package jukeboxfs

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dhowden/tag"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Jukefs struct {
	*sync.RWMutex
	path string
	root *fsutil.Dir
	info map[string]tag.Metadata
}

func NewJukefs(root string) (*Jukefs, error) {
	fs := &Jukefs{
		&sync.RWMutex{},
		root,
		fsutil.CreateDir("/"),
		make(map[string]tag.Metadata),
	}
	if err := fs.updateMetadata(); err != nil {
		return nil, err
	}
	fs.updateTree()
	return fs, nil
}

type msg struct {
	t tag.Metadata
	s string
}

func (fs *Jukefs) updateMetadata() error {
	fs.Lock()
	wg := &sync.WaitGroup{}
	parseout := make(chan msg, 4)
	walkout := make(chan msg, 4)
	go func() {
		for i := range parseout {
			fs.info[i.s] = i.t
		}
	}()
	wg.Add(4)
	for i := 0; i < 4; i++ {
		go func(){
			for j := range walkout {
				f, err := os.Open(j.s)
				if err != nil {
					log.Println(err)
					continue
				}
				m, err := tag.ReadFrom(f)
				f.Close()
				if err != nil {
					log.Println(err)
					continue
				}
				j.t = m
				parseout <- j
			}
			wg.Done()
		}()
	}
	err := filepath.Walk(fs.path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && (strings.HasSuffix(info.Name(), "mp3") || strings.HasSuffix(info.Name(), "flac")) {
			walkout <- msg{s: path}
		}
		return nil
	})
	close(walkout)
	wg.Wait()
	close(parseout)
	fs.Unlock()
	return err
}

func (fs *Jukefs) updateTree() {
	fs.Lock()
	albums := make(map[string]*fsutil.Dir)
	for k, v := range fs.info {
		a := v.Album()
		a = strings.Replace(a, "/", "", -1)
		t := strings.Replace(v.Title(), "/", "", -1)
		if a == "" {
			a = "UNDEFINED"
		}
		if d, ok := albums[a]; ok {
			d.Append(fsutil.CreateFile([]byte(k), 0644, t).Stats)
		} else {
			newd := fsutil.CreateDir(a, fsutil.CreateFile([]byte(k), 0644, t).Stats)
			albums[a] = newd
			fs.root.Append(newd.Stats)
		}
	}
	fs.Unlock()
}

func (fs *Jukefs) Stat(fpath string) (os.FileInfo, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch {
	case fpath == "/":
		return fs.root.Stat()
	case strings.HasSuffix(fpath, "/index.html"):
		return fsutil.CreateFile([]byte{}, 0644, "index.html").Stats, nil
	default:
		return fs.root.Walk(fpath)
	}
}

func (fs *Jukefs) ReadDir(fpath string) (ffs.Dir, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch fpath {
	case "/":
		return fs.root.Dup(), nil
	default:
		return fs.root.WalkForDir(fpath)
	}
}

func (fs *Jukefs) Open(fpath string, mode int) (ffs.File, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch {
	case fpath == "/index.html":
		f := fsutil.CreateFile([]byte{}, 0644, "index.html")
		return f.Dup(), dir2html(f, fs.root.Copy())
	case strings.HasSuffix(fpath, "/index.html"):
		fpath = strings.TrimSuffix(fpath, "/index.html")
		d, err := fs.root.WalkForDir(fpath)
		if err != nil {
			return nil, err
		}
		f := fsutil.CreateFile([]byte{}, 0644, "index.html")
		return f.Dup(), dir2html(f, d.Copy())
	default:
		f, err := fs.root.WalkForFile(fpath)
		if err != nil {
			return nil, err
		}
		//These files store the path, not the file contents
		f.Seek(0, io.SeekStart)
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		return os.OpenFile(string(b), os.O_RDONLY, 0555)
	}
}

func dir2html(f ffs.Writer, fi []os.FileInfo) error {
	t := template.New("page")
	t, err := t.Parse(homepage)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(f, "page", fi)
}

const homepage = `
<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<title>Jukefs</title>
	</head>
	<body>
		{{ range . }}
		{{ if .IsDir }}
		<a href="{{.Name}}/index.html">{{.Name}}</a>
		{{ else }}
		<a href="{{.Name}}">{{.Name}}</a>
		{{ end }}
		<br>
		{{ end }}
	</body>
</html>
`