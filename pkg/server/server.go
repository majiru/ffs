package server

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"log"
	
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
	"aqwari.net/net/styx"
)

type Server struct {
	FS ffs.Fs
}

func (srv Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestedFile := r.URL.Path
	requestedFile = filepath.Join("/", filepath.FromSlash(path.Clean("/"+requestedFile)))
	file, err := srv.FS.Read(requestedFile)
	if err != nil {
		log.Println("Error: " + err.Error() + " for request " + r.URL.Path)
		if err == os.ErrNotExist {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		http.Error(w, "Internal server error", 500)
		return
	}
	fi, _ := srv.FS.Stat(requestedFile)
	http.ServeContent(w, r, requestedFile, fi.ModTime(), file)
	return
}

func (srv Server) Serve9P( s *styx.Session){
	for s.Next() {
		msg := s.Request()
		fi, err := srv.FS.Stat(msg.Path())
		if err != nil {
			msg.Rerror(os.ErrNotExist.Error())
		}
		switch t := msg.(type) {
		case styx.Twalk:
			t.Rwalk(fi, nil)
		case styx.Topen:
			if fi.IsDir() {
				files, err := srv.FS.ReadDir(msg.Path())
				t.Ropen(fsutil.CreateDir(files...), err)
			} else {
				t.Ropen(srv.FS.Read(msg.Path()))
			}
		case styx.Tstat:
			t.Rstat(fi, nil)
		}
	}
}