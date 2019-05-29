package mkvfs

import (
	"encoding/binary"
	"strconv"
	"time"

	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/remko/go-mkvparse"
)

type parseError string

func (p parseError) Error() string {
	return "mkvfs.TreeParser: " + string(p)
}

type TreeParser struct {
	Root *fsutil.Dir
	cur *fsutil.Dir
}

func NewTreeParser(root *fsutil.Dir) *TreeParser {
	if root == nil {
		root = fsutil.CreateDir("/")
	}
	return &TreeParser{root, nil}
}

func (p *TreeParser) HandleMasterBegin(id mkvparse.ElementID, info mkvparse.ElementInfo) (bool, error) {
	switch id {
	case mkvparse.CuesElement:
		fallthrough
	case mkvparse.ClusterElement:
		return false, nil
	}
	newdir := fsutil.CreateDir(mkvparse.NameForElementID(id))
	if info.Level == 0 {
		p.Root.Append(newdir.Stats)
	} else {
		p.cur.Append(newdir.Stats)
	}
	p.cur = newdir
	return true, nil
}

func (p *TreeParser) HandleMasterEnd(id mkvparse.ElementID, info mkvparse.ElementInfo) error {
	return nil
}

func (p *TreeParser) HandleString(id mkvparse.ElementID, value string, info mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	p.cur.Append(fsutil.CreateFile([]byte(value), 0644, mkvparse.NameForElementID(id)).Stats)
	return nil
}

func (p *TreeParser) HandleInteger(id mkvparse.ElementID, value int64, info mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	newdir := fsutil.CreateDir(mkvparse.NameForElementID(id))
	raw := fsutil.CreateFile([]byte(""), 0644, "raw")
	if err := binary.Write(raw, binary.BigEndian, value); err != nil {
		return err
	}
	newdir.Append(fsutil.CreateFile([]byte(strconv.FormatInt(value, 10)), 0644, "str").Stats, raw.Stats)
	p.cur.Append(newdir.Stats)
	return nil
}

func (p *TreeParser) HandleFloat(id mkvparse.ElementID, value float64, info mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	newdir := fsutil.CreateDir(mkvparse.NameForElementID(id))
	raw := fsutil.CreateFile([]byte(""), 0644, "raw")
	if err := binary.Write(raw, binary.BigEndian, value); err != nil {
		return err
	}
	newdir.Append(fsutil.CreateFile([]byte(strconv.FormatFloat(value, 'E', -1, 64)), 0644, "str").Stats, raw.Stats)
	p.cur.Append(newdir.Stats)
	return nil
}

func (p *TreeParser) HandleDate(id mkvparse.ElementID, value time.Time, info mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	p.cur.Append(fsutil.CreateFile([]byte(value.String()), 0644, mkvparse.NameForElementID(id)).Stats)
	return nil
}

func (p *TreeParser) HandleBinary(id mkvparse.ElementID, value []byte, info mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	p.cur.Append(fsutil.CreateFile(value, 0644, mkvparse.NameForElementID(id)).Stats)
	return nil
}