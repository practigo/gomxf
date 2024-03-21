package gomxf

import (
	"bytes"
	"fmt"
	"io"
)

var (
	KeyFillItem = [16]byte{0x06, 0x0e, 0x2b, 0x34, 0x01, 0x01, 0x01, 0x02,
		0x03, 0x01, 0x02, 0x10, 0x01, 0x00, 0x00, 0x00}
	KeyHeader = [14]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01, 0x02}
)

const (
	HeaderPartitionPack = "HeaderPartitionPack"
	BodyPartitionPack   = "BodyPartitionPack"
	FooterPartitionPack = "FooterPartitionPack"
)

type KLVData interface {
	Known() bool
	IsFill() bool
	View() string
}

type Dummy struct {
	known  bool
	filled bool
}

func (d Dummy) Known() bool {
	return d.known
}

func (d Dummy) IsFill() bool {
	return d.filled
}

func (d Dummy) View() string {
	if !d.known {
		return "Unknown KLV ..."
	}
	if d.filled {
		return "Fill Item ..."
	}
	return ""
}

type Pack struct {
	Kind      string
	Closed    bool
	Completed bool
	// todo
	// data inside
}

func (p *Pack) Known() bool {
	return true
}

func (p *Pack) IsFill() bool {
	return false
}

func (p *Pack) View() string {
	return fmt.Sprintf("%s: closed=%v, completed=%v", p.Kind, p.Closed, p.Completed)
}

func IsPartitionPack(key []byte) bool {
	return bytes.Equal(key[:13], KeyHeader[:13]) && key[13] >= 2 && key[13] <= 4
}

func decodePack(r io.ReaderAt, klv *KLV) (*Pack, error) {
	k := klv.Key
	p := Pack{
		Closed:    k[14] == 0x02 || k[14] == 0x04,
		Completed: k[14] == 0x03 || k[14] == 0x04,
	}
	switch k[13] {
	case 0x02:
		p.Kind = HeaderPartitionPack
	case 0x03:
		p.Kind = BodyPartitionPack
	case 0x04:
		p.Kind = FooterPartitionPack
	}
	_ = r // not used yet
	return &p, nil
}

func IsFillItem(key []byte) bool {
	return bytes.Equal(key, KeyFillItem[:])
}

func Decode4View(r io.ReaderAt, ks KLVs) (ds []KLVData, err error) {
	for _, k := range ks {
		switch {
		case IsFillItem(k.Key):
			ds = append(ds, Dummy{
				known:  true,
				filled: true,
			})
		case IsPartitionPack(k.Key):
			pack, err := decodePack(r, k)
			if err != nil {
				return ds, err
			}
			ds = append(ds, pack)
		default:
			ds = append(ds, Dummy{})
		}
	}
	return
}
