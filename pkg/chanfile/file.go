package chanfile

import (
	"os"

	"github.com/majiru/ffs/pkg/fsutil"
)

const (
	//ReqMsg
	Read = iota
	Write
	Trunc
	Close

	//RecvMsg
	Commit
	Discard
)

type ReqMsg struct {
	Type   int
	Offset int64
	Len    int64
	//Only populated on writes
	Content []byte
}

type RecvMsg struct {
	Type int
	Err  error
}

type File struct {
	Content *fsutil.File
	Req     chan ReqMsg
	Recv    chan RecvMsg
}

func CreateFile(content []byte, mode os.FileMode, name string) *File {
	f := &File{
		fsutil.CreateFile(content, mode, name),
		make(chan ReqMsg),
		make(chan RecvMsg),
	}
	f.Content.Stats.File = f
	return f
}

func WrapFile(f *fsutil.File) *File {
	chanf := &File{
		f,
		make(chan ReqMsg),
		make(chan RecvMsg),
	}
	chanf.Content.Stats.File = chanf
	return chanf
}

func (f *File) Dup() *File {
	return &File{f.Content.Dup(), f.Req, f.Recv}
}

func (f *File) Write(b []byte) (int, error) {
	f.Req <- ReqMsg{Write, f.Content.SeekPos(), int64(len(b)), b}
	m := <-f.Recv
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Type == Commit {
		return f.Content.Write(b)
	}
	return 0, nil
}

func (f *File) WriteAt(b []byte, off int64) (int, error) {
	f.Req <- ReqMsg{Write, off, int64(len(b)), b}
	m := <-f.Recv
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Type == Commit {
		return f.Content.WriteAt(b, off)
	}
	return 0, nil
}

func (f *File) Truncate(size int64) error {
	f.Req <- ReqMsg{Trunc, 0, size, nil}
	m := <-f.Recv
	if m.Err != nil {
		return m.Err
	}
	if m.Type == Commit {
		return f.Content.Truncate(size)
	}
	return nil
}

func (f *File) Read(b []byte) (int, error) {
	f.Req <- ReqMsg{Read, f.Content.SeekPos(), int64(len(b)), nil}
	m := <-f.Recv
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Type == Commit {
		return f.Content.Read(b)
	}
	return 0, nil
}

func (f *File) ReadAt(b []byte, off int64) (int, error) {
	f.Req <- ReqMsg{Read, off, int64(len(b)), nil}
	m := <-f.Recv
	if m.Err != nil {
		return 0, m.Err
	}
	if m.Type == Commit {
		return f.Content.ReadAt(b, off)
	}
	return 0, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	//Read & Write messages always specify offsets, so we don't ask for permission
	return f.Content.Seek(offset, whence)
}

func (f *File) Stat() (os.FileInfo, error) {
	return f.Content.Stats, nil
}

func (f *File) Close() error {
	f.Req <- ReqMsg{Close, 0, 0, nil}
	m := <-f.Recv
	if m.Err != nil {
		return m.Err
	}
	return f.Content.Close()
}