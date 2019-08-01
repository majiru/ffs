package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/majiru/ffs"
	"github.com/majiru/ffs/fs/diskfs"
	"github.com/majiru/ffs/fs/mkvfs"
	"github.com/majiru/ffs/fs/domainfs"
	"github.com/majiru/ffs/fs/mediafs"
	"github.com/majiru/ffs/fs/pastefs"
	"github.com/majiru/ffs/fs/jukeboxfs"
)

type FSConf struct {
	Name string
	SubDom string
	Args []string
	fs ffs.Fs
}

type Config struct {
	ServeHTTPS bool
	Database string
	Base *FSConf
	Doms []string
	FS []*FSConf
}

func genDefaultConf(f io.WriteSeeker) error {
	webfs := &FSConf{"diskfs", "www", []string{"./www"}, nil}
	conf := Config{
		false,
		"",
		webfs,
		[]string{"localhost", "example.com"},
		[]*FSConf{
			webfs,
			&FSConf{"pastefs", "paste", []string{}, nil},
		},
	}

	json, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		return err
	}
	_, err = f.Write(json)
	if err != nil {
		return err
	}
	f.Seek(0, io.SeekStart)
	return err
}

func parseFSConf(c *FSConf) error {
	var err error

	switch c.Name {
	case "diskfs":
		c.fs = &diskfs.Diskfs{c.Args[0]}
	case "pastefs":
		c.fs = pastefs.NewPastefs()
	case "mediafs":
		if len(c.Args) > 0 {
			f, err := os.Open(c.Args[1])
			if err != nil {
				return err
			}
			c.fs, err = mediafs.NewMediafs(f)
			return err
		}
		c.fs, err = mediafs.NewMediafs(nil)
	case "mkvfs":
		c.fs = mkvfs.NewMKVfs()
	case "jukefs", "jukeboxfs":
		if len(c.Args) != 1 {
			return errors.New("parseFSConf: Not enough/Too many args to jukeboxfs")
		}
		c.fs, err = jukeboxfs.NewJukefs(c.Args[0])
	default:
		return errors.New("parseFSConf: Unknown fs")
	}
	return err
}

func readConf(confFile io.ReadSeeker) (*Config, error) {
	b, err := ioutil.ReadAll(confFile)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	if err = json.Unmarshal(b, conf); err != nil {
		return nil, err
	}

	if err = parseFSConf(conf.Base); err != nil {
		return nil, err
	}
	for i := range conf.FS {
		if err = parseFSConf(conf.FS[i]); err != nil {
			return nil, err
		}
	}
	return conf, nil
}

func conf2Domfs(conf *Config) *domainfs.Domainfs {
	domfs := domainfs.NewDomainfs()
	domfs.AddSub(conf.Base.fs, "www")
	domfs.AddDNS(conf.Base.fs, conf.Doms...)
	for _, fsc := range conf.FS {
		domfs.AddSub(fsc.fs, fsc.SubDom)
	}
	return domfs
}
