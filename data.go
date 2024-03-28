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
	// Partition Pack, [13] is pack kind (0x02 ~ 0x04)
	KeyPartitionPack = [13]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01}
	// Operational Patterns
	KeyOP = [12]byte{0x06, 0x0e, 0x2b, 0x34, 0x04, 0x01, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01}
	// Structural Metadata Sets, [5] is xx for length (0xff as placeholder)
	KeyStructural = [13]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0xff, 0x01, 0x01,
		0x0d, 0x01, 0x01, 0x01, 0x01}
	// Primer Set, [14] is Version of the Primer Pack
	KeyPrimer = [14]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01, 0x05}
	// TODO: ? from ffmpeg mxfdec
	KeyEssenceElement = [12]byte{0x06, 0x0e, 0x2b, 0x34, 0x01, 0x02, 0x01, 0x01,
		0x0d, 0x01, 0x03, 0x01}
	// Index Table Segment, [5] is xx for length (0xff as placeholder)
	KeyIndexTable = [15]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0xff, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01, 0x10, 0x01}
	// Random Index Pack
	KeyRIP = [15]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01, 0x11, 0x01}
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
	return bytes.Equal(key[:13], KeyPartitionPack[:]) && key[13] >= 2 && key[13] <= 4
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

func isStructuralMeta(key []byte) bool {
	return bytes.Equal(key[:5], KeyStructural[:5]) && bytes.Equal(key[6:13], KeyStructural[6:])
}

func whichMeta(v uint16) string {
	switch v {
	case 0x012f:
		return "SM - Preface"
	case 0x0130:
		return "SM - Identification"
	case 0x0118:
		return "SM - Content Storage"
	case 0x0123:
		return "SM - Essence Container Data"
	case 0x0136:
		return "SM - Material Package"
	case 0x0137:
		return "SM - Source Package"
	case 0x013b:
		return "SM - Timeline Track"
	case 0x010f:
		return "SM - Sequence"
	case 0x0111:
		return "SM - Source Clip"
	case 0x0114:
		return "SM - Timecode Component"
	case 0x0144:
		return "SM - Multiple Descriptor"
		// TODO: others
	default:
		return "Structural Metadata " + fmt.Sprintf("0x%04x", v)
	}
}

// MXF Local Set:  An MXF Set employing 2-byte Local Tag encoding.
type LocalSet struct {
	Tag   uint16
	Len   int
	Value []byte
}

func (l *LocalSet) View() string {
	return fmt.Sprintf("Tag: %04x, len %d, value: %x", l.Tag, l.Len, l.Value)
}

func ParseLocalSets(bs []byte) []LocalSet {
	i := 0
	n := len(bs)
	ret := make([]LocalSet, 0)
	for {
		if n-i < 4 { // 2+2
			break
		}
		l := int(binary.BigEndian.Uint16(bs[i+2 : i+4]))
		ret = append(ret, LocalSet{
			Tag:   binary.BigEndian.Uint16(bs[i : i+2]),
			Len:   l,
			Value: bs[i+4 : i+4+l],
		})

		i += (4 + l)
	}
	return ret
}

func isIndexTable(key []byte) bool {
	return bytes.Equal(key[:5], KeyIndexTable[:5]) && bytes.Equal(key[6:15], KeyIndexTable[6:])
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
		case bytes.Equal(k.Key[:14], KeyPrimer[:]):
			ds = append(ds, Dummy{
				name:  "Primer Pack",
				known: true,
			})
		case isStructuralMeta(k.Key):
			k14_15 := binary.BigEndian.Uint16(k.Key[13:15])
			ds = append(ds, Dummy{
				name:  whichMeta(k14_15),
				known: true,
			})
		case isIndexTable(k.Key):
			ds = append(ds, Dummy{
				name:  "Index Table",
				known: true,
			})
		case isEssenceElement(k.Key):
			ds = append(ds, Dummy{
				name:  "Essence Element",
				known: true,
			})
		case bytes.Equal(k.Key[:15], KeyRIP[:]):
			ds = append(ds, Dummy{
				name:  "Random Index Pack",
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
