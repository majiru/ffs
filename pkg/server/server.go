package server

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"log"
	"errors"
	"io"
	
	"github.com/majiru/ffs"
	"aqwari.net/net/styx"
)

type Server struct {
	FS ffs.Fs
}

func (srv Server) ReadHTTP(w http.ResponseWriter, r *http.Request, path string) (content ffs.File, err error){
	file, err := srv.FS.Read(path)
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

func (srv Server) WriteHTTP(w http.ResponseWriter, r *http.Request, path string) (writer ffs.File, err error){
	file, err := srv.FS.Read(path)
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
	//POSTs truncate for now, this should probably be changed
	if err = content.Truncate(1); err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	content.Seek(0, io.SeekStart)
	io.Copy(content, r.Body)
	//Don't expect POSTS to end in new line
	content.Write([]byte("\n"))
	return
}

func (srv Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestedFile := r.URL.Path
	requestedFile = filepath.Join("/", filepath.FromSlash(path.Clean("/"+requestedFile)))
	fi, err := srv.FS.Stat(requestedFile)
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

func (srv Server) Serve9P( s *styx.Session){
	for s.Next() {
		msg := s.Request()
		fi, err := srv.FS.Stat(msg.Path())
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
				files, e := srv.FS.ReadDir(msg.Path())
				t.Ropen(files, e)
			} else {
				file, err := srv.FS.Read(msg.Path())
				if t.Flag & os.O_TRUNC != 0 {
					w, ok := file.(ffs.Writer)
					if !ok {
						t.Ropen(nil, errors.New("Not Supported"))
						continue
					}
					if truncerr := w.Truncate(1); truncerr != nil {
						t.Ropen(nil, truncerr)
						continue
					}
				}
				t.Ropen(file, err)
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