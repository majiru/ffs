package pastefs

import (
	"html/template"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Pastefs struct {
	newpaste *fsutil.File
	pastes   *fsutil.Dir
}

func dir2html(w io.Writer, dir *fsutil.Dir) (err error) {
	content := struct{ Files []os.FileInfo }{dir.Copy()}
	t := template.New("homepage")
	t, err = t.Parse(homepage)
	if err != nil {
		return
	}
	err = t.ExecuteTemplate(w, "homepage", content)
	return
}

func (fs *Pastefs) root() ffs.Dir {
	fi, _ := fs.newpaste.Stat()
	d, _ := fs.pastes.Stat()
	return fsutil.CreateDir("/", d, fi)
}

func (fs *Pastefs) Stat(file string) (os.FileInfo, error) {
	if file == "/" {
		return fs.root().Stat()
	}
	if file == "/new" || file == "/index.html" {
		return fs.newpaste.Stat()
	}
	if file == "/pastes" {
		return fs.pastes.Stat()
	}

	return fs.pastes.Find(path.Base(file))
}

func (fs *Pastefs) ReadDir(path string) (ffs.Dir, error) {
	if path == "/" {
		return fs.root(), nil
	} else if path == "/pastes" {
		return fs.pastes.Dup(), nil
	}

	return nil, os.ErrNotExist
}

func (fs *Pastefs) Open(file string, mode int) (interface{}, error) {
	if file == "/index.html" {
		f := fsutil.CreateFile([]byte(""), 0644, "/index.html")
		err := dir2html(f, fs.pastes)
		return f, err
	}
	if file == "/new" {
		if mode&os.O_RDWR != 0 || mode&os.O_WRONLY != 0 || mode&os.O_TRUNC != 0 {
			name := strconv.FormatInt(time.Now().Unix(), 10)
			f := fsutil.CreateFile([]byte(""), 0777, name)
			fi, _ := f.Stat()
			fs.pastes.Append(fi)
			return f, nil
		}
		return fs.newpaste.Dup(), nil
	}
	if f, err := fs.pastes.Find(path.Base(file)); err == nil {
		return f.Sys(), nil
	}
	return nil, os.ErrNotExist
}

func NewPastefs() *Pastefs {
	return &Pastefs{fsutil.CreateFile([]byte("Write to this file to paste\n"), 0777, "new"), fsutil.CreateDir("pastes")}
}

const homepage = `
<!DOCTYPE HTML>
<head>
	<title>PasteFS</title>
</head>
<body>
	<h1>Paste FS</h1><br><br>
	<p>Recent Pastes:</p><br>
	{{ range .Files }}
	<a href="/pastes/{{ .Name }}">{{ .Name }}</a><br>
	{{end}}
</body>
`
