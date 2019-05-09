package main

import (
	"log"
	"net/http"
	"os"

	"aqwari.net/net/styx"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/fs/diskfs"
	"github.com/majiru/ffs/fs/domainfs"
	"github.com/majiru/ffs/fs/pastefs"
	"github.com/majiru/ffs/pkg/server"
)

func main() {
	var styxServer styx.Server
	styxServer.TraceLog = log.New(os.Stderr, "", 0)
	styxServer.ErrorLog = log.New(os.Stderr, "", 0)

	m := make(map[string]ffs.Fs)
	m["127.0.0.1"] = &diskfs.Diskfs{"/tmp"}
	m["localhost"] = pastefs.NewPastefs()
	dfs := &domainfs.Domainfs{m}

	srv := server.Server{dfs}
	styxServer.Handler = styx.HandlerFunc(srv.Serve9P)
	styxServer.Addr = ":564"
	go http.ListenAndServe(":80", dfs)
	log.Fatal(styxServer.ListenAndServe())
}
