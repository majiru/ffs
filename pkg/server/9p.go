package server

import (
	"log"
	"os"

	"aqwari.net/net/styx"
	"github.com/majiru/ffs"
)

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
