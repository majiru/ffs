package mkvfs

import (
	"encoding/binary"
	"io"
	"fmt"
	"os"
	"strings"
	"strconv"

	"github.com/majiru/ffs/pkg/chanfile"
)

type Decoder struct {
	EBML *chanfile.File
	Block *chanfile.File
	f *os.File
}

func NewDecoder() *Decoder {
	d := &Decoder{
		chanfile.CreateFile([]byte{}, 0644, "EBML"),
		chanfile.CreateFile([]byte{}, 0644, "Block"),
		nil,
	}
	go d.blockproc()
	return d
}

func (d *Decoder) decodeBlock(req string) error {
	var (
		tmp []byte
		err error
	)
	parts := strings.Split(req, " ")
	if len(parts) != 2 {
		fmt.Errorf("Usage:offset length")
	}
	offset, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	//length, err := strconv.Atoi(parts[1])
	//if err != nil {
	//	return err
	//}
	
	if _, err = d.f.Seek(int64(offset), io.SeekStart); err != nil {
		return err
	}
	if err = d.Block.Content.Truncate(1); err != nil {
		return err
	}
	if _, err = d.Block.Seek(0, io.SeekStart); err != nil {
		return err
	}
	tmp = make([]byte, 1)
	if _, err = d.f.Read(tmp); err != nil {
		return err
	}
	var trackNum uint16
	switch {
	case tmp[0] & 0x80 == 0x80:
		trackNum = uint16(tmp[0] & 0x7f)
	case tmp[0] & 0x40 == 0x40:
		trackNum = uint16((tmp[0] & 0x3f) << 8)
		if _, err = d.f.Read(tmp); err != nil {
			return err
		}
		trackNum = trackNum & uint16(tmp[0])
	default:
		return fmt.Errorf("Unexpected first byte %q", tmp)
	}
	if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("Block Header:\n\tTrack number: %d\n", trackNum))); err != nil {
		return err
	}
	tmp = make([]byte, 2)
	if _, err = d.f.Read(tmp); err != nil {
		return err
	}
	relativeOffset := binary.BigEndian.Uint16(tmp)
	if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("\tRelative Offset: %d\n", relativeOffset))); err != nil {
		return err
	}
	if _, err = d.Block.Content.Write([]byte("\tFlags:\n")); err != nil {
		return err
	}
	var (
		invis bool
		lacing string
	)
	tmp = make([]byte, 1)
	if _, err = d.f.Read(tmp); err != nil {
		return err
	}
	if tmp[0] & 0x10 == 0x10 {
		invis = true
	}
	switch f := tmp[0] & 0x60; f {
	case 0:
		lacing = "No Lacing"
	case 0x20:
		lacing = "XIPH lacing"
	case 0x60:
		lacing = "EBML lacing"
	case 0x40:
		lacing = "fixed-size lacing"
	}
	_, err = d.Block.Content.Write([]byte(fmt.Sprintf("\t\tInvisable: %t\n\t\tLacing: %s\n", invis, lacing)))
	return err
}

func (d *Decoder) blockproc() {
	for {
		switch m := <- d.Block.Req; m.Type {
		case chanfile.Read:
			d.Block.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Write:
			d.Block.Recv <- chanfile.RecvMsg{chanfile.Commit, d.decodeBlock(string(m.Content))}
		case chanfile.Trunc:
			d.Block.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Close:
			d.Block.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		}
	}
}