package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"aqwari.net/net/styx"
	"github.com/majiru/ffs/pkg/server"
	"github.com/majiru/aitm"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Printf("Usage %s: http_port https_port 9p_port config_file\n", os.Args[0])
		os.Exit(1)
	}
	porthttp := ":" + os.Args[1]
	porthttps := ":" + os.Args[2]
	port9p := ":" + os.Args[3]

	var styxServer styx.Server
	styxServer.TraceLog = log.New(os.Stderr, "", 0)
	styxServer.ErrorLog = log.New(os.Stderr, "", 0)

	var f *os.File
	var err error
	if f, err = os.Open(os.Args[4]); err != nil {
		log.Println(os.Args[4], "not found, creating default one")
		f, err = os.Create(os.Args[4])
		if err != nil {
			log.Fatal(err)
		}
		if err = genDefaultConf(f); err != nil {
			log.Fatal(err)
		}
	}

	conf, err := readConf(f)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	domfs := conf2Domfs(conf)
	srv := server.Server{domfs}
	styxServer.Handler = styx.HandlerFunc(srv.Serve9P)
	styxServer.Addr = port9p

	if conf.Database != "" {
		f, err = os.Open(conf.Database)
		if err != nil {
			log.Fatal(err)
		}
	}

	switch {
	case conf.ServeHTTPS && conf.Database != "":
		srv := aitm.WrapServer(domfs.HTTPSServer(porthttps, porthttp))
		if err := srv.LoadUsers(f); err != nil {
			log.Fatal(err)
		}
		go srv.ListenAndServeTLS("", "")
	case conf.ServeHTTPS && conf.Database == "":
		srv := domfs.HTTPSServer(porthttps, porthttp)
		go srv.ListenAndServeTLS("", "")
	case !conf.ServeHTTPS && conf.Database != "":
		srv := aitm.WrapServer(&http.Server{Addr: porthttp, Handler: domfs})
		if err := srv.LoadUsers(f); err != nil {
			log.Fatal(err)
		}
		go srv.ListenAndServe()
	case !conf.ServeHTTPS && conf.Database == "":
		srv := &http.Server{Addr: porthttp, Handler: domfs}
		go srv.ListenAndServe()
	}

	log.Fatal(styxServer.ListenAndServe())
}
