package gomxf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var (
	KeyFillItem = [16]byte{0x06, 0x0e, 0x2b, 0x34, 0x01, 0x01, 0x01, 0x02,
		0x03, 0x01, 0x02, 0x10, 0x01, 0x00, 0x00, 0x00}
	KeyHeader = [14]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01, 0x02}
	KeyOP = [12]byte{0x06, 0x0e, 0x2b, 0x34, 0x04, 0x01, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01}
	KeySets = [13]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0xff, 0x01, 0x01, // [5] is xx for length
		0x0d, 0x01, 0x01, 0x01, 0x01}
	KeyEssenceElement = [12]byte{0x06, 0x0e, 0x2b, 0x34, 0x01, 0x02, 0x01, 0x01,
		0x0d, 0x01, 0x03, 0x01}
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
	name   string
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
	return d.name + " ..."
}

type PackMeta struct {
	KAGSize           uint32
	ThisPartition     uint64
	PreviousPartition uint64
	FooterPartition   uint64
	HeaderByteCount   uint64
	IndexByteCount    uint64
	IndexSID          uint32
	BodyOffset        uint64
	BodySID           uint32
}

type OperationalPattern struct {
	Valid bool // an empty OP would be invalid
	//
	ItemComplexity    string
	PackageComplexity string
	// qualifier, true for 1
	External      bool
	NonStreamFile bool
	MultiTrack    bool
}

func ParseOP(bs []byte) OperationalPattern {
	op := OperationalPattern{
		Valid:         true,
		External:      bs[14]&0b10 > 0,
		NonStreamFile: bs[14]&0b100 > 0,
		MultiTrack:    bs[14]&0b1000 > 0,
	}
	switch bs[12] {
	case 0x01:
		op.ItemComplexity = "Single"
	case 0x02:
		op.ItemComplexity = "Play-list"
	case 0x03:
		op.ItemComplexity = "Edit"
	}
	switch bs[13] {
	case 0x01:
		op.PackageComplexity = "Single"
	case 0x02:
		op.PackageComplexity = "Ganged"
	case 0x03:
		op.PackageComplexity = "Alternate"
	}
	return op
}

type Batch struct {
	N        int
	Len      int
	Elements [][]byte
}

func ParseBatch(data []byte) (b Batch) {
	b.N = int(binary.BigEndian.Uint32(data[:4]))
	b.Len = int(binary.BigEndian.Uint32(data[4:8]))
	offset := 8
	for i := 0; i < b.N; i++ {
		b.Elements = append(b.Elements, data[offset:offset+b.Len])
		offset += b.Len
	}
	return b
}

type Pack struct {
	Kind      string
	Closed    bool
	Completed bool
	Meta      PackMeta
	OP        OperationalPattern
	Essences  Batch
}

func (p *Pack) Known() bool {
	return true
}

func (p *Pack) IsFill() bool {
	return false
}

func (p *Pack) View() string {
	ret := fmt.Sprintf("%s: closed=%v, completed=%v, OP: %+v, nContainers: %d, meta: %+v",
		p.Kind, p.Closed, p.Completed, p.OP, p.Essences.N, p.Meta)
	for i, e := range p.Essences.Elements {
		ret += fmt.Sprintf("\nEssence#%d: %s", i, keyString(e))
	}
	return ret
}

func IsPartitionPack(key []byte) bool {
	return bytes.Equal(key[:13], KeyHeader[:13]) && key[13] >= 2 && key[13] <= 4
}

func readData(r io.ReaderAt, klv *KLV) (bs []byte, err error) {
	bs = make([]byte, klv.Length)
	_, err = r.ReadAt(bs, klv.ValueStart)
	return
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
	default:
		return &p, nil
	}
	bs, err := readData(r, klv)
	if err != nil {
		return nil, err
	}
	p.Meta.KAGSize = binary.BigEndian.Uint32(bs[4:8])
	p.Meta.ThisPartition = binary.BigEndian.Uint64(bs[8:16])
	p.Meta.PreviousPartition = binary.BigEndian.Uint64(bs[16:24])
	p.Meta.FooterPartition = binary.BigEndian.Uint64(bs[24:32])
	p.Meta.HeaderByteCount = binary.BigEndian.Uint64(bs[32:40])
	p.Meta.IndexByteCount = binary.BigEndian.Uint64(bs[40:48])
	p.Meta.IndexSID = binary.BigEndian.Uint32(bs[48:52])
	p.Meta.BodyOffset = binary.BigEndian.Uint64(bs[52:60])
	p.Meta.BodySID = binary.BigEndian.Uint32(bs[60:64])
	if bytes.Equal(KeyOP[:], bs[64:76]) {
		p.OP = ParseOP(bs[64:80])
	}
	p.Essences = ParseBatch(bs[80:])
	return &p, nil
}

func IsFillItem(key []byte) bool {
	return bytes.Equal(key, KeyFillItem[:])
}

func isEssenceElement(key []byte) bool {
	return bytes.Equal(key[:12], KeyEssenceElement[:])
}

func isMetaSets(key []byte) bool {
	return bytes.Equal(key[:5], KeySets[:5]) && bytes.Equal(key[6:13], KeySets[6:])
}

func Decode4View(r io.ReaderAt, ks KLVs) (ds []KLVData, err error) {
	// var es Batch
	for _, k := range ks {
		switch {
		case IsFillItem(k.Key):
			ds = append(ds, Dummy{
				name:   "Fill Item",
				known:  true,
				filled: true,
			})
		case IsPartitionPack(k.Key):
			pack, err := decodePack(r, k)
			if err != nil {
				return ds, err
			}
			ds = append(ds, pack)
		case isMetaSets(k.Key):
			ds = append(ds, Dummy{
				name:  "Struct Metadata Set",
				known: true,
			})
		case isEssenceElement(k.Key):
			ds = append(ds, Dummy{
				name:  "Essence Element",
				known: true,
			})
		default:
			ds = append(ds, Dummy{
				name: "Unknown KLV",
			})
		}
	}
	return
}
