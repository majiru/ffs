package mkvfs

import (
	"encoding/binary"
	"fmt"
	"strings"
	"strconv"
	"time"
	"os"

	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/remko/go-mkvparse"
)

type parseError string

func (p parseError) Error() string {
	return "mkvfs.TreeParser: " + string(p)
}

type crumb struct {
	dir *fsutil.Dir
	parent *crumb
}

type TreeParser struct {
	Root *fsutil.Dir
	cur *fsutil.Dir
	parent *crumb
	lastlevel int
}

func NewTreeParser(root *fsutil.Dir) *TreeParser {
	if root == nil {
		root = fsutil.CreateDir("/")
	}
	return &TreeParser{root, nil, &crumb{nil, nil}, 0}
}

//We need each filename to be unique, appending an incrementing
//numeric is a lazy way to do it
func getunique(d *fsutil.Dir, name, cur string) (string, error) {
	var tosearch string
	if cur != "" {
		tosearch = cur
	} else {
		tosearch = name
	}
	fi, err := d.Find(tosearch)
	if err == os.ErrNotExist {
		return tosearch, nil
	}
	tail := strings.TrimPrefix(fi.Name(), name)
	if len(tail) == 0 {
		return getunique(d, name, fmt.Sprintf("%s%d", name, 2))
	}
	if i, err := strconv.Atoi(tail); err == nil {
		i++
		return getunique(d, name, fmt.Sprintf("%s%d", name, i))
	} else {
		return "", err
	}
}

func genunique(d *fsutil.Dir, name string) (string, error) {
	return getunique(d, name, "")
}

func (p *TreeParser) HandleMasterBegin(id mkvparse.ElementID, info mkvparse.ElementInfo) (bool, error) {
	if info.Level == 0 {
		newdir := fsutil.CreateDir(mkvparse.NameForElementID(id))
		p.Root.Append(newdir.Stats)
		p.cur = newdir
	}
	if info.Level > p.lastlevel {
		p.parent = &crumb{p.cur, p.parent}
		name, err := genunique(p.cur, mkvparse.NameForElementID(id))
		if err != nil {
			return false, err
		}
		newdir := fsutil.CreateDir(name)
		p.cur.Append(newdir.Stats)
		p.cur = newdir
	} else if info.Level < p.lastlevel {
		for i := p.lastlevel; i != info.Level; i-- {
			if p.parent.parent == nil {
				return false, parseError("Illegal climb")
			}
			p.cur = p.parent.dir
			p.parent = p.parent.parent
		}
	}
	p.lastlevel = info.Level
	return true, nil
}

func (p *TreeParser) HandleMasterEnd(id mkvparse.ElementID, info mkvparse.ElementInfo) error {
	return nil
}

func (p *TreeParser) HandleString(id mkvparse.ElementID, value string, info mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	name, err := genunique(p.cur, mkvparse.NameForElementID(id))
	if err != nil {
		return err
	}
	p.cur.Append(fsutil.CreateFile([]byte(value), 0644, name).Stats)
	return nil
}

func (p *TreeParser) HandleInteger(id mkvparse.ElementID, value int64, info mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	name, err := genunique(p.cur, mkvparse.NameForElementID(id))
	if err != nil {
		return err
	}
	newdir := fsutil.CreateDir(name)
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
	name, err := genunique(p.cur, mkvparse.NameForElementID(id))
	if err != nil {
		return err
	}
	newdir := fsutil.CreateDir(name)
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
	name, err := genunique(p.cur, mkvparse.NameForElementID(id))
	if err != nil {
		return err
	}
	p.cur.Append(fsutil.CreateFile([]byte(value.String()), 0644, name).Stats)
	return nil
}

func (p *TreeParser) HandleBinary(id mkvparse.ElementID, value []byte, info mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	name, err := genunique(p.cur, mkvparse.NameForElementID(id))
	if err != nil {
		return err
	}
	p.cur.Append(fsutil.CreateFile(value, 0644, name).Stats)
	return nil
}