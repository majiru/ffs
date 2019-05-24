package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"aqwari.net/net/styx"
	"github.com/majiru/ffs/fs/diskfs"
	"github.com/majiru/ffs/fs/domainfs"
	"github.com/majiru/ffs/fs/mediafs"
	"github.com/majiru/ffs/fs/pastefs"
	"github.com/majiru/ffs/pkg/server"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage %s: http_port 9p_port [domains...]\n", os.Args[0])
		os.Exit(1)
	}
	porthttp := ":" + os.Args[1]
	port9p := ":" + os.Args[2]
	os.Args = os.Args[3:]

	var styxServer styx.Server
	styxServer.TraceLog = log.New(os.Stderr, "", 0)
	styxServer.ErrorLog = log.New(os.Stderr, "", 0)

	mfs, err := mediafs.NewMediafs(nil)
	if err != nil {
		log.Fatal("Could not create mfs:", err)
	}
	dfs := &diskfs.Diskfs{"./www"}
	pfs := pastefs.NewPastefs()
	domfs := domainfs.NewDomainfs()
	domfs.AddSub(dfs, "www")
	if len(os.Args) > 0 {
		domfs.AddDNS(dfs, os.Args...)
	} else {
		domfs.AddDNS(dfs, "localhost", "example.com")
	}
	domfs.AddSub(pfs, "paste")
	domfs.AddSub(mfs, "media")

	srv := server.Server{domfs}
	styxServer.Handler = styx.HandlerFunc(srv.Serve9P)
	styxServer.Addr = port9p
	go http.ListenAndServe(porthttp, domfs)
	log.Fatal(styxServer.ListenAndServe())
}
