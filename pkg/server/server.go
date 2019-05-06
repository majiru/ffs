package server

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"log"
	"io"
	"errors"
	
	"github.com/majiru/ffs"
	"aqwari.net/net/styx"
)

type Server struct {
	FS ffs.Fs
}

func (srv Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestedFile := r.URL.Path
	requestedFile = filepath.Join("/", filepath.FromSlash(path.Clean("/"+requestedFile)))
	file, err := srv.FS.Read(requestedFile)
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
	fi, err := srv.FS.Stat(requestedFile)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	http.ServeContent(w, r, requestedFile, fi.ModTime(), content)
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
				w, ok := file.(ffs.Writer)
				if t.Flag & os.O_TRUNC != 0 {
					if !ok {
						t.Ropen(nil, errors.New("Not Supported"))
						continue
					}
					if truncerr := w.Truncate(1); truncerr != nil {
						t.Ropen(nil, truncerr)
						continue
					}
				}
				if t.Flag & os.O_APPEND != 0 {
					//BUG: O_APPEND is never set
					if !ok {
						t.Ropen(nil, errors.New("Not Supported"))
						continue
					}
					if _, seekerr := w.Seek(-1, io.SeekEnd); seekerr != nil {
						t.Ropen(nil, seekerr)
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