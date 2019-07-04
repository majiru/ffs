package mkvfs

import (
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
		p.parent = &crumb{p.cur, nil}
		p.lastlevel = 0
		return true, nil
	}
	switch {
	case info.Level > p.lastlevel:
		if info.Level != p.lastlevel+1 {
			return false, parseError("Multi level descent")
		}
		p.parent = &crumb{p.cur, p.parent}
	case info.Level < p.lastlevel:
		for i := p.lastlevel; i != info.Level; i-- {
			if p.parent.parent == nil {
				return false, parseError("Illegal climb on " + mkvparse.NameForElementID(id))
			}
			p.parent = p.parent.parent
		}
		fallthrough
	default:
		p.cur = p.parent.dir
	}
	name, err := genunique(p.cur, mkvparse.NameForElementID(id))
	if err != nil {
		return false, err
	}
	newdir := fsutil.CreateDir(name)
	p.cur.Append(newdir.Stats)
	p.cur = newdir
	p.lastlevel = info.Level
	return true, nil
}

func (p *TreeParser) HandleMasterEnd(id mkvparse.ElementID, info mkvparse.ElementInfo) error {
	return nil
}

func formatinfo(info *mkvparse.ElementInfo) string {
	return strconv.FormatInt(info.Offset, 10) + " " + strconv.FormatInt(info.Size, 10)
}

func (p *TreeParser) write(id *mkvparse.ElementID, info *mkvparse.ElementInfo) error {
	if p.cur == nil {
		return parseError("Orphaned element")
	}
	name, err := genunique(p.cur, mkvparse.NameForElementID(*id))
	if err != nil {
		return err
	}
	p.cur.Append(fsutil.CreateFile([]byte(formatinfo(info)), 0644, name).Stats)
	return nil
}

func (p *TreeParser) HandleString(id mkvparse.ElementID, value string, info mkvparse.ElementInfo) error {
	return p.write(&id, &info)
}

func (p *TreeParser) HandleInteger(id mkvparse.ElementID, value int64, info mkvparse.ElementInfo) error {
	return p.write(&id, &info)
}

func (p *TreeParser) HandleFloat(id mkvparse.ElementID, value float64, info mkvparse.ElementInfo) error {
	return p.write(&id, &info)
}

func (p *TreeParser) HandleDate(id mkvparse.ElementID, value time.Time, info mkvparse.ElementInfo) error {
	return p.write(&id, &info)
}

func (p *TreeParser) HandleBinary(id mkvparse.ElementID, value []byte, info mkvparse.ElementInfo) error {
	return p.write(&id, &info)
}