package pastefs

import (
	"html/template"
	"io"
	"os"
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

func (fs *Pastefs) root() *fsutil.Dir {
	fi, _ := fs.newpaste.Stat()
	d, _ := fs.pastes.Stat()
	return fsutil.CreateDir("/", d, fi)
}

func (fs *Pastefs) Stat(file string) (os.FileInfo, error) {
	switch file {
	case "/":
		return fs.root().Stat()
	case "/index.html":
		return fsutil.CreateFile([]byte(""), 0644, "/index.html").Stat()
	default:
		return fs.root().Walk(file)
	}
}

func (fs *Pastefs) ReadDir(path string) (ffs.Dir, error) {
	switch path {
	case "/":
		return fs.root(), nil
	default:
		return fs.root().WalkForDir(path)
	}
}

func (fs *Pastefs) Open(file string, mode int) (interface{}, error) {
	switch file {
	case "/index.html":
		f := fsutil.CreateFile([]byte(""), 0644, "/index.html")
		err := dir2html(f, fs.pastes)
		return f, err
	case "/new":
		if mode&os.O_RDWR != 0 || mode&os.O_WRONLY != 0 || mode&os.O_TRUNC != 0 {
			name := strconv.FormatInt(time.Now().Unix(), 10)
			f := fsutil.CreateFile([]byte(""), 0777, name)
			fi, _ := f.Stat()
			fs.pastes.Append(fi)
			return f, nil
		}
		return fs.newpaste.Dup(), nil
	default:
		return fs.root().WalkForFile(file)
	}
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
