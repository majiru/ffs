package server

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"aqwari.net/net/styx"
	"github.com/majiru/ffs"
)

type Server struct {
	Fs ffs.Fs
}

func (srv Server) ReadHTTP(w http.ResponseWriter, r *http.Request, path string) (content ffs.File, err error) {
	file, err := srv.Fs.Open(path, os.O_RDONLY)
	content, ok := file.(ffs.File)
	if !ok || err != nil {
		log.Println("Error: " + err.Error() + " for request " + r.URL.Path)
		if err == os.ErrNotExist {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		http.Error(w, "Internal server error", 500)
		return
	}
	return
}

func (srv Server) WriteHTTP(w http.ResponseWriter, r *http.Request, path string) (writer ffs.File, err error) {
	file, err := srv.Fs.Open(path, os.O_RDWR|os.O_TRUNC)
	content, ok := file.(ffs.Writer)
	if !ok || err != nil {
		log.Println("Error: " + err.Error() + " for request " + r.URL.Path)
		if err == os.ErrNotExist {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		http.Error(w, "Internal server error", 500)
		return
	}
	if r.Body == nil {
		return
	}
	io.Copy(content, r.Body)
	//Don't expect POSTS to end in new line
	content.Write([]byte("\n"))
	return
}

func (srv Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestedFile := r.URL.Path
	requestedFile = filepath.Join("/", filepath.FromSlash(path.Clean("/"+requestedFile)))
	fi, err := srv.Fs.Stat(requestedFile)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		content, err := srv.ReadHTTP(w, r, requestedFile)
		if err == nil && content != nil {
			http.ServeContent(w, r, requestedFile, fi.ModTime(), content)
		}
	case http.MethodPost:
		content, err := srv.WriteHTTP(w, r, requestedFile)
		if err == nil && content != nil {
			http.ServeContent(w, r, requestedFile, fi.ModTime(), content)
		}
	}
	return
}

func (srv Server) Serve9P(s *styx.Session) {
	for s.Next() {
		msg := s.Request()
		fi, err := srv.Fs.Stat(msg.Path())
		if err != nil {
			log.Println(err.Error())
			msg.Rerror(os.ErrNotExist.Error())
			continue
		}
		switch t := msg.(type) {
		case styx.Twalk:
			t.Rwalk(fi, nil)
		case styx.Topen:
			if fi.IsDir() {
				t.Ropen(srv.Fs.ReadDir(msg.Path()))
			} else {
				t.Ropen(srv.Fs.Open(msg.Path(), t.Flag))
			}
		case styx.Tstat:
			t.Rstat(fi, nil)
		case styx.Ttruncate:
			if w, ok := fi.Sys().(ffs.Writer); ok {
				t.Rtruncate(w.Truncate(t.Size))
			}
		}
	}
}
